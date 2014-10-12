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
	sm.syncRoutes()
	go sm.monitorSubnets()
	go sm.reconciler()
	return sm
}

func (sm stateManager) stop() {
	close(sm.stopChan)
	sm.wg.Wait()
}

func (sm stateManager) syncRoutes() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	routeTable := make(map[string]string)
	resp, err := sm.client.Get(sm.prefix, false, true)
	if err != nil {
		return err
	}
	for _, node := range resp.Node.Nodes {
		subnet := strings.Replace(path.Base(node.Key), "-", "/", -1)
		var ri routeInfo
		err := json.Unmarshal([]byte(node.Value), &ri)
		if err != nil {
			return err
		}
		routeTable[ri.PublicIP] = subnet
	}
	log.Printf("syncing routes")
	err = sm.routeManager.Sync(routeTable)
	if err != nil {
		return err
	}
	return nil
}

func (sm stateManager) monitorSubnets() {
	sm.wg.Add(1)
	defer sm.wg.Done()
	respChan := make(chan struct{})
	go func() {
		for {
			resp, err := sm.client.Watch(sm.prefix, sm.lastIndex+1, true, nil, sm.stopChan)
			if err != nil {
				log.Println(err.Error())
				time.Sleep(10 * time.Second)
				continue
			}
			sm.lastIndex = resp.Node.ModifiedIndex
			respChan <- struct{}{}
		}
	}()
	for {
		select {
		case <-respChan:
			sm.syncRoutes()
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
			sm.syncRoutes()
		}
	}
}
