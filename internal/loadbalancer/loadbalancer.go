package loadbalancer

import (
	"errors"
	"log/slog"
	"net/http"
	"sync/atomic"
)

// LoadBalancer implements a weighted round robin load balancer
type LoadBalancer struct {
	Backends  []*Backend
	RRCounter atomic.Uint32
}

// NewLoadBalancer creates a new loadbalancer with the given backends
func NewLoadBalancer(backends []*Backend) *LoadBalancer {
	slog.Info("initializing load balancer", "backend_count", len(backends))
	return &LoadBalancer{
		Backends: backends,
	}
}

// ServeHTTP serves a request that is proxied to one of available (and healthy) backends
func (l *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	backend, err := l.NextBackend()
	if err != nil {
		slog.Error("failed to get next backend", "error", err)
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	slog.Info("proxying request", "method", r.Method, "path", r.URL.Path, "backend", backend.Addr)
	backend.ReverseProxy.ServeHTTP(w, r)
}

// NextBackend tries to find the next healthy backend to proxy a request to
func (l *LoadBalancer) NextBackend() (*Backend, error) {
	// Consider only healthy backends
	healthyBackends := make([]*Backend, 0, len(l.Backends))
	for _, backend := range l.Backends {
		if !backend.Healthy {
			slog.Debug("skipping unhealthy backend", "backend", backend.Addr)
			continue
		}

		healthyBackends = append(healthyBackends, backend)
	}

	if len(healthyBackends) == 0 {
		slog.Error("no healthy backends available", "total_backends", len(l.Backends))
		return nil, errors.New("no backends available")
	}

	current := l.RRCounter.Add(1)
	nextBackend := current % uint32(len(healthyBackends))
	slog.Debug("selected backend",
		"backend", healthyBackends[nextBackend].Addr,
		"counter", current,
		"index", nextBackend)
	return healthyBackends[nextBackend], nil
}
