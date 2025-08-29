package main

import (
	"context"
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"keyboard/config"
	"keyboard/input"
	"keyboard/server"
)

//go:embed static
var staticFiles embed.FS

func main() {
	cfg := config.Load()

	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatalf("Failed to create static filesystem: %v", err)
	}

	httpServer := server.NewHTTPServer(cfg, staticFS)
	wsServer := server.NewWebSocketServer()

	keySimulator := input.NewKeySimulator()
	wsServer.SetInputSimulator(keySimulator)

	go func() {
		log.Printf("Starting server on :%s", cfg.Port)
		if err := httpServer.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	wsServer.SetupRoutes(httpServer.Router())
	go wsServer.Run()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	wsServer.Shutdown()
	log.Println("Server stopped")
}
