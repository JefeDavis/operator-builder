module gitlab.eng.vmware.com/landerr/operator-builder

go 1.15

require (
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/afero v1.2.2
	github.com/spf13/cobra v1.1.3
	github.com/spf13/pflag v1.0.5
	gitlab.eng.vmware.com/landerr/k8s-object-code-generator v0.0.0-20210512212731-c0af185a493f
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/apimachinery v0.22.0-alpha.0
	sigs.k8s.io/kubebuilder/v3 v3.0.0
	sigs.k8s.io/yaml v1.2.0
)
