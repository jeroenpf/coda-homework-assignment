package servicediscovery

type ServiceWatcher interface {
	Start(serviceName string, handler func([]string)) error
	Stop() error
}
