package loadbalancer

import (
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// HealthChecker periodically checks the health of backends
type HealthChecker struct {
	backends []*Backend
	interval time.Duration
	stopChan chan struct{}
	client   *http.Client
	mu       sync.Mutex
}

// NewHealthChecker creates a new HealthChecker that periodically checks the health of a backend
func NewHealthChecker(backends []*Backend, interval time.Duration) *HealthChecker {
	return &HealthChecker{
		interval: interval,
		stopChan: make(chan struct{}),
		client: &http.Client{
			Timeout: time.Second,
		},
		backends: backends,
	}
}

// Start starts a periodically scheduled task to check the health of the registered backends
func (h *HealthChecker) Start() {
	ticker := time.NewTicker(h.interval)
	go func() {
		for {
			select {
			case <-ticker.C:
				h.checkHealth()
			case <-h.stopChan:
				ticker.Stop()
				return
			}
		}
	}()
}

// Stop stops checking the health of registered backends
func (h *HealthChecker) Stop() {
	close(h.stopChan)
}

// checkHealth concurrently performs a check on the health of each registered backend
func (h *HealthChecker) checkHealth() {
	var wg sync.WaitGroup
	for _, backend := range h.backends {
		wg.Add(1)
		go func(backend *Backend) {
			defer wg.Done()
			healthy := backend.IsHealthy(h.client)
			h.mu.Lock()
			backend.Healthy = healthy
			backend.LastCheck = time.Now()
			h.mu.Unlock()
			slog.Info(fmt.Sprintf("Backend %s health check: %v", backend.Addr, healthy))
		}(backend)
	}
	wg.Wait()
}
