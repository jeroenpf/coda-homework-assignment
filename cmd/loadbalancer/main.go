package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jeroenpf/coda-homework-assignment/internal/loadbalancer"
	"golang.org/x/sync/errgroup"
)

/**
	Concept:
	Weighted Round Robin
	We will prefer sending requests to lower weighted backends because they perform better
    Slower backends or backends that have a high error ratio should receive less traffic
	We also periodically check the health of the backends. If we find a backend is down,
	we exclude it from round robin.

	- 503 when all backends are 'unhealthy'
*/

func main() {
	ctx := context.Background()
	urls := []string{
		"http://localhost:8080",
		"http://localhost:8081",
	}

	backends, err := loadbalancer.NewBackends(urls)
	if err != nil {
		log.Fatal(err)
	}

	lb := loadbalancer.NewLoadBalancer(backends)

	srv := &http.Server{
		Addr:         ":3000",
		Handler:      lb,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		slog.Info("Starting Loadbalancer", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("server failed to start: %v", err)
		}
		return nil
	})

	g.Go(func() error {
		// Check for backend health every 5 seconds
		hc := loadbalancer.NewHealthChecker(backends, 5*time.Second)
		hc.Start()

		// Wait for context cancellation
		<-ctx.Done()
		hc.Stop()
		return nil
	})

	g.Go(func() error {
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
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("server failed to shutdown: %v", err)
		}

		return nil
	})

	if err := g.Wait(); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}
