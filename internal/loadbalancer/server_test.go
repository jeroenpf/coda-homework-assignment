package loadbalancer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"syscall"
	"testing"
	"time"
)

func TestServer(t *testing.T) {
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

	t.Run("Server starts and shuts dowwn gracefully", func(t *testing.T) {
		config := DefaultConfig()
		config.Port = "8099"
		config.BackendUrls = []string{backend1.URL, backend2.URL}

		srv, err := NewServer(config)
		if err != nil {
			t.Fatalf("Failed to create server: %v", err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		errCh := make(chan error, 1)

		go func() {
			errCh <- srv.Start(ctx)
		}()

		time.Sleep(100 * time.Millisecond)

		resp, err := http.Get("http://localhost:8099/healthz")
		if err != nil {
			t.Fatalf("Failed to get healthz: %v", err)
		}

		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}

		cancel()

		select {
		case err := <-errCh:
			if err != nil {
				t.Fatalf("Failed to stop server: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Errorf("Failed to stop server within timeout")
		}
	})

	t.Run("Handle shutdown signals", func(t *testing.T) {
		config := DefaultConfig()
		config.Port = "8099"
		config.BackendUrls = []string{backend1.URL, backend2.URL}

		srv, err := NewServer(config)
		if err != nil {
			t.Fatalf("Failed to create server: %v", err)
		}

		errCh := make(chan error, 1)
		go func() {
			errCh <- srv.Start(context.Background())
		}()

		// Add some time to let the server start
		time.Sleep(100 * time.Millisecond)

		p, err := os.FindProcess(os.Getpid())
		if err != nil {
			t.Fatalf("Failed to find process: %v", err)
		}

		err = p.Signal(syscall.SIGTERM)
		if err != nil {
			t.Fatalf("Failed to send SIGTERM: %v", err)
		}

		select {
		case err := <-errCh:
			if err != nil {
				t.Fatalf("Failed to stop server: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Errorf("Failed to stop server within timeout")
		}
	})
}
