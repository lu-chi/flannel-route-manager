package main

type RouteManager interface {
	DeleteAllRoutes() error
	Sync(map[string]string) error
}
