package consul

import (
	api "github.com/hashicorp/consul/api"
	"github.com/yangzhares/kube2consul/pkg/types"
    "github.com/golang/glog"
)

// Register registers a service
func (c *Consul) Register(s *types.Service){
	agentService := &api.AgentService{
		ID:      s.ID,
		Service: s.Name,
		Tags:    s.Tags,
		Port:    s.Port,
		Address: s.Address,
	}

	catalogRegistration := &api.CatalogRegistration{
		Node:    s.Node,
		Address: s.Address,
		Service: agentService,
	}

	_, err := c.Catalog.Register(catalogRegistration, nil)
	if err != nil {
		glog.Errorf("faild to register service/endpoint { %q, %q, %q }, error: %v", s.Name, s.Node, s.Address, err)
	} else {
        glog.Infof("success to register service/endpoint { %q, %q, %q }", s.Name, s.Node, s.Address)
    }
}

// Deregister deregisters a service
func (c *Consul) Deregister(s *types.Service) {
	catalogDeregistration := &api.CatalogDeregistration{
		Node:      s.Node,
		Address:   s.Address,
		ServiceID: s.ID,
	}

	_, err := c.Catalog.Deregister(catalogDeregistration, nil)
	if err != nil {
		glog.Errorf("faild to deregister service/endpoint { %q, %q, %q } error: %v", s.Name, s.Node, s.Address, err)
	} else {
        glog.Infof("success to deregister service/endpoint { %q, %q, %q }", s.Name, s.Node, s.Address)
    }
}

// Service return the service with name and tag
func (c *Consul) Service(name, tag string) ([]*types.Service, error) {
	result := []*types.Service{}
	services, _, err := c.Catalog.Service(name, tag, nil)
	if err != nil {
		return result, err
	}

	for _, service := range services {
		svc := &types.Service{
			Node:    service.Node,
			Address: service.Address,
			ID:      service.ServiceID,
			Name:    service.ServiceName,
			Tags:    service.ServiceTags,
			Port:    service.ServicePort,
		}
		result = append(result, svc)
	}

	return result, nil
}
