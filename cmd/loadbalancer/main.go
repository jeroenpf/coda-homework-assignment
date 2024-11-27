package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/jeroenpf/coda-homework-assignment/internal/loadbalancer"
)

func main() {

	urls := []string{
		"http://localhost:8080",
		"http://localhost:8081",
	}

	config := loadbalancer.DefaultConfig()
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
