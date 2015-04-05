package main

import (
	"fmt"
	"log"
	. "mux_extender"
	"net/http"
	"strconv"
)

type ctxKey int

const testCtx ctxKey = 1
const userCtx ctxKey = 2

type errorResp struct {
	Error string `json:"error"`
}

type User struct {
	Id int `json:"id"`
}

var authWrapper = RouteFnWrapper(
	func(r *RequestContext, wrapped RouteFn) interface{} {
		fmt.Println("authWrapper before")
		resp := wrapped(r)
		fmt.Println("authWrapper after")
		return resp
	})

var userWrapper = RouteFnWrapper(
	func(r *RequestContext, wrapped RouteFn) interface{} {
		fmt.Println("userWrapper before")

		user_id := r.GetParam("id")
		fmt.Printf("Got request for user ID: %s\n", user_id)

		int_id, err := strconv.Atoi(user_id)
		if err != nil {
			r.Respond(Response(
				http.StatusBadRequest, errorResp{Error: err.Error()}))
			return nil
		}

		r.SetState(userCtx, &User{Id: int_id})

		resp := wrapped(r)
		fmt.Println("userWrapper after")
		return resp
	})

var defaultRouter = NewRouter()
var GET = GETRegistrar(defaultRouter)
var usersGET = GETRegistrar(usersRoute.Subrouter())

var usersRoute = GET("/users/{id}", userWrapper(getUser))

func getUser(r *RequestContext) interface{} {
	fmt.Printf("Got data: %+v\n", r.GetState(testCtx))

	user := r.GetState(userCtx).(*User)
	fmt.Printf("In getUser with: %d\n", user.Id)

	return user
}

var _ = usersGET("/profile", userWrapper(getUserProfile))

func getUserProfile(r *RequestContext) interface{} {
	user := r.GetState(userCtx).(*User)
	fmt.Printf("In getUserProfile with: %d\n", user.Id)
	return Response(
		http.StatusBadRequest,
		errorResp{Error: "Not implemented"},
	)
}

func main() {
	config := &struct{ S string }{S: "Test string"}
	// global state to copy to each request
	defaultRouter.SetState(testCtx, config)

	server := &http.Server{
		Addr:    ":8080",
		Handler: defaultRouter.WrapAll(authWrapper),
	}

	fmt.Printf("Starting server on port 8080.\n")
	log.Fatal(server.ListenAndServe())
}
