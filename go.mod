module zeebe-operator

go 1.15

require (
	contrib.go.opencensus.io/exporter/prometheus v0.2.0 // indirect
	contrib.go.opencensus.io/exporter/stackdriver v0.13.5 // indirect
	github.com/go-logr/logr v0.1.0
	github.com/google/uuid v1.1.2
	github.com/mattbaird/jsonpatch v0.0.0-20171005235357-81af80346b1a // indirect
	github.com/onsi/ginkgo v1.11.0
	github.com/onsi/gomega v1.7.0
	github.com/salaboy/camunda-cloud-go-client v0.0.7
	github.com/tektoncd/pipeline v0.9.0
	github.com/zeebe-io/zeebe/clients/go v0.0.0-20200618134307-1a98d6f027dc
	go.opencensus.io v0.22.6 // indirect
	google.golang.org/api v0.40.0 // indirect
	istio.io/client-go v1.9.0 // indirect
	k8s.io/api v0.18.1
	k8s.io/apimachinery v0.18.1
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	knative.dev/pkg v0.0.0-20191127211322-cac31abb7f36
	sigs.k8s.io/controller-runtime v0.2.2
)
