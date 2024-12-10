package loadbalancer

import (
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/jeroenpf/coda-homework-assignment/internal/servicediscovery"
)

// LoadBalancer implements a weighted round robin load balancer
type LoadBalancer struct {
	Backends       []*Backend
	RRCounter      atomic.Uint32
	serviceName    string
	serviceWatcher servicediscovery.ServiceWatcher
	mu             sync.RWMutex
}

// NewLoadBalancer creates a new loadbalancer with the given backends
func NewLoadBalancer(watcher servicediscovery.ServiceWatcher, serviceName string) *LoadBalancer {
	slog.Info("initializing load balancer")
	return &LoadBalancer{
		serviceName:    serviceName,
		serviceWatcher: watcher,
	}
}

func (lb *LoadBalancer) updateBackends(urls []string) {
	backends, err := NewBackends(urls)
	if err != nil {
		slog.Error("failed to create backends", "error", err)
	}

	lb.mu.Lock()
	lb.Backends = backends
	lb.mu.Unlock()

	slog.Info("updated backend list", "count", len(lb.Backends))
}

func (lb *LoadBalancer) StartServiceWatcher() error {
	return lb.serviceWatcher.Start(lb.serviceName, lb.updateBackends)
}

func (lb *LoadBalancer) StopServiceWatcher() error {
	return lb.serviceWatcher.Stop()
}

// ServeHTTP serves a request that is proxied to one of available (and healthy) backends
func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	backend, err := lb.NextBackend()
	if err != nil {
		slog.Error("failed to get next backend", "error", err)
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	slog.Info("proxying request", "method", r.Method, "path", r.URL.Path, "backend", backend.Addr)
	backend.ReverseProxy.ServeHTTP(w, r)
}

// NextBackend tries to find the next healthy backend to proxy a request to
func (lb *LoadBalancer) NextBackend() (*Backend, error) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	// Consider only healthy backends
	healthyBackends := make([]*Backend, 0, len(lb.Backends))
	for _, backend := range lb.Backends {
		if !backend.Healthy {
			slog.Debug("skipping unhealthy backend", "backend", backend.Addr)
			continue
		}

		healthyBackends = append(healthyBackends, backend)
	}

	if len(healthyBackends) == 0 {
		slog.Error("no healthy backends available", "total_backends", len(lb.Backends))
		return nil, errors.New("no backends available")
	}

	current := lb.RRCounter.Load()
	nextBackend := current % uint32(len(healthyBackends))
	lb.RRCounter.Add(1)
	slog.Debug("selected backend",
		"backend", healthyBackends[nextBackend].Addr,
		"counter", current,
		"index", nextBackend)
	return healthyBackends[nextBackend], nil
}
