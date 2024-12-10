package servicediscovery

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/consul/api"
)

type ConsulServiceWatcher struct {
	consulClient *api.Client
	done         chan struct{}
	started      bool
	mu           sync.Mutex
}

func NewConsulServiceWatcher(consulClient *api.Client) *ConsulServiceWatcher {
	return &ConsulServiceWatcher{
		consulClient: consulClient,
	}
}

func (w *ConsulServiceWatcher) Start(serviceName string, handler func([]string)) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.started {
		return fmt.Errorf("consul service watcher already started")
	}

	w.started = true
	w.done = make(chan struct{})

	go w.watch(serviceName, handler)
	return nil
}

func (w *ConsulServiceWatcher) watch(serviceName string, handler func([]string)) {
	var lastIndex uint64
	for {
		select {
		case <-w.done:
			return
		default:
			services, meta, err := w.consulClient.Health().Service(
				serviceName,
				"",
				true,
				&api.QueryOptions{
					WaitIndex: lastIndex,
					WaitTime:  10 * time.Second,
				})

			if err != nil {
				slog.Error("failed to fetch services", "error", err)
				time.Sleep(time.Second)
				continue
			}

			lastIndex = meta.LastIndex

			var urls []string
			for _, service := range services {
				addr := strings.Replace(service.Service.Address, "host.docker.internal", "localhost", 1)
				slog.Info("found service", "addr", addr, "port", service.Service.Port)
				urls = append(urls, fmt.Sprintf("http://%s:%d", addr, service.Service.Port))
			}
			handler(urls)
		}
	}
}

func (w *ConsulServiceWatcher) Stop() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.started {
		return fmt.Errorf("consul service watcher already stopped")
	}

	close(w.done)
	w.started = false
	return nil
}
