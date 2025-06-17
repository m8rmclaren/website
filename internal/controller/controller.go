package controller

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/m8rmclaren/website/internal/util"
)

type Controller struct {
	handlers map[string]http.Handler
	static   http.Handler
}

type Option func(*Controller)

func WithStaticRoot(staticDirectoryName string) Option {
	fs := http.FileServer(http.Dir(staticDirectoryName))
	return func(s *Controller) {
		s.handlers[staticDirectoryName] = fs
		s.static = fs
	}
}

func WithRoute(path string, handler http.Handler) Option {
	return func(s *Controller) {
		s.handlers[path] = handler
	}
}

func NewController(ctx context.Context, opts ...Option) *Controller {
	service := &Controller{}
	service.handlers = make(map[string]http.Handler)

	service.handlers["health"] = service.healthCheckHandler()

	for _, opt := range opts {
		opt(service)
	}

	return service
}

func (s *Controller) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	var head string
	var shiftedPath string
	head, shiftedPath = util.ShiftPath(req.URL.Path)

	handler, ok := s.handlers[head]
	if ok {
		req.URL.Path = shiftedPath
		handler.ServeHTTP(res, req)
		return
	}

	s.static.ServeHTTP(res, req)
	// s.respondWithError(res, fmt.Errorf("%s is not a valid path or it hasn't been registered", head), http.StatusBadRequest)
}

type Error struct {
	Error string `json:"error"`
}

func (s *Controller) respondWithError(w http.ResponseWriter, err error, code int) {
	errStruct := Error{}

	errStruct.Error = err.Error()
	log.Printf("error: %v", err)

	w.WriteHeader(code)
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(errStruct)
	if err != nil {
		log.Printf("failed to write to json encoder: %v", err)
	}
}

func (s *Controller) healthCheckHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte("{\"status\": \"ok\"}"))
		if err != nil {
			log.Fatal("failed to write to response writer")
		}
	})
}
