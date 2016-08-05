package kubernetes

import (
	kapi "k8s.io/kubernetes/pkg/api"
	kcache "k8s.io/kubernetes/pkg/client/cache"
	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	kframework "k8s.io/kubernetes/pkg/controller/framework"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/util/wait"
	"k8s.io/kubernetes/pkg/watch"

	"fmt"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/yangzhares/kube2consul/pkg/types"
	"github.com/yangzhares/kube2consul/pkg/consul"
)

const (
	resyncPeriod      = 5 * time.Minute
	kubernetesSvcName = "kubernetes"
)

// Kube2Consul is to work with Consul when a headless service created
type Kube2Consul struct {
	consul              *consul.Consul
	client              clientset.Interface
	endpointsStore      kcache.Store
	servicesStore       kcache.Store
	endpointsController *kframework.Controller
	serviceController   *kframework.Controller
}

// NewKube2Consul creates Kube2Consul
func NewKube2Consul(client clientset.Interface, consul *consul.Consul) (*Kube2Consul, error) {
	k2c := &Kube2Consul{
		client: client,
		consul: consul,
	}

	k2c.setEndpointsStore()
	k2c.setServicesStore()

	return k2c, nil
}

// Start starts Kube2Consul to work with consul
func (k2c *Kube2Consul) Start() {
	go k2c.endpointsController.Run(wait.NeverStop)
	go k2c.serviceController.Run(wait.NeverStop)
	k2c.waitForKubernetesService()
}

func (k2c *Kube2Consul) setServicesStore() {
	k2c.servicesStore, k2c.serviceController = kframework.NewInformer(
		&kcache.ListWatch{
			ListFunc: func(options kapi.ListOptions) (runtime.Object, error) {
				return k2c.client.Core().Services(kapi.NamespaceAll).List(options)
			},
			WatchFunc: func(options kapi.ListOptions) (watch.Interface, error) {
				return k2c.client.Core().Services(kapi.NamespaceAll).Watch(options)
			},
		},
		&kapi.Service{},
		resyncPeriod,
		kframework.ResourceEventHandlerFuncs{
			AddFunc:    k2c.newService,
			DeleteFunc: k2c.removeService,
			UpdateFunc: k2c.updateService,
		},
	)
}

func (k2c *Kube2Consul) setEndpointsStore() {
	k2c.endpointsStore, k2c.endpointsController = kframework.NewInformer(
		&kcache.ListWatch{
			ListFunc: func(options kapi.ListOptions) (runtime.Object, error) {
				return k2c.client.Core().Endpoints(kapi.NamespaceAll).List(options)
			},
			WatchFunc: func(options kapi.ListOptions) (watch.Interface, error) {
				return k2c.client.Core().Endpoints(kapi.NamespaceAll).Watch(options)
			},
		},
		&kapi.Endpoints{},
		resyncPeriod,
		kframework.ResourceEventHandlerFuncs{
			AddFunc: k2c.handleEndpointAdd,
			UpdateFunc: func(oldObj, newObj interface{}) {
				k2c.handleEndpointAdd(newObj)
			},
		},
	)
}

func assertIsService(obj interface{}) (*kapi.Service, bool) {
	service, ok := obj.(*kapi.Service)
    if ok {
		return service, ok
	}
	glog.Errorf("Type assertion failed! Expected 'Service', got %T.", service)
	return nil, ok
}

func (k2c *Kube2Consul) newService(obj interface{}) {
	if svc, ok := assertIsService(obj); ok {
		glog.Infof("Add/Updated for service %q", svc.Name)
		if !kapi.IsServiceIPSet(svc) {
			glog.Info("start to register service into Consul.")
			k2c.newHeadlessService(svc)
		} else {
			glog.Errorf("service %q in namespace %q is none headless service, skip it.", svc.Name, svc.Namespace)
		}
	}
}

func (k2c *Kube2Consul) removeService(obj interface{}) {
	if svc, ok := assertIsService(obj); ok {
		glog.Infof("remove for service %q", svc.Name)
		k2c.deregisterHeadlessService(svc)
	}
}

func (k2c *Kube2Consul) deregisterHeadlessService(svc *kapi.Service) {
	svcName := generateServiceName(svc.Name, svc.Namespace)

	services, err := k2c.consul.Service(svcName, "")
	if err != nil {
		glog.Errorf("faild to query service %q in namespace %q from consul: %v", svcName, svc.Namespace, err)
		return
	}

	if len(services) == 0 {
		glog.Infof("no service %q in namespace %q from consul", svcName, svc.Namespace)
		return
	}

	for _, service := range services {
		k2c.consul.Deregister(service)
	}
}

func (k2c *Kube2Consul) updateService(oldObj, newObj interface{}) {
	k2c.newService(newObj)
}

