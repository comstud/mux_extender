package mux_extender

import (
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
)

type RouteWrapFn func(*RequestContext, RouteFn) interface{}
type RouteFn func(*RequestContext) interface{}
type RouteRegistrarFn func(path string, fn RouteFn) *Route

type Router struct {
	*mux.Router
	routes      map[string]*Route
	parentRoute *Route // Set if we're a sub-router
}

func NewRouter() *Router {
	return &Router{
		Router: mux.NewRouter(),
		routes: make(map[string]*Route),
	}
}

func (self *Router) addRoute(rt *Route) *Route {
	self.routes[rt.fullPath] = rt
	return rt
}

func (self *Router) WrapAll(fns ...func(RouteFn) RouteFn) *Router {
	for _, rt := range self.routes {
		last := rt.Fn
		for i := len(fns) - 1; i >= 0; i-- {
			last = fns[i](last)
		}
		rt.Fn = last
	}
	return self
}

// Set some global state that will get copied to all requests
func (self *Router) SetState(key, val interface{}) {
	context.Set(nil, key, val)
}

func (self *Router) GetState(key, val interface{}) interface{} {
	return context.Get(nil, key)
}

type Route struct {
	Method     string
	Path       string
	Fn         RouteFn
	fullPath   string
	router     *Router
	baseRouter *Router
	*mux.Route
}

func (self *Route) Register(r *Router) *Route {
	if r.parentRoute != nil {
		self.fullPath = r.parentRoute.fullPath + self.Path
		self.baseRouter = r.parentRoute.baseRouter
	} else {
		self.fullPath = self.Path
		self.baseRouter = r
	}
	self.router = r
	handleFn := routeHandler(self)
	self.Route = r.HandleFunc(self.Path, handleFn)
	self.Route.Name(self.fullPath).Methods(self.Method)
	if err := self.Route.GetError(); err != nil {
		panic(err)
	}
	self.baseRouter.addRoute(self)
	self.router.addRoute(self)
	return self
}

func (self *Route) Subrouter() *Router {
	sub_r := self.router.PathPrefix(self.Path + "/").Subrouter()
	return &Router{
		Router:      sub_r,
		routes:      make(map[string]*Route),
		parentRoute: self,
	}
}

func RouteFnWrapper(wrapper RouteWrapFn) func(RouteFn) RouteFn {
	return func(wrapped RouteFn) RouteFn {
		return func(r *RequestContext) interface{} {
			return wrapper(r, wrapped)
		}
	}
}

func makeRegistrar(method string, r *Router) RouteRegistrarFn {
	return func(path string, fn RouteFn) *Route {
		rt := &Route{
			Method: method,
			Path:   path,
			Fn:     fn,
		}
		return rt.Register(r)
	}
}

func GETRegistrar(r *Router) RouteRegistrarFn {
	return makeRegistrar("GET", r)
}

func HEADRegistrar(r *Router) RouteRegistrarFn {
	return makeRegistrar("HEAD", r)
}

func PATCHRegistrar(r *Router) RouteRegistrarFn {
	return makeRegistrar("PATCH", r)
}

func PUTRegistrar(r *Router) RouteRegistrarFn {
	return makeRegistrar("PUT", r)
}

func POSTRegistrar(r *Router) RouteRegistrarFn {
	return makeRegistrar("POST", r)
}

func DELETERegistrar(r *Router) RouteRegistrarFn {
	return makeRegistrar("DELETE", r)
}
