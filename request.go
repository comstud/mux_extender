package mux_extender

import (
	"encoding/json"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"net/http"
)

func routeHandler(rt *Route) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		req := &RequestContext{
			Route:          rt,
			ResponseWriter: w,
			Request:        r,
			Params:         vars,
			StatusCode:     http.StatusOK,
		}
		global_vars := context.GetAll(nil)
		if global_vars != nil {
			for k, v := range global_vars {
				req.SetState(k, v)
			}
		}
		resp := rt.Fn(req)
		if resp != nil {
			req.Respond(resp)
		}
	}
}

type RequestContext struct {
	Route          *Route
	ResponseWriter http.ResponseWriter
	Request        *http.Request
	Params         map[string]string
	StatusCode     int
	statusWritten  bool
}

func (self *RequestContext) GetParam(name string) string {
	return self.Params[name]
}

func (self *RequestContext) SetState(key, val interface{}) {
	context.Set(self.Request, key, val)
}

func (self *RequestContext) GetState(key interface{}) interface{} {
	return context.Get(self.Request, key)
}

func (self *RequestContext) SetHeader(name, val string) {
	self.ResponseWriter.Header()[name] = []string{val}
}

func (self *RequestContext) SetStatus(status int) {
	self.StatusCode = status
}

func (self *RequestContext) Write(data []byte) (int, error) {
	if !self.statusWritten {
		self.ResponseWriter.WriteHeader(self.StatusCode)
		self.statusWritten = true
	}
	return self.ResponseWriter.Write(data)
}

type APIResponse struct {
	StatusCode int
	Response   interface{}
}

func Response(status int, v interface{}) *APIResponse {
	return &APIResponse{StatusCode: status, Response: v}
}

func (self *RequestContext) Respond(v interface{}) {
	if api_resp, ok := v.(*APIResponse); ok {
		self.StatusCode = api_resp.StatusCode
		v = api_resp.Response
	}

	if !self.statusWritten {
		self.ResponseWriter.WriteHeader(self.StatusCode)
		self.statusWritten = true
	}

	enc := json.NewEncoder(self.ResponseWriter)
	err := enc.Encode(v)
	if err != nil {
		panic(err)
	}
}
