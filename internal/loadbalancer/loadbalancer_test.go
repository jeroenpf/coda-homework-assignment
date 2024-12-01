package loadbalancer

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestLoadBalancer(t *testing.T) {
	// create mock backends
	backend1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Server-Id", "backend 1")
		w.WriteHeader(http.StatusOK)
	}))
	defer backend1.Close()

	backend2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Server-Id", "backend 2")
		w.WriteHeader(http.StatusOK)
	}))
	defer backend2.Close()

	backends, err := NewBackends([]string{backend1.URL, backend2.URL})
	if err != nil {
		t.Fatalf("Failed to create backends: %v", err)
	}

	lb := NewLoadBalancer(backends)

	req, err := http.NewRequest("GET", "/test", nil)

	t.Run("Round Robin will distribute requests", func(t *testing.T) {
		response := make([]string, 4)

		for i := 0; i < 4; i++ {
			rec := httptest.NewRecorder()
			lb.ServeHTTP(rec, req)
			fmt.Println(rec.Header())
			response[i] = rec.Header().Get("X-Server-Id")
		}

		if response[0] == response[1] || response[2] == response[3] {
			t.Errorf("Expected alternating server IDs, got %v", response)
		}
	})

}

func TestBackendHealthChecker(t *testing.T) {
	var isHealthy1 atomic.Bool
	isHealthy1.Store(true)

	var isHealthy2 atomic.Bool
	isHealthy2.Store(true)

	backend1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			if isHealthy1.Load() {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusServiceUnavailable)
			}
			return
		}

		w.Header().Set("X-Server-Id", "failing")
		w.WriteHeader(http.StatusOK)
	}))
	defer backend1.Close()

	backend2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			if isHealthy2.Load() {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusServiceUnavailable)
			}
			return
		}
		w.Header().Set("X-Server-Id", "backup")
		w.WriteHeader(http.StatusOK)
	}))
	defer backend2.Close()

	backends, err := NewBackends([]string{backend1.URL, backend2.URL})
	if err != nil {
		t.Fatalf("Failed to create backends: %v", err)
	}

	hc := NewHealthChecker(backends, 100*time.Millisecond)
	lb := NewLoadBalancer(backends)

	hc.Start()
	defer hc.Stop()

	t.Run("Using healthy backend on backend failure", func(t *testing.T) {

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		lb.ServeHTTP(rec, req)
		if rec.Header().Get("X-Server-Id") != "failing" {
			fmt.Println(rec.Header().Get("X-Server-Id"))
			t.Error("First request should go to first backend")
		}

		isHealthy1.Store(false)

		// Give some time for the health checker to detect backend failure
		time.Sleep(200 * time.Millisecond)

		// Do another request and make sure we continue hitting the backup backend
		rec = httptest.NewRecorder()
		lb.ServeHTTP(rec, req)

		if rec.Header().Get("X-Server-Id") != "backup" {
			t.Error("Second request should go to second backend")
		}

		// Finally do an additional request to make sure that we still go to the backup backend
		rec = httptest.NewRecorder()
		lb.ServeHTTP(rec, req)

		if rec.Header().Get("X-Server-Id") != "backup" {
			t.Error("Second request should go to second backend")
		}
	})

	t.Run("No healthy backends causes error", func(t *testing.T) {
		isHealthy1.Store(false)
		isHealthy2.Store(false)
		req := httptest.NewRequest(http.MethodGet, "/test", nil)

		// Give some time for the health checker to detect backend failure
		time.Sleep(200 * time.Millisecond)

		// We expect an error
		rec := httptest.NewRecorder()
		lb.ServeHTTP(rec, req)

		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("Unexpected status code: %d", rec.Code)
		}
	})
}
