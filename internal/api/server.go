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

	"github.com/avast/retry-go/v4"
	"github.com/google/uuid"
	"github.com/hashicorp/consul/api"
	"golang.org/x/sync/errgroup"
)

type ConsulConfig struct {
	Address string
	Timeout time.Duration
}

type Config struct {
	Port         int
	ConsulConfig ConsulConfig
	Environment  string
}

func Run(cfg Config) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	consulClient, err := initConsulClient(cfg.ConsulConfig)
	if err != nil {
		return fmt.Errorf("init consul client: %w", err)
	}

	serviceId := fmt.Sprintf("backend-%s", uuid.New())
	registration := createServiceRegistration(serviceId, cfg)

	// Set up the HTTP server
	server := createHTTPServer(cfg)
	g, gCtx := errgroup.WithContext(ctx)
	// Start the API
	g.Go(func() error {
		return runHTTPServer(server)
	})

	// Register service with Consul
	g.Go(func() error {
		return registerServiceWithRetry(ctx, consulClient, registration)
	})

	// Handle context done and termination signals
	g.Go(func() error {
		return handleShutdown(gCtx, server, consulClient, serviceId, cancel)
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

func initConsulClient(cfg ConsulConfig) (*api.Client, error) {
	config := api.DefaultConfig()
	config.Address = cfg.Address
	return api.NewClient(config)
}

func createServiceRegistration(serviceId string, cfg Config) *api.AgentServiceRegistration {
	return &api.AgentServiceRegistration{
		ID:      serviceId,
		Name:    "backend",
		Port:    cfg.Port,
		Address: "host.docker.internal",
		Tags:    []string{"backend", "api", "v1"},
		Meta: map[string]string{
			"version": "1.0",
			"env":     cfg.Environment,
		},
		Check: &api.AgentServiceCheck{
			HTTP:                           fmt.Sprintf("http://host.docker.internal:%d/healthz", cfg.Port),
			Interval:                       "10s",
			Timeout:                        "5s",
			DeregisterCriticalServiceAfter: "30s",
		},
	}
}

func createHTTPServer(cfg Config) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/", withLogging(jsonEchoHandler()))
	mux.HandleFunc("/healthz", withLogging(healthCheckHandler()))

	return &http.Server{
		Addr:         fmt.Sprintf("0.0.0.0:%d", cfg.Port),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

func runHTTPServer(server *http.Server) error {
	slog.Info(fmt.Sprintf("Starting API server on port %s", server.Addr))
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server failed to start: %v", err)
	}
	return nil
}

func handleShutdown(ctx context.Context, server *http.Server, consulClient *api.Client, serviceId string, cancel context.CancelFunc) error {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		slog.Info("shutting down gracefully after termination signal")
	case <-ctx.Done():
		slog.Info("shutting down gracefully after context cancellation")
	}
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := consulClient.Agent().ServiceDeregister(serviceId); err != nil {
		slog.Error("deregistering service failed: %v", err)
	}

	cancel()

	slog.Info("initiating graceful shutdown")
	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server failed to shutdown: %v", err)
	}

	return nil
}

func registerServiceWithRetry(ctx context.Context, consulClient *api.Client, registration *api.AgentServiceRegistration) error {
	return retry.Do(func() error {
		return consulClient.Agent().ServiceRegister(registration)
	},
		retry.OnRetry(func(u uint, err error) {
			slog.Warn("service registration failed, retrying", "attempt", u+1, err)
		}),
		retry.Context(ctx),
		retry.Attempts(5),
		retry.Delay(time.Second),
		retry.DelayType(retry.BackOffDelay),
	)
}
