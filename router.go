package mux_extender

import (
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"net/http"
)

type RouteWrapFn func(*RequestContext, RouteFn) interface{}
type RouteFn func(*RequestContext) interface{}
type RouteRegistrarFn func(path string, fn RouteFn) *Route

type Router struct {
	*mux.Router
	routes           map[string]*Route
	muxRouteToRoutes map[*mux.Route]*Route
	parentRoute      *Route // Set if we're a sub-router
	middlewares      []func(http.Handler) http.Handler
	wrappers         []func(RouteFn) RouteFn
	mwHead           http.Handler
	wrapperHead      RouteFn
	wrapperTail      RouteFn
}

func newRouter(mux_r *mux.Router, parent_rt *Route) *Router {
	wrapper := func(r *RequestContext) interface{} {
		resp := r.Route.Fn(r)
		if resp != nil {
			r.Respond(resp)
		}
		return resp
	}
	r := &Router{
		Router:           mux_r,
		routes:           make(map[string]*Route),
		muxRouteToRoutes: make(map[*mux.Route]*Route),
		parentRoute:      parent_rt,
		middlewares:      make([]func(http.Handler) http.Handler, 0),
		wrapperTail:      wrapper,
		wrapperHead:      wrapper,
	}
	r.mwHead = http.HandlerFunc(r.wrapperProcess)
	return r
}

func NewRouter() *Router {
	return newRouter(mux.NewRouter(), nil)
}

func (self *Router) GetRoutes() map[string]*Route {
	return self.routes
}

func (self *Router) addRoute(rt *Route) *Route {
	self.routes[rt.FullPath] = rt
	self.muxRouteToRoutes[rt.Route] = rt
	return rt
}

func (self *Router) RouteForMuxRoute(mux_rt *mux.Route) *Route {
	return self.muxRouteToRoutes[mux_rt]
}

func (self *Router) RouteForRequest(r *http.Request) *Route {
	return self.RouteForMuxRoute(mux.CurrentRoute(r))
}

// Set some global state that will get copied to all requests
func (self *Router) SetState(key, val interface{}) {
	context.Set(nil, key, val)
}

func (self *Router) GetState(key, val interface{}) interface{} {
	return context.Get(nil, key)
}

func (self *Router) AddMiddlewares(handlers ...func(http.Handler) http.Handler) *Router {
	self.middlewares = append(self.middlewares, handlers...)
	var last http.Handler
	last = http.HandlerFunc(self.wrapperProcess)
	for i := len(self.middlewares) - 1; i >= 0; i-- {
		last = self.middlewares[i](last)
	}
	self.mwHead = last
	return self
}

func (self *Router) AddWrappers(fns ...func(RouteFn) RouteFn) *Router {
	self.wrappers = append(self.wrappers, fns...)
	last := self.wrapperTail
	for i := len(self.wrappers) - 1; i >= 0; i-- {
		last = self.wrappers[i](last)
	}
	self.wrapperHead = last
	return self
}

func (self *Router) mwProcess(w http.ResponseWriter, r *http.Request) {
	self.mwHead.ServeHTTP(w, r)
}

func (self *Router) wrapperProcess(w http.ResponseWriter, r *http.Request) {
	rt := self.RouteForRequest(r)
	req := newRequestContext(rt, w, r)
	self.wrapperHead(req)
}

type Route struct {
	Method     string
	Path       string
	Fn         RouteFn
	FullPath   string
	router     *Router
	baseRouter *Router
	*mux.Route
}

func (self *Route) Register(r *Router) *Route {
	if r.parentRoute != nil {
		self.FullPath = r.parentRoute.FullPath + self.Path
		self.baseRouter = r.parentRoute.baseRouter
	} else {
		self.FullPath = self.Path
		self.baseRouter = r
	}
	self.router = r
	self.Route = r.HandleFunc(self.Path, self.baseRouter.mwProcess)
	self.Route.Name(self.FullPath).Methods(self.Method)
	if err := self.Route.GetError(); err != nil {
		panic(err)
	}
	self.baseRouter.addRoute(self)
	self.router.addRoute(self)
	return self
}

func (self *Route) Subrouter() *Router {
	sub_r := self.router.PathPrefix(self.Path + "/").Subrouter()
	return newRouter(sub_r, self)
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
