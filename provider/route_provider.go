package provider

import (
	"fmt"
	"github.com/go-chi/chi"
	"github.com/spf13/viper"
	"gorgany/app/core"
	"gorgany/http"
	"gorgany/http/router"
	"gorgany/internal"
	http2 "net/http"
	"reflect"
	"strings"
)

type RouteProvider struct {
	router core.Router
}

func NewRouteProvider() *RouteProvider {
	return &RouteProvider{}
}

func (thiz *RouteProvider) InitProvider() {
	thiz.RegisterRouter(router.NewGorganyRouter())
	thiz.caseSensitiveRoutes()
}

func (thiz *RouteProvider) RegisterRouter(router core.Router) {
	internal.GetFrameworkRegistrar().RegisterRouter(router)
	thiz.router = router
}

func (thiz *RouteProvider) RegisterController(controller core.IController) {
	availableLangsRegex := thiz.buildLangRegex()
	routerEngine := thiz.router.Engine().(chi.Router)

	caseSensitiveRoutes := viper.GetBool("app.server.caseSensitiveRoutes")

	for _, rc := range controller.GetRoutes() {
		routeConfig := rc.(*router.RouteConfig)
		handler := routeConfig.Handler

		reflectedHandler := reflect.TypeOf(handler)
		if reflectedHandler.Kind() != reflect.Func {
			reflectedController := reflect.TypeOf(controller)
			panic(fmt.Sprintf("Handler must be function. Controller: %s, route: %s", reflectedController.String(), routeConfig.Path))
		}

		middlewares := routeConfig.Middlewares

		thiz.router.RegisterRoute(routeConfig)
		route := routeConfig.Path
		if routeConfig.Namespace != "" {
			route = fmt.Sprintf("/{namespace:%s}%s", routeConfig.Namespace, route)
		}

		patterns := []string{route}

		if viper.GetBool("i18n.enabled") {
			patterns = append(patterns, fmt.Sprintf("/{lang:^(%s)?$}%s", availableLangsRegex, route))
		}

		for _, pattern := range patterns {
			if !caseSensitiveRoutes {
				pattern = strings.ToLower(pattern)
			}

			if pattern[len(pattern)-1] == '/' && len(pattern) > 1 {
				pattern = pattern[:len(pattern)-1]
			}

			routerEngine.Options(pattern, func(w http2.ResponseWriter, r *http2.Request) {
				http.Dispatch(w, r, nil, middlewares)
			})

			switch routeConfig.Method {
			case core.GET:
				routerEngine.Get(pattern, func(w http2.ResponseWriter, r *http2.Request) {
					http.Dispatch(w, r, handler, middlewares)
				})
				break
			case core.PUT:
				routerEngine.Put(pattern, func(w http2.ResponseWriter, r *http2.Request) {
					http.Dispatch(w, r, handler, middlewares)
				})
				break
			case core.DELETE:
				routerEngine.Delete(pattern, func(w http2.ResponseWriter, r *http2.Request) {
					http.Dispatch(w, r, handler, middlewares)
				})
				break
			case core.POST:
				routerEngine.Post(pattern, func(w http2.ResponseWriter, r *http2.Request) {
					http.Dispatch(w, r, handler, middlewares)
				})
				break
			default:
				panic("Method is unsupported yet")
			}
		}
	}
}

func (thiz *RouteProvider) SetHomeUrl(url string) {
	internal.GetFrameworkRegistrar().SetHomeUrl(url)
}

func (thiz *RouteProvider) RegisterMiddleware(middleware core.IMiddleware) {
	internal.GetFrameworkRegistrar().RegisterMiddleware(middleware)
}

func (thiz *RouteProvider) buildLangRegex() string {
	availableLangs := viper.GetStringSlice("i18n.lang.available")
	availableLangs = append(availableLangs, viper.GetString("i18n.lang.default"))

	return strings.Join(availableLangs, "|")
}

func (thiz *RouteProvider) caseSensitiveRoutes() {
	if !viper.GetBool("app.server.caseSensitiveRoutes") {
		thiz.router.Engine().(chi.Router).Use(func(next http2.Handler) http2.Handler {
			fn := func(w http2.ResponseWriter, r *http2.Request) {
				r.URL.Path = strings.ToLower(r.URL.Path)
				next.ServeHTTP(w, r)
			}
			return http2.HandlerFunc(fn)
		})
	}
}
