package loadbalancer

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

type Backend struct {
	Addr         string
	ReverseProxy *httputil.ReverseProxy
	Healthy      bool
	LastCheck    time.Time
}

// NewBackend Creates a new backend for the provided URL
func NewBackend(addr string) (*Backend, error) {
	backendUrl, err := url.Parse(addr)

	if err != nil {
		return nil, err
	}

	proxy := httputil.NewSingleHostReverseProxy(backendUrl)

	return &Backend{
		Addr:         addr,
		ReverseProxy: proxy,
		Healthy:      true,
		LastCheck:    time.Now(),
	}, nil
}

// NewBackends returns a slice of backends based on a given slice of backend URL's
func NewBackends(urls []string) ([]*Backend, error) {
	backends := make([]*Backend, 0, len(urls))

	for _, backendUrl := range urls {
		backend, err := NewBackend(backendUrl)

		if err != nil {
			return nil, err
		}

		backends = append(backends, backend)
	}

	return backends, nil
}

// IsHealthy checks if the backend's /healthz endpoint can be reached and returns a valid status code
func (b *Backend) IsHealthy(client *http.Client) bool {
	resp, err := client.Get(b.Addr + "/healthz")

	if err != nil {
		return false
	}

	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}