func (k2c *Kube2Consul) handleEndpointAdd(obj interface{}) {
	if eps, ok := obj.(*kapi.Endpoints); ok {
		glog.Infof("Add/Update for endpoints %q.", eps.Name)
		k2c.registerHeadlessService(eps)
	}
}

func (k2c *Kube2Consul) newHeadlessService(svc *kapi.Service) {
	key, err := kcache.MetaNamespaceKeyFunc(svc)
	if err != nil {
		glog.Errorf("MetaNamespaceKeyFunc gets key error: %v.", err)
		return
	}

	obj, exist, err := k2c.endpointsStore.GetByKey(key)
	if err != nil {
		glog.Errorf("faild to get endpoints from endpointsStore: %v.", err)
		return
	}

	if !exist {
		glog.Infof("could not find endpoints for service %q in namespace %q, will be registered once endpoints show up.", svc.Name, svc.Namespace)
		return
	}

	if eps, ok := obj.(*kapi.Endpoints); ok {
		k2c.registerHeadlessService(eps)
	} else {
		glog.Errorf("a none endpoints object in endpoints store: %v.", obj)
	}
}

func (k2c *Kube2Consul) registerHeadlessService(eps *kapi.Endpoints) {
	svc, err := k2c.getServiceFromEndpoints(eps)
	if err != nil {
		glog.Errorf("getServiceFromEndpoints gets endpoints by service error: %v.", err)
		return
	}

	if svc == nil {
		glog.Error("getServiceFromEndpoints returns nil service object.")
		return
	}

	if kapi.IsServiceIPSet(svc) {
		glog.Errorf("service %q in namespace %q is none headless service, skip it.", svc.Name, svc.Namespace)
		return
	}

	svcName := generateServiceName(svc.Name, svc.Namespace)
	services := generateServices(svcName, eps)
	for _, service := range services {
		k2c.consul.Register(service)
	}
	k2c.removeDeletedServices(svcName, services)
}

func generateServiceName(name string, namespace string) string {
	return name + "." + namespace
}

func generateServices(name string, eps *kapi.Endpoints) []*types.Service {
	result := []*types.Service{}

	for _, ep := range eps.Subsets {
		for _, addr := range ep.Addresses {
			var svc types.Service

			svc.Name = name
			svc.Node = addr.TargetRef.Name
			svc.Address = addr.IP
			svc.ID = addr.TargetRef.Name + ":" + addr.IP
			for _, port := range ep.Ports {
				svc.Port = int(port.Port)
				protocol := strings.ToLower(string(port.Protocol))

				if port.Protocol != "" && protocol == "udp" {
					svc.Tags = append(svc.Tags, protocol)
					svc.Tags = append(svc.Tags, port.Name)
				}
			}
			result = append(result, &svc)
		}
	}
	return result
}

func (k2c *Kube2Consul) removeDeletedServices(name string, services []*types.Service) {
	svcs, err := k2c.consul.Service(name, "")
	if err != nil {
		glog.Errorf("faild to query service %q from consul, error: %v", name, err)
		return
	}

	for _, svc := range svcs {
		if !isServiceExist(svc.Node, svc.Address, svc.Port, services) {
			k2c.consul.Deregister(svc)
		}
	}
}

func isServiceExist(node, address string, port int, services []*types.Service) bool {
	for _, service := range services {
		if node == service.Node && address == service.Address && port == service.Port {
			return true
		}
	}
	return false
}

func (k2c *Kube2Consul) getServiceFromEndpoints(eps *kapi.Endpoints) (*kapi.Service, error) {
	key, err := kcache.MetaNamespaceKeyFunc(eps)
	if err != nil {
		return nil, err
	}

	obj, exist, err := k2c.servicesStore.GetByKey(key)
	if err != nil {
		return nil, fmt.Errorf("faild to get service from service store: %v", err)
	}

	if !exist {
		glog.Infof("can't find service for endpoint %s in namespace %s.", eps.Name, eps.Namespace)
		return nil, nil
	}

	if svc, ok := assertIsService(obj); ok {
		return svc, nil
	}

	return nil, fmt.Errorf("a none service object in service store: %v", obj)
}

func (k2c *Kube2Consul) waitForKubernetesService() (svc *kapi.Service) {
	name := fmt.Sprintf("%v/%v", kapi.NamespaceDefault, kubernetesSvcName)
	glog.Infof("Waiting for service: %v", name)
	var err error
	servicePollInterval := 1 * time.Second
	for {
		svc, err = k2c.client.Core().Services(kapi.NamespaceDefault).Get(kubernetesSvcName)
		if err != nil || svc == nil {
			glog.Infof("Ignoring error while waiting for service %v: %v. Sleeping %v before retrying.", name, err, servicePollInterval)
			time.Sleep(servicePollInterval)
			continue
		}
		break
	}
	return
}
