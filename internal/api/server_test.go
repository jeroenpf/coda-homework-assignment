package api

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"syscall"
	"testing"
	"time"
)

func TestServer(t *testing.T) {

	cfg := Config{
		Port: 8181,
	}
	errCh := make(chan error, 1)
	go func() {
		errCh <- Run(cfg)
	}()

	// Wait for server
	time.Sleep(100 * time.Millisecond)

	tests := []struct {
		name           string
		method         string
		path           string
		contentType    string
		body           string
		expectedStatus int
	}{
		{
			name:           "Health check",
			method:         http.MethodGet,
			path:           "/healthz",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Submit valid JSON",
			method:         http.MethodPost,
			path:           "/",
			expectedStatus: http.StatusOK,
			contentType:    "application/json",
			body:           `{"game":"Mobile Legends", "gamerID":"GYUTDTE", "points":20}`,
		},
		{
			name:           "Submit invalid JSON",
			method:         http.MethodPost,
			path:           "/",
			expectedStatus: http.StatusBadRequest,
			contentType:    "application/json",
			body:           "{invalid}",
		},
		{
			name:           "Submit invalid content type",
			method:         http.MethodPost,
			path:           "/",
			expectedStatus: http.StatusBadRequest,
			body:           `{"game":"Mobile Legends", "gamerID":"GYUTDTE", "points":20}`,
		},
		{
			name:           "Submit invalid http method",
			method:         http.MethodGet,
			path:           "/",
			expectedStatus: http.StatusMethodNotAllowed,
			contentType:    "application/json",
			body:           `{"game":"Mobile Legends", "gamerID":"GYUTDTE", "points":20}`,
		},
	}

	client := &http.Client{Timeout: 5 * time.Second}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			var body io.Reader
			if tt.body != "" {
				body = bytes.NewBuffer([]byte(tt.body))
			}

			req, err := http.NewRequest(tt.method, fmt.Sprintf("http://localhost:%d%s", cfg.Port, tt.path), body)

			if err != nil {
				t.Errorf("http.NewRequest() error = %v", err)
			}

			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			response, err := client.Do(req)
			if err != nil {
				t.Errorf("client.Do() error = %v", err)
			}
			defer response.Body.Close()

			if response.StatusCode != tt.expectedStatus {
				t.Errorf("server returned wrong status code: got %v want %v", response.StatusCode, tt.expectedStatus)
			}
		})
	}

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
	case <-time.After(6 * time.Second):
		t.Errorf("Failed to stop server within timeout")
	}

	// Verify server is actually stopped
	_, err = http.Get(fmt.Sprintf("http://localhost:%d/healthz", cfg.Port))
	if err == nil {
		t.Error("Server should not be accepting connections")
	}
}
