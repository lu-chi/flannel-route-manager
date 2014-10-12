// Copyright (c) 2014 Kelsey Hightower. All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0
// that can be found in the LICENSE file.
package google

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"strings"

	"code.google.com/p/goauth2/compute/serviceaccount"
	"code.google.com/p/google-api-go-client/compute/v1"
)

var metadataEndpoint = "http://169.254.169.254/computeMetadata/v1"

var replacer = strings.NewReplacer(".", "-", "/", "-")

type GoogleRouterManager struct {
	computeService *compute.Service
	network        string
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
	network, err := networkFromMetadata()
	if err != nil {
		return nil, err
	}
	project, err := projectFromMetadata()
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

func (rm GoogleRouterManager) Sync(routeTable map[string]string) error {
	network, err := rm.computeService.Networks.Get(rm.project, rm.network).Do()
	if err != nil {
		return err
	}
	routeMap, err := rm.routeMap()
	if err != nil {
		return err
	}
	if err := rm.deleteStaleRoutes(routeTable, routeMap); err != nil {
		return err
	}
	tags := make([]string, 0)
	for ip, subnet := range routeTable {
		name := formatRouteName(rm.network, subnet)
		route := &compute.Route{
			Name:      name,
			DestRange: subnet,
			Network:   network.SelfLink,
			NextHopIp: ip,
			Priority:  1000,
			Tags:      tags,
		}
		_, err = rm.computeService.Routes.Insert(rm.project, route).Do()
		if err != nil {
			return err
		}
	}
	return nil
}

func (rm GoogleRouterManager) deleteStaleRoutes(routeTable map[string]string, routeMap map[string]*compute.Route) error {
	for _, route := range routeMap {
		subnet, ok := routeTable[route.NextHopIp]
		if !ok || subnet != route.DestRange {
			fmt.Printf("delete route: %s\n", route.Name)
			_, err := rm.computeService.Routes.Delete(rm.project, route.Name).Do()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (rm GoogleRouterManager) routeMap() (map[string]*compute.Route, error) {
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
	filter := fmt.Sprintf("name eq flannel-%s-.*", rm.network)
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

func networkFromMetadata() (string, error) {
	network, err := metadataGet("/instance/network-interfaces/0/network")
	if err != nil {
		return "", err
	}
	return path.Base(network), nil
}

func projectFromMetadata() (string, error) {
	return metadataGet("/project/project-id")
}

func metadataGet(path string) (string, error) {
	req, err := http.NewRequest("GET", metadataEndpoint+path, nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("Metadata-Flavor", "Google")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
