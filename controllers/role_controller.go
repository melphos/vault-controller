/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"encoding/json"
	defaulterror "errors"
	"fmt"

	"github.com/go-logr/logr"
	vaultapi "github.com/hashicorp/vault/api"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	apiv1 "redoute.io/api/vault/vault-controller/api/v1"
	vaultv1 "redoute.io/api/vault/vault-controller/api/v1"
)

// RoleReconciler reconciles a Role object
type RoleReconciler struct {
	client.Client
	Log       logr.Logger
	Scheme    *runtime.Scheme
	APIClient *vaultapi.Client
	Recorder  record.EventRecorder
}

// +kubebuilder:rbac:groups=vault.redoute.io,resources=policies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=vault.redoute.io,resources=policies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=events,verbs=create

func (r *RoleReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("role", req.NamespacedName)

	role := &apiv1.Role{}
	log.Info(fmt.Sprintf("starting reconcile loop for role %v", req.NamespacedName))
	defer log.Info(fmt.Sprintf("completed reconcile loop for role %v", req.NamespacedName))
	err := r.Get(ctx, req.NamespacedName, role)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	// Initializing vault config
	config, err := r.getConfig()
	if err != nil {
		r.Recorder.Event(role, corev1.EventTypeWarning, "failed", fmt.Sprintf("failed to get vault config: %s", err))
		return ctrl.Result{}, nil
	}
	if config != nil {
		address := config.Data["address"]
		token := config.Data["token"]
		r.APIClient, err = GetClient(address, token)
	}
	if err != nil {
		r.Recorder.Event(role, corev1.EventTypeWarning, "failed", fmt.Sprintf("failed to init vault client: %s", err))
		return ctrl.Result{}, nil
	}

	if role.IsBeingDeleted() {
		log.Info("run finalizer")
		err := r.handleFinalizer(role)
		if err != nil {
			r.Recorder.Event(role, corev1.EventTypeWarning, "failed", fmt.Sprintf("failed to delete finalizer: %s", err))
			return ctrl.Result{}, fmt.Errorf("error when handling finalizer: %v", err)
		}
		r.Recorder.Event(role, corev1.EventTypeNormal, "deleted", "object finalizer is deleted")
		return ctrl.Result{}, nil
	}

	isUptoDate, err := r.IsUptoDate(role)
	if err != nil {
		r.Recorder.Event(role, corev1.EventTypeWarning, "failed", fmt.Sprintf("failed to check role object up to date: %s", err))
		return ctrl.Result{}, fmt.Errorf("error when checking role IsUptoDate: %v", err)
	}

	if !role.IsCreated() || !isUptoDate {
		r.Log.Info(fmt.Sprintf("creating/updating role %v", role.Spec.Name))
		if err := r.put(role); err != nil {
			if !role.IsCreated() {
				r.Recorder.Event(role, corev1.EventTypeWarning, "failed", fmt.Sprintf("failed to create role object: %s", err))
			}
			r.Recorder.Event(role, corev1.EventTypeWarning, "failed", fmt.Sprintf("failed to update role object: %s", err))
			return ctrl.Result{}, fmt.Errorf("error when creating role: %v", err)
		}

		if !role.HasFinalizer(apiv1.RoleFinalizer) {
			r.Log.Info(fmt.Sprintf("add finalizer for role %v", req.NamespacedName))
			if err := r.addFinalizer(role); err != nil {
				r.Recorder.Event(role, corev1.EventTypeWarning, "failed", fmt.Sprintf("failed to add finalizer to role: %s", err))
				return ctrl.Result{}, fmt.Errorf("error when adding finalizer to role: %v", err)
			}
			r.Recorder.Event(role, corev1.EventTypeNormal, "added", "object finalizer is added")
		}
		if !role.IsCreated() {
			r.Recorder.Event(role, corev1.EventTypeNormal, "created", "role is created")
		}
		r.Recorder.Event(role, corev1.EventTypeNormal, "updated", "role is updated")
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

func (r *RoleReconciler) getConfig() (*corev1.ConfigMap, error) {
	config := &corev1.ConfigMap{}
	err := r.Client.Get(
		context.TODO(),
		types.NamespacedName{
			Name:      "config",
			Namespace: apiv1.WatchNamespace,
		},
		config)
	if err != nil {
		return nil, err
	}
	return config, nil
}

func (r *RoleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&vaultv1.Role{}).
		Complete(r)
}

func (r *RoleReconciler) delete(p *apiv1.Role) error {
	role := &apiv1.Role{}
	r.Log.Info(fmt.Sprintf("deleting role %s", p.GetName()))
	if p.Status == nil {
		return nil
	}

	var vaultURIRequest string

	switch p.Spec.Type {
	case "kubernetes":
		vaultURIRequest = "/v1/auth/kubernetes/role/" + p.Spec.Name
	case "ldap":
		vaultURIRequest = "/v1/auth/ldap/groups/" + p.Spec.Name
	default:
		errMsg := "DEBUG: You need specify the Type of role you want to create. Ex.: Kubernetes / ldap"
		roleSpecTypeError := defaulterror.New(errMsg)
		r.Recorder.Event(p, corev1.EventTypeWarning, "failed", fmt.Sprintf(errMsg))
		return roleSpecTypeError
	}

	roleDelete := r.APIClient.NewRequest("DELETE", vaultURIRequest)
	resp, err := r.APIClient.RawRequest(roleDelete)
	if err != nil {
		r.Recorder.Event(role, corev1.EventTypeWarning, "failed", fmt.Sprintf("failed to create a request on Vault API: %s", err))
	}
	defer resp.Body.Close()
	return err
}

func (r *RoleReconciler) put(p *apiv1.Role) error {
	// Debug role CRD request
	r.Log.Info(fmt.Sprintf("DEBUG: CRD:\nName: %s\nServiceAccount: %s\nNameSpace: %s\nPolicie: %s\nType: %s", p.Spec.Name, p.Spec.ServiceAccount, p.Spec.Namespace, p.Spec.Policy, p.Spec.Type))

	var body map[string]interface{}
	var vaultURIRequest string

	if p.Spec.Type == "kubernetes" {
		r.Log.Info(fmt.Sprintf("DEBUG: Role type: KUBERNETES"))
		vaultURIRequest = "/v1/auth/kubernetes/role/" + p.Spec.Name
		body = make(map[string]interface{})
		body["bound_service_account_names"] = p.Spec.ServiceAccount
		body["bound_service_account_namespaces"] = p.Spec.Namespace
		body["policies"] = p.Spec.Policy
	}
	if p.Spec.Type == "ldap" {
		r.Log.Info(fmt.Sprintf("DEBUG: Role type: LDAP"))
		vaultURIRequest = "/v1/auth/ldap/groups/" + p.Spec.Name
		body = make(map[string]interface{})
		body["policies"] = p.Spec.Policy
	}

	// Marshal the map into a JSON string.
	bodyData, err := json.Marshal(body)
	if err != nil {
		fmt.Println()
		r.Log.Info(fmt.Sprintf(err.Error()))
		return err
	}
	jsonStr := string(bodyData)
	r.Log.Info(fmt.Sprintf("DEBUG: JSON Body to sent to Vault API auth:\n Role: %s:", jsonStr))

	rolePut := r.APIClient.NewRequest("POST", vaultURIRequest)
	if err := rolePut.SetJSONBody(body); err != nil {
		r.Log.Info(fmt.Sprintf("failed to set JSONBody to a request on Vault API: %s", err))
	}
	resp, err := r.APIClient.RawRequest(rolePut)
	if err != nil {
		r.Recorder.Event(p, corev1.EventTypeWarning, "failed", fmt.Sprintf("failed to create a request on Vault API: %s", err))
	}
	defer resp.Body.Close()
	r.Log.Info(fmt.Sprintf("Response form Vault to create role: %s", resp))

	if err != nil {
		return err
	}
	hash, err := p.GetHash()
	if err != nil {
		return err
	}
	p.Status = &apiv1.RoleStatus{
		Hash:  hash,
		State: apiv1.RoleCreatedState,
	}
	err = r.Update(context.Background(), p)
	if err != nil {
		return err
	}
	return nil
}

// IsUptoDate returns true if a sysauth config is current
func (p *RoleReconciler) IsUptoDate(s *apiv1.Role) (bool, error) {
	hash, err := s.GetHash()
	if err != nil {
		return false, fmt.Errorf("error when calculating role hash: %v", err)
	}
	if s.Status == nil {
		return false, nil
	}
	if s.Status.Hash != hash {
		return false, nil
	}
	return true, nil
}
