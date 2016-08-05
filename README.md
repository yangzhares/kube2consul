# Kube2Consul
Kube2Consul is a tool which registers Kubernetes headless services to Consul for service discovery with Consul. By default, Kubernetes uses Kube-proxy for service discovery on each node, sometimes if don’t need or want load-balancing and a single service IP provided by Kube-proxy, you can create “headless” services by specifying "None" for the cluster IP, and do service discovery with your any means, like HAProxy, Nignx, LVS or others. Kube2Consul can help you on this.

# Build Kube2Consul
Firstly you need git a copy of Kube2Consul to your desktop or local enviroment.

```
git clone https://github.com/yangzhares/kube2consul.git
```

Change to `$GOPATH/github.com/yangzhares/kube2consul/cmd`, then start to compile.

```
go build -o kube2consul
```
For more information about how to use kube2consul, please check its help usage.
```
$ ./kube2consul --help
Usage of ./kube2consul:
      --alsologtostderr[=false]: log to standard error as well as files
      --consul-api="127.0.0.1:8500": Address for  access to consul api service(default: 127.0.0.1:8500).
      --consul-auth="": HTTP basic authentication username(and optional password), separated by a colon(default: not set).
      --consul-ca="": Path to a certificate file for the certificate authority(default: not set).
      --consul-cert="": Path to a client cert file for TLS(default: not set).
      --consul-key="": Path to a client key file for TLS(default: not set).
      --consul-ssl[=false]: Enable SSL when access consul api service(default: false).
      --consul-ssl-verify[=true]: Enable SSL verfiy when access consul api service(default: true).
      --consul-timeout=0: Set a timeout(in seconds) to access consul api(default: 0).
      --consul-token="": Consul ACL token used to access consul api(default: not set).
      --kube-config="": Path to a kubeconfig file for access to kubernetes master service(default: not set).
      --kube-master="": URL to reach kubernetes master, Env variables in this flag will be expanded(default: 127.0.0.1:8080).
      --log-backtrace-at=:0: when logging hits line file:N, emit a stack trace
      --log-dir="": If non-empty, write log files in this directory
      --log-flush-frequency=5s: Maximum number of seconds between log flushes
      --logtostderr[=true]: log to standard error instead of files
      --stderrthreshold=2: logs at or above this threshold go to stderr
      --v=0: log level for V logs
      --vmodule=: comma-separated list of pattern=N settings for file-filtered logging
```
