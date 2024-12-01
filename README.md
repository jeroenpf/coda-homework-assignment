# HTTP Round Robin API
This repository contains a simple Round Robin load balancer and echo API server written in Golang.

## How to use

### Run the servers
For convenience, docker-compose is used to run 3 backend servers and the Round Robin loadbalancer.

Simply run `docker-compose up` - this starts 3 api servers and 1 Round Robin loadbalancer. 
The balancer is exposed on port 8080.

### Access endpoint
The backends expose an endpoint that accepts a `POST` request. You can request this through the loadbalancer:

```bash
curl -X POST -H "Content-Type: application/json" \
-d '{"game":"Mobile Legends", "gamerID":"GYUTDTE", "points":20}' \
http://localhost:8080
```

### Running without docker
A makefile has been included that builds and runs backend servers

To start 3 backend servers on ports 8081, 8082 and 8083 and a loadbalancer on port 8080, run the following:

`make start-all`

And to stop it all:

`make stop-all`

## Running tests
Tests are included and can be run as follows: `go test ./...` - these tests cover the round robin logic, loadbalancing and api backends.

Additionally, you could run ab benchmarks to stress-test the application.

## Description

The project consists of two programs: 

### API
The API exposes an echo endpoint (`POST /` ) that accepts a JSON payload and returns 
the same payload in the response. 

Additionally, a health endpoint ( `GET /healthz` ) is exposed that can
be used to get the server health. For now, this is a dummy endpoint that always returns a HTTP OK 200 response.
In a real-world scenario, the health endpoint would take into consideration various metrics (e.g. db connectivity) to determine its health.

### Round Robin Loadbalancer
The loadbalancer expects comma separated backends to be defined via the `BACKEND_SERVERS` environment variable.
The loadbalancer also checks the health of the backends periodically and only forwards requests to recently healthy backends.

## Considerations

- Security: the loadbalancer and API currently supports only HTTP. Ideally, TLS should be enforced.
- We could improve this by implementing a weighted variant of round robin that takes into account the capacity and performance of the backends and dynamically adjusts the weights.
- We could add a circuit-breaker mechanism that further prevents sending traffic to failing backends
- Collect metrics such as active connections, backend health, failed requests, etc. For display in a monitoring tool such as prometheus.