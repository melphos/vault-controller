module redoute.io/api/vault/vault-controller

go 1.15

require (
	github.com/go-logr/logr v0.1.0
	github.com/hashicorp/vault/api v1.0.4
	github.com/mitchellh/hashstructure v1.0.0
	github.com/onsi/ginkgo v1.11.0
	github.com/onsi/gomega v1.8.1
	github.com/prometheus/client_golang v1.0.0
	k8s.io/api v0.17.2
	k8s.io/apimachinery v0.17.2
	k8s.io/client-go v0.17.2
	sigs.k8s.io/controller-runtime v0.5.0
)

replace redoute.io/api/vault/vault-controller => ./
