package consul

import (
	"net/http"
	"time"

	api "github.com/hashicorp/consul/api"
	"github.com/hashicorp/go-cleanhttp"
    "strings"
    "fmt"

    "github.com/spf13/pflag"
)

// Config is used to Consul Config
type Config struct {
	address string

	auth       basicAuth

	enableSSL bool
	sslVerify bool
	certFile   string
	keyFile    string
	caCertFile string

	token   string
	timeout int

    //Datacenter string
}

//Consul is to create Consul Catalog
type Consul struct {
	Catalog *api.Catalog
}

// New creates Consul catalog
func New(conf *Config) (*Consul, error) {
	c := &Consul{}

	client, err := newClient(conf)
	if err != nil {
		return nil, err
	}

	c.Catalog = client.Catalog()
	return c, nil
}

// New creates a Consul client
func newClient(conf *Config) (*api.Client, error) {
	config := api.DefaultConfig()
	config.HttpClient = http.DefaultClient
	config.Address = conf.address
	//config.Datacenter = conf.Datacenter
	config.Scheme = "http"

	if conf.auth.Enabled {
		config.HttpAuth = &api.HttpBasicAuth{
			Username: conf.auth.Username,
			Password: conf.auth.Password,
		}
	}

	if conf.enableSSL {
		tlsConfigDesc := &api.TLSConfig{
			Address:            conf.address,
			CAFile:             conf.caCertFile,
			CertFile:           conf.certFile,
			KeyFile:            conf.keyFile,
			InsecureSkipVerify: false,
		}

		if !conf.sslVerify {
			tlsConfigDesc.InsecureSkipVerify = true
		}

		tlsConfig, err := api.SetupTLSConfig(tlsConfigDesc)
		if err != nil {
			return nil, err
		}

		config.Scheme = "https"
		transport := cleanhttp.DefaultPooledTransport()
		transport.TLSClientConfig = tlsConfig
		config.HttpClient.Transport = transport
	}

	config.WaitTime = time.Duration(conf.timeout)
	config.Token = conf.token

	client, err := api.NewClient(config)

	return client, err
}


type basicAuth struct {
	Enabled  bool
	Username string
	Password string
}

// AuthVar implements the Flag.Value interface and allows the user to specify
// authentication in the username[:password] form.
type authVar basicAuth

func (a *authVar) Set(value string) error {
	a.Enabled = true

	if strings.Contains(value, ":") {
		split := strings.SplitN(value, ":", 2)
		a.Username = split[0]
		a.Password = split[1]
	} else {
		a.Username = value
	}

	return nil
}

func (a *authVar) String() string {
	if a.Password == "" {
		return a.Username
	}

	return fmt.Sprintf("%s:%s", a.Username, a.Password)
}


func (a *authVar) Type() string {
    return "string"
}

// NewConfig creates default consul config
func NewConfig() *Config {
    return &Config {
        address : "127.0.0.1:8500",
        enableSSL : false,
        sslVerify : true,
        timeout : 0,
    }
}

// AddFlags configure Consul config from command argument
func (c *Config) AddFlags(fs *pflag.FlagSet) {
    fs.StringVar(&c.address, "consul-api", c.address, "Address for  access to consul api service(default: 127.0.0.1:8500).")
    fs.Var((*authVar)(&c.auth), "consul-auth", "HTTP basic authentication username(and optional password), separated by a colon(default: not set).")
	fs.BoolVar(&c.enableSSL, "consul-ssl", c.enableSSL, "Enable SSL when access consul api service(default: false).")
    fs.BoolVar(&c.sslVerify, "consul-ssl-verify", c.sslVerify, "Enable SSL verfiy when access consul api service(default: true).")
    fs.StringVar(&c.caCertFile, "consul-ca", c.caCertFile, "Path to a certificate file for the certificate authority(default: not set).")
    fs.StringVar(&c.certFile, "consul-cert", c.certFile, "Path to a client cert file for TLS(default: not set).")
    fs.StringVar(&c.keyFile, "consul-key", c.keyFile, "Path to a client key file for TLS(default: not set).")
    fs.StringVar(&c.token, "consul-token", c.token, "Consul ACL token used to access consul api(default: not set).")
    fs.IntVar(&c.timeout, "consul-timeout", c.timeout, "Set a timeout(in seconds) to access consul api(default: 0).")
}