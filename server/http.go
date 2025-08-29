package server

import (
	"context"
	"io/fs"
	"net/http"
	"time"

	"keyboard/config"

	"github.com/gorilla/mux"
)

type HTTPServer struct {
	server  *http.Server
	router  *mux.Router
	staticFS fs.FS
}

func NewHTTPServer(cfg *config.Config, staticFS fs.FS) *HTTPServer {
	router := mux.NewRouter()

	return &HTTPServer{
		server: &http.Server{
			Addr:         ":" + cfg.Port,
			Handler:      router,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		router:   router,
		staticFS: staticFS,
	}
}

func (s *HTTPServer) Router() *mux.Router {
	return s.router
}

func (s *HTTPServer) Start() error {
	s.setupStaticFiles()
	return s.server.ListenAndServe()
}

func (s *HTTPServer) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (s *HTTPServer) setupStaticFiles() {
	s.router.PathPrefix("/").Handler(http.FileServer(http.FS(s.staticFS)))
}
