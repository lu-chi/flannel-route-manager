package google

import (
	"fmt"
	"strings"

	"code.google.com/p/goauth2/compute/serviceaccount"
	"code.google.com/p/google-api-go-client/compute/v1"
)

var metadataEndpoint = "http://169.254.169.254/computeMetadata/v1"

var replacer = strings.NewReplacer(".", "-", "/", "-")

type GoogleRouterManager struct {
	computeService *compute.Service
	network        *compute.Network
	project        string
}

func New() (*GoogleRouterManager, error) {
	client, err := serviceaccount.NewClient(&serviceaccount.Options{})
	if err != nil {
		return nil, err
	}
	computeService, err := compute.New(client)
	if err != nil {
		return nil, err
	}
	networkName, err := networkFromMetadata()
	if err != nil {
		return nil, err
	}
	project, err := projectFromMetadata()
	if err != nil {
		return nil, err
	}
	network, err := computeService.Networks.Get(project, networkName).Do()
	if err != nil {
		return nil, err
	}
	rm := &GoogleRouterManager{
		computeService: computeService,
		network:        network,
		project:        project,
	}
	return rm, nil
}

func (rm GoogleRouterManager) Delete(subnet string) error {
	return rm.delete(formatRouteName(rm.network.Name, subnet))
}

func (rm GoogleRouterManager) DeleteAllRoutes() error {
	var lastError error
	rs, err := rm.routes()
	if err != nil {
		return err
	}
	for _, r := range rs {
		if err := rm.delete(r.Name); err != nil {
			lastError = err
		}
	}
	return lastError
}

func (rm GoogleRouterManager) Insert(ip, subnet string) error {
	return rm.insert(ip, subnet)
}

func (rm GoogleRouterManager) Sync(routes map[string]string) error {
	return rm.sync(routes)
}

func (rm GoogleRouterManager) delete(name string) error {
	_, err := rm.computeService.Routes.Delete(rm.project, name).Do()
	return err
}

func (rm GoogleRouterManager) insert(ip, subnet string) error {
	name := formatRouteName(rm.network.Name, subnet)
	route := &compute.Route{
		Name:      name,
		DestRange: subnet,
		Network:   rm.network.SelfLink,
		NextHopIp: ip,
		Priority:  1000,
		Tags:      []string{},
	}
	_, err := rm.computeService.Routes.Insert(rm.project, route).Do()
	return err
}

func (rm GoogleRouterManager) sync(in map[string]string) error {
	existing := make(map[string]bool)
	routemap, err := rm.routemap()
	if err != nil {
		return err
	}
	for _, route := range routemap {
		subnet, ok := in[route.NextHopIp]
		if !ok || subnet != route.DestRange {
			if err := rm.delete(route.Name); err != nil {
				return err
			}
			continue
		}
		existing[route.NextHopIp] = true
	}
	for ip, subnet := range in {
		if !existing[ip] {
			if err := rm.insert(ip, subnet); err != nil {
				return err
			}
		}
	}
	return nil
}

func (rm GoogleRouterManager) routemap() (map[string]*compute.Route, error) {
	m := make(map[string]*compute.Route)
	routes, err := rm.routes()
	if err != nil {
		return nil, err
	}
	for _, r := range routes {
		m[r.Name] = r
	}
	return m, nil
}

func (rm GoogleRouterManager) routes() ([]*compute.Route, error) {
	rs := make([]*compute.Route, 0)
	filter := fmt.Sprintf("name eq flannel-%s-.*", rm.network.Name)
	routeList, err := rm.computeService.Routes.List(rm.project).Filter(filter).Do()
	if err != nil {
		return nil, err
	}
	for {
		for _, r := range routeList.Items {
			rs = append(rs, r)
		}
		if routeList.NextPageToken == "" {
			break
		}
		routeList, err = rm.computeService.Routes.List(rm.project).PageToken(routeList.NextPageToken).Do()
		if err != nil {
			return nil, err
		}
	}
	return rs, nil
}

func formatRouteName(network, subnet string) string {
	return fmt.Sprintf("flannel-%s-%s", network, replacer.Replace(subnet))
}
