package main

type RouteManager interface {
	Delete(route string) error
	DeleteAllRoutes() error
	Insert(ip, subnet string) error
	Sync(map[string]string) error
}
