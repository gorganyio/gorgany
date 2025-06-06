package http

import (
	"fmt"
	"gorgany/app/core"
	err2 "gorgany/err"
	"gorgany/internal"
	"gorgany/service"
	"gorgany/util"
	"net/http"
	"reflect"
)

func Dispatch(w http.ResponseWriter, r *http.Request, handler core.HandlerFunc, middlewares []core.IMiddleware) {
	message := &Message{}
	err := service.GetContainer().Make(message, map[string]any{"writer": w, "request": r})
	if err != nil {
		err2.HandleErrorWithStacktrace(err)
		w.WriteHeader(500)
		return
	}

	defer func() {
		if r := recover(); r != nil {
			err, ok := r.(error)
			if !ok {
				err = fmt.Errorf("%v", r)
			}
			Catch(err, message)
		}
	}()

	for _, middleware := range internal.GetFrameworkRegistrar().GetMiddlewares() {
		middlewares = util.Prepend[core.IMiddleware](middlewares, middleware)
	}

	if !preProcess(middlewares, message) {
		return
	}

	if handler == nil {
		return
	}

	reflectedHandler := reflect.ValueOf(handler)
	resolver := inputResolver{
		reflectedHandler: reflectedHandler,
		message:          message,
	}

	args, err := resolver.resolve()
	if err != nil {
		Catch(err, message)
		return
	}

	reflectedHandler.Call(args)
}

func preProcess(middlewares []core.IMiddleware, message *Message) bool {
	if len(middlewares) == 0 {
		return true
	}

	preProcessed := true
	for _, middleware := range middlewares {
		res := middleware.Handle(message)
		if res == false {
			preProcessed = false
			break
		}
	}

	return preProcessed
}
