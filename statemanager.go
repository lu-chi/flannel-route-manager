package main

import (
	"encoding/json"
	"log"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-etcd/etcd"
)

type stateManager struct {
	mu           sync.Mutex
	client       *etcd.Client
	lastIndex    uint64
	prefix       string
	routeManager RouteManager
	stopChan     chan bool
	syncInterval int
	wg           sync.WaitGroup
}

func newStateManager(etcdEndpoint, prefix string, syncInterval int, routeManager RouteManager) stateManager {
	return stateManager{
		client:       etcd.NewClient([]string{etcdEndpoint}),
		prefix:       path.Join(prefix, "subnets"),
		routeManager: routeManager,
		stopChan:     make(chan bool),
		syncInterval: syncInterval,
	}
}

func (sm stateManager) start() stateManager {
	sm.syncAllRoutes()
	go sm.monitorSubnets()
	go sm.reconciler()
	return sm
}

func (sm stateManager) stop() {
	close(sm.stopChan)
	sm.wg.Wait()
}

func (sm stateManager) syncAllRoutes() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	routeTable := make(map[string]string)
	resp, err := sm.client.Get(sm.prefix, false, true)
	if err != nil {
		return err
	}
	sm.lastIndex = resp.EtcdIndex
	for _, node := range resp.Node.Nodes {
		subnet := strings.Replace(path.Base(node.Key), "-", "/", -1)
		var ri routeInfo
		err := json.Unmarshal([]byte(node.Value), &ri)
		if err != nil {
			return err
		}
		routeTable[ri.PublicIP] = subnet
	}
	log.Printf("syncing all routes")
	err = sm.routeManager.Sync(routeTable)
	if err != nil {
		return err
	}
	return nil
}

func (sm stateManager) syncRoute(resp *etcd.Response) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	subnet := strings.Replace(path.Base(resp.Node.Key), "-", "/", -1)
	switch resp.Action {
	case "create", "set", "update":
		var ri routeInfo
		err := json.Unmarshal([]byte(resp.Node.Value), &ri)
		if err != nil {
			log.Println(err.Error())
			return
		}
		err = sm.routeManager.Insert(ri.PublicIP, subnet)
		if err != nil {
			log.Println(err.Error())
		}
	case "delete":
		err := sm.routeManager.Delete(subnet)
		if err != nil {
			log.Println(err.Error())
		}
	default:
		log.Printf("unknown etcd action: %s\n", resp.Action)
	}
}

func (sm stateManager) monitorSubnets() {
	sm.wg.Add(1)
	defer sm.wg.Done()
	respChan := make(chan *etcd.Response)
	go func() {
		for {
			resp, err := sm.client.Watch(sm.prefix, sm.lastIndex, true, nil, sm.stopChan)
			if err != nil {
				log.Println(err.Error())
				time.Sleep(10 * time.Second)
				continue
			}
			sm.lastIndex = resp.Node.ModifiedIndex + 1
			respChan <- resp
		}
	}()
	for {
		select {
		case resp := <-respChan:
			sm.syncRoute(resp)
		case <-sm.stopChan:
			break
		}
	}
}

func (sm stateManager) reconciler() {
	sm.wg.Add(1)
	defer sm.wg.Done()
	for {
		select {
		case <-sm.stopChan:
			break
		case <-time.After(time.Duration(sm.syncInterval) * time.Second):
			sm.syncAllRoutes()
		}
	}
}
