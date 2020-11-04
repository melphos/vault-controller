# vault-controller
A K8s controller to manage Hashicorp Vault configuration using CRDs.

![VaultController](https://gitlab.com/laredoute/infra/k8s/vault-controller/-/wikis/uploads/5f2c3b62727862cf869bc9d34efd4c86/VaultController.png)

## Features:
* Policy
* Roles
* SysAuth

## Deploy
```
kubectl apply -f config/pre-deploy.yaml
kubectl apply -f config/deploy.yaml
```
This deploy will configure all objects on Kubernetes:

### Pre Deploy
  1. Create namespace: `vault-controller-system`
  2. Create a PodSecurityPolicy
  3. Create a Role and RoleBinding

### Deploy
  1. Create CustomResourceDefinition:
     1. Policies
     2. Roles
     3. SysAuth
  2. Create Role and RoleBinding for **leader-election**
  3. Create a ClusterRole and ClusterRoleBinding to the manager service
  4. Create a Service
  5. Create a Deployment of vault-controller-manager

# Create a Vault TOKEN to configure vault-controller-manager to connect to Vault API

`vault token create -policy=admin-integration`

### Configuration
To enable the controller to talk to vault API, create a configmap.
```
apiVersion: v1
kind: ConfigMap
metadata:
  name: config
  namespace: vault-controller-system
data:
  address: https://vault.security.svc.cluster.local:8200
  token: <Token generated on last step>
```

### Policy
```
apiVersion: vault.redoute.io/v1
kind: Policy
metadata:
  name: policy-sample
  namespace: vault-controller-system
spec:
  name: testpolicy
  rules: |
    path "secret/data/*" {
      capabilities = ["list"]
    }
```

### Role for LDAP Groups
```
apiVersion: vault.redoute.io/v1
kind: Role
metadata:
  name: policy-sample-ldap
  namespace: vault-controller-system
spec:
  name: dosi_ops_int
  type: ldap
  policy: admin-integration-operator
```

### Role for Kubernetes serviceAccount
```
apiVersion: vault.redoute.io/v1
kind: Role
metadata:
  name: policy-sample-kube-svc
  namespace: vault-controller-system
spec:
  name: app-1
  serviceAccount: app1 
  type: kubernetes
  namespace: finance
  policy: app1-dev-policy
```

### SysAuth
```
apiVersion: vault.redoute.io/v1
kind: SysAuth
metadata:
  name: sysauth-sample
  namespace: vault-controller-system
spec:
  path: "testapprole"
  description: "testing"
  type: "approle"
```


# To deploy all defaultRoles 

`kubectl --namespace vault-controller-system apply -f ./config/policies/ `

`kubectl --namespace vault-controller-system apply -f ./config/roles/ `

# Architecture Documentation

This Vault Controller use the kubebuilder to create an operator template. And use the Vault API to create a interface with Vault.

The main architecture is:
  - Listen to the CRD (Policy, Roles, SysAuth) from apiVersion: vault.redoute.io
  - Open the connection with Vault bring the information from configMap.
  - Understand the type of request (Reconcile, Create, Delete), process the CRD and send data to Vault API.