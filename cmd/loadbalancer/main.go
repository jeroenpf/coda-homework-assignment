package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"strings"

	"github.com/jeroenpf/coda-homework-assignment/internal/loadbalancer"
)

func main() {

	backends := os.Getenv("BACKEND_SERVERS")
	urls := strings.Split(backends, ",")

	var port string
	flag.StringVar(&port, "port", "8080", "port to listen on")
	flag.Parse()

	config := loadbalancer.DefaultConfig()
	config.Port = port
	config.BackendUrls = urls

	srv, err := loadbalancer.NewServer(config)

	if err != nil {
		slog.Error("failed to create server", "error", err)
		os.Exit(1)
	}

	if err := srv.Start(context.Background()); err != nil {
		slog.Error("failed to start server", "error", err)
		os.Exit(1)
	}
}
