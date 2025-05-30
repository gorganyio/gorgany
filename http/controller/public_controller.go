package controller

import (
	"gorgany/app/core"
	"gorgany/http/router"
	"mime"
	"os"
	path2 "path"
	"strings"
)

func NewPublicController() *PublicController {
	return &PublicController{}
}

type PublicController struct {
}

func (thiz PublicController) load(message core.HttpMessage) {
	r := message.GetRequest()
	url := r.URL
	path := url.Path
	file, err := os.ReadFile(path2.Join("resource", path))
	if err != nil {
		message.Response("", 404)
		return
	}

	splittedPath := strings.Split(path, "/")
	fileName := splittedPath[len(splittedPath)-1]
	splittedName := strings.Split(fileName, ".")
	ext := splittedName[len(splittedName)-1]
	kind := mime.TypeByExtension("." + ext)
	message.ResponseHeader().Set("content-type", kind)
	message.ResponseBytes(file, 200)
}

func (thiz PublicController) GetRoutes() []core.IRouteConfig {
	return []core.IRouteConfig{
		&router.RouteConfig{
			Path:    "/public/*",
			Method:  core.GET,
			Handler: thiz.load,
		},
	}
}
