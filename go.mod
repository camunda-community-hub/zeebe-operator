module zeebe-operator

go 1.15

require (
	github.com/go-logr/logr v0.1.0
	github.com/google/uuid v1.1.2
	github.com/salaboy/camunda-cloud-go-client v0.0.7
	github.com/tektoncd/pipeline v0.20.0
	github.com/zeebe-io/zeebe/clients/go v0.0.0-20200618134307-1a98d6f027dc
	github.com/Azure/go-autorest v11.1.2+incompatible // indirect
	github.com/coreos/bbolt v1.3.1-coreos.6 // indirect
	github.com/coreos/etcd v3.3.13+incompatible // indirect
	github.com/natefinch/lumberjack v2.0.0+incompatible // indirect
	github.com/onsi/ginkgo v1.11.0
	github.com/onsi/gomega v1.8.1
	github.com/vmware-labs/reconciler-runtime v0.1.0
	golang.org/x/text v0.3.2 // indirect
	gopkg.in/yaml.v1 v1.0.0-20140924161607-9f9df34309c0 // indirect
	k8s.io/api v0.17.4
	k8s.io/apimachinery v0.17.4
	k8s.io/client-go v0.17.4
	sigs.k8s.io/controller-runtime v0.5.2
	sigs.k8s.io/controller-tools v0.2.1 // indirect
	sigs.k8s.io/testing_frameworks v0.1.1 // indirect
)
