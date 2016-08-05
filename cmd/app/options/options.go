package options

import (
	"fmt"
	"net/url"
	"os"

	"github.com/spf13/pflag"

	"github.com/golang/glog"
	"k8s.io/kubernetes/pkg/api/unversioned"

	"github.com/yangzhares/kube2consul/pkg/consul"
	k2c "github.com/yangzhares/kube2consul/pkg/kubernetes"
	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	"k8s.io/kubernetes/pkg/client/restclient"
	kclientcmd "k8s.io/kubernetes/pkg/client/unversioned/clientcmd"
)

// Kube2ConsulConfig is kube2Consul configuration
type Kube2ConsulConfig struct {
	KubeMasterURL  string
	KubeConfigFile string
	ConsulConfig   *consul.Config
}

// NewKube2ConsulConfig create default config
func NewKube2ConsulConfig() *Kube2ConsulConfig {
	return &Kube2ConsulConfig{
		KubeMasterURL:  "",
		KubeConfigFile: "",
		ConsulConfig:   consul.NewConfig(),
	}
}

// APP indicates a kube2consul app
type APP struct {
	k2c *k2c.Kube2Consul
}

// NewAPP creates a APP
func NewAPP(config *Kube2ConsulConfig) *APP {
	client, err := newKubeClient(config)
	if err != nil {
		glog.Fatalf("Failed to create a kubernetes client: %v", err)
	}

	consul, err := consul.New(config.ConsulConfig)
	if err != nil {
		glog.Fatalf("Failed to create a consul client: %v", err)
	}

	kube2consul, _ := k2c.NewKube2Consul(client, consul)
	
    app := &APP {
        k2c : kube2consul,
    }

	return app
}

// Start starts kube2consul
func (app *APP) Start() {
	app.k2c.Start()
}

func newKubeClient(conf *Kube2ConsulConfig) (clientset.Interface, error) {
	var (
		config *restclient.Config
		err    error
	)

	if conf.KubeMasterURL != "" && conf.KubeConfigFile == "" {
		// Only --kubemaster was provided.
		config = &restclient.Config{
			Host:          conf.KubeMasterURL,
			ContentConfig: restclient.ContentConfig{GroupVersion: &unversioned.GroupVersion{Version: "v1"}},
		}
	} else {
		// We either have:
		//  1) --kubemaster and --kubeconfig
		//  2) just --kubeconfig
		//  3) neither flag
		// In any case, the logic is the same.  If (3), this will automatically
		// fall back on the service account token.
		overrides := &kclientcmd.ConfigOverrides{}
		overrides.ClusterInfo.Server = conf.KubeMasterURL                                // might be "", but that is OK
		rules := &kclientcmd.ClientConfigLoadingRules{ExplicitPath: conf.KubeConfigFile} // might be "", but that is OK
		if config, err = kclientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides).ClientConfig(); err != nil {
			return nil, err
		}
	}

	glog.Infof("Using %s for kubernetes master", config.Host)
	glog.Infof("Using kubernetes API %v", config.GroupVersion)
	return clientset.NewForConfig(config)
}

type kubeMasterURLVar struct {
	val *string
}

func (m kubeMasterURLVar) Set(v string) error {
	parsedURL, err := url.Parse(os.ExpandEnv(v))
	if err != nil {
		return fmt.Errorf("failed to parse kube-master url")
	}
	if parsedURL.Scheme == "" || parsedURL.Host == "" || parsedURL.Host == ":" {
		return fmt.Errorf("invalid kube-master url specified")
	}
	*m.val = v
	return nil
}

func (m kubeMasterURLVar) String() string {
	return *m.val
}

func (m kubeMasterURLVar) Type() string {
	return "string"
}

// AddFlags sets config
func (k *Kube2ConsulConfig) AddFlags(flags *pflag.FlagSet) {
    k.ConsulConfig.AddFlags(flags)
	flags.Var(kubeMasterURLVar{&k.KubeMasterURL}, "kube-master", "URL to reach kubernetes master, Env variables in this flag will be expanded(default: 127.0.0.1:8080).")
	flags.StringVar(&k.KubeConfigFile, "kube-config", k.KubeConfigFile, "Path to a kubeconfig file for access to kubernetes master service(default: not set).")
}
