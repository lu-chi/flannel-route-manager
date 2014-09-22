// Copyright (c) 2014 Kelsey Hightower. All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0
// that can be found in the LICENSE file.
package google

import (
	"fmt"
	"strings"

	"code.google.com/p/goauth2/compute/serviceaccount"
	"code.google.com/p/google-api-go-client/compute/v1"
)

var replacer = strings.NewReplacer(".", "-", "/", "-")

type GoogleRouterManager struct {
	computeService *compute.Service
	network        string
	project        string
}

func New(project, network string) (*GoogleRouterManager, error) {
	client, err := serviceaccount.NewClient(&serviceaccount.Options{})
	if err != nil {
		return nil, err
	}
	computeService, err := compute.New(client)
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
	tags := make([]string, 0)
	for ip, subnet := range routeTable {
		route := &compute.Route{
			Name:      formatRouteName(rm.network, subnet),
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

func formatRouteName(network, subnet string) string {
	return fmt.Sprintf("flannel-%s-%s", network, replacer.Replace(subnet))
}
