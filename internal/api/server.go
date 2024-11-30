package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
)

type Config struct {
	Port int
}

func Run(cfg Config) error {
	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()
	// Set up the HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/", withLogging(jsonEchoHandler()))
	mux.HandleFunc("/healthz", withLogging(healthCheckHandler()))
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start the API
	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		slog.Info(fmt.Sprintf("Starting API server on port %s", server.Addr))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("server failed to start: %v", err)
		}
		return nil
	})

	// Handle context done and termination signals
	g.Go(func() error {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		select {
		case <-quit:
			slog.Info("shutting down gracefully after termination signal")
		case <-gCtx.Done():
			slog.Info("shutting down gracefully after context cancellation")
		}
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		cancel()

		slog.Info("initiating graceful shutdown")
		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("server failed to shutdown: %v", err)
		}

		return nil
	})

	if err := g.Wait(); err != nil {
		return fmt.Errorf("http server error %w", err)
	}

	return nil
}

func withLogging(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		handler(w, r)
		slog.Info(
			"Request handled",
			"method", r.Method,
			"path", r.URL.Path,
			"port", r.URL.Port(),
			"duration",
			time.Since(start),
		)
	}
}

func healthCheckHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
}

func jsonEchoHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Ensure that we are receiving JSON
		if r.Header.Get("Content-Type") != "application/json" {
			http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
			return
		}

		// Get the body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Invalid body", http.StatusBadRequest)
			return
		}

		// Ensure we received valid JSON body
		var jsonData interface{}
		if err := json.Unmarshal(body, &jsonData); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Write the original body back into the response
		w.Write(body)
	}
}
