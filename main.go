// Copyright (c) 2014 Kelsey Hightower. All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0
// that can be found in the LICENSE file.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"
	"log"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"github.com/kelseyhightower/flannel-route-manager/backends/google"

	"github.com/coreos/go-etcd/etcd"
)

var (
	backend      string
	etcdEndpoint string
	etcdPrefix   string
	network      string
	project      string
	syncInterval int
)

type routeInfo struct {
	PublicIP string
}

func init() {
	flag.StringVar(&backend, "backend", "google", "backend provider")
	flag.StringVar(&etcdEndpoint, "etcd-endpoint", "http://127.0.0.1:4001", "etcd endpoint")
	flag.StringVar(&etcdPrefix, "etcd-prefix", "/coreos.com/network", "etcd prefix")
	flag.StringVar(&network, "network", "default", "google compute network")
	flag.StringVar(&project, "project", "", "google compute project name")
	flag.IntVar(&syncInterval, "sync-interval", 30, "sync interval")
}

func main() {
	flag.Parse()
	log.SetFlags(0)
	var routeManager RouteManager
	var err error
	switch backend {
	case "google":
		routeManager, err = google.New(project, network)
		if err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatal("unknown backend ", backend)
	}
	etcdClient := etcd.NewClient([]string{etcdEndpoint})
	key := path.Join(etcdPrefix, "subnets")
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	for {
		routeTable := make(map[string]string)
		resp, err := etcdClient.Get(key, false, true)
		if err != nil {
			log.Println(err)
			goto L1
		}
		for _, node := range resp.Node.Nodes {
			subnet := strings.Replace(path.Base(node.Key), "-", "/", -1)
			var ri routeInfo
			err := json.Unmarshal([]byte(node.Value), &ri)
			if err != nil {
				log.Println(err)
				goto L1
			}
			routeTable[ri.PublicIP] = subnet
		}
		log.Printf("syncing routes")
		err = routeManager.Sync(routeTable)
		if err != nil {
			log.Println(err)
		}
	L1:
		select {
		case c := <-signalChan:
			log.Println(fmt.Sprintf("captured %v exiting...", c))
			os.Exit(0)
		case <-time.After(time.Duration(syncInterval) * time.Second):
			// Continue syncing routes.
		}
	}
}
