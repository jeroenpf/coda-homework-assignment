package loadbalancer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
)

type Config struct {
	Port                string
	BackendUrls         []string
	ReadTimeout         time.Duration
	WriteTimeout        time.Duration
	IdleTimeout         time.Duration
	ShutdownTimeout     time.Duration
	HealthCheckInterval time.Duration
}

func DefaultConfig() Config {
	return Config{
		Port:                "3000",
		ReadTimeout:         15 * time.Second,
		WriteTimeout:        15 * time.Second,
		IdleTimeout:         60 * time.Second,
		ShutdownTimeout:     5 * time.Second,
		HealthCheckInterval: 15 * time.Second,
	}
}

type Server struct {
	config Config
	srv    *http.Server
	lb     *LoadBalancer
	hc     *HealthChecker
}

// NewServer creates a new serve
func NewServer(config Config) (*Server, error) {
	backends, err := NewBackends(config.BackendUrls)
	if err != nil {
		return nil, fmt.Errorf("could not create backends: %w", err)
	}

	lb := NewLoadBalancer(backends)
	hc := NewHealthChecker(backends, config.HealthCheckInterval)

	srv := &http.Server{
		Addr:         config.Port,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		IdleTimeout:  config.IdleTimeout,
		Handler:      lb,
	}

	return &Server{
		config: config,
		srv:    srv,
		lb:     lb,
		hc:     hc,
	}, nil
}

// Start starts the server
func (s *Server) Start(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	// Starting the HTTP server
	g.Go(func() error {
		slog.Info("starting loadbalancer", "addr", s.srv.Addr)
		if err := s.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("server failed to start: %w", err)
		}
		return nil
	})

	// Start checking backend health
	g.Go(func() error {
		s.hc.Start()
		<-ctx.Done()
		s.hc.Stop()
		return nil
	})

	// Handle shutdown signals
	g.Go(func() error {
		return s.handleSignals(ctx)
	})

	return g.Wait()
}

func (s *Server) handleSignals(ctx context.Context) error {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigs:
		slog.Info("Received signal", "signal", sig)
	case <-ctx.Done():
		slog.Info("Context cancelled")
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := s.srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server failed to shutdown: %v", err)
	}

	return nil
}
