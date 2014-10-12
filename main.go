package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/kelseyhightower/flannel-route-manager/backends/google"
)

var (
	backend      string
	etcdEndpoint string
	etcdPrefix   string
	deleteRoutes bool
	syncInterval int
)

type routeInfo struct {
	PublicIP string
}

func init() {
	flag.StringVar(&backend, "backend", "google", "backend provider")
	flag.StringVar(&etcdEndpoint, "etcd-endpoint", "http://127.0.0.1:4001", "etcd endpoint")
	flag.StringVar(&etcdPrefix, "etcd-prefix", "/coreos.com/network", "etcd prefix")
	flag.BoolVar(&deleteRoutes, "delete-all-routes", false, "delete all flannel routes")
	flag.IntVar(&syncInterval, "sync-interval", 300, "sync interval")
}

func main() {
	flag.Parse()
	log.SetFlags(0)
	var routeManager RouteManager
	var err error
	switch backend {
	case "google":
		routeManager, err = google.New()
		if err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatal("unknown backend ", backend)
	}
	if deleteRoutes {
		err := routeManager.DeleteAllRoutes()
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
		os.Exit(0)
	}
	sm := newStateManager(etcdEndpoint, etcdPrefix, syncInterval, routeManager).start()
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	c := <-signalChan
	log.Println(fmt.Sprintf("captured %v exiting...", c))
	sm.stop()
}
