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

package v1

import (
	"fmt"

	"github.com/mitchellh/hashstructure"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	//RoleFinalizer name of the sysauth finalizer
	RoleFinalizer = "sysauth.finalizers.vault.redoute.io"
	//RoleWatchNamespace name of the namespace on which the controller is operating
	RoleWatchNamespace = "vault-controller-system"
	//RoleFailedState state when failed
	RoleFailedState = "failed"
	//RoleCreatedState state when created
	RoleCreatedState = "created"
	//RoleUpdatedState state when updated
	roleUpdatedState = "updated"
)

// RoleSpec defines the desired state of Policy
type RoleSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	//Name is the role name
	Name string `json:"name,omitempty"`
	//Rules defines the vault role rules
	Policy         []string `json:"policy,omitempty"`
	ServiceAccount string   `json:"serviceAccount,omitempty"`
	Type           string   `json:"type,omitempty"`
	Namespace      string   `json:"namespace,omitempty"`
}

// PolicyStatus defines the observed state of Policy
type RoleStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	State string `json:"state,omitempty"`
	Hash  string `json:"hash,omitempty"`
}

// +kubebuilder:object:root=true

// Policy is the Schema for the policies API
type Role struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   *RoleSpec   `json:"spec,omitempty"`
	Status *RoleStatus `json:"status,omitempty"`
}

// IsBeingDeleted returns true if a deletion timestamp is set
func (r *Role) IsBeingDeleted() bool {
	return !r.ObjectMeta.DeletionTimestamp.IsZero()
}

// IsCreated returns true if a sysauth config has been created
func (r *Role) IsCreated() bool {
	if r.Status == nil {
		return false
	}
	return true
}

// HasFinalizer returns true if item has a finalizer with input name
func (r *Role) HasFinalizer(name string) bool {
	return containsString(r.ObjectMeta.Finalizers, name)
}

// AddFinalizer adds the input finalizer
func (r *Role) AddFinalizer(name string) {
	r.ObjectMeta.Finalizers = append(r.ObjectMeta.Finalizers, name)
}

// RemoveFinalizer removes the input finalizer
func (r *Role) RemoveFinalizer(name string) {
	r.ObjectMeta.Finalizers = removeString(r.ObjectMeta.Finalizers, name)
}

// GetHash returns a hash of the struct
func (r *Role) GetHash() (string, error) {
	hash, err := hashstructure.Hash(r.Spec.ServiceAccount, nil)
	return fmt.Sprintf("%d", hash), err
}

// +kubebuilder:object:root=true

// RoleList contains a list of Role
type RoleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Role `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Role{}, &RoleList{})
}
