# HTTP Round Robin Load Balancer

A high-performance Round Robin load balancer and API server implementation in Go, with health checks, graceful shutdown, and unit testing.

## Features

- Round Robin load balancing with health checking
- Multiple backend support with automatic failover
- Health monitoring with configurable check intervals
- Graceful shutdown handling
- Docker support for easy testing
- Test coverage
- JSON echo API server implementation

## Architecture

The project consists of two main components:

### Load Balancer
- Distributes incoming requests across multiple backends using Round Robin algorithm
- Monitors backend health and removes unhealthy instances from rotation
- Provides automatic failover when backends become unavailable
- Implements graceful shutdown with configurable timeout

### Echo API Server
- Accepts POST requests with JSON payloads
- Validates incoming JSON content
- Returns exact copy of received JSON payload
- Includes health check endpoint (`/healthz`)
- Supports multiple concurrent instances

## Getting Started

### Prerequisites
- Go 1.23 or higher
- Docker and Docker Compose (optional)

### Running with Docker Compose

The easiest way to run the complete setup is using Docker Compose:

```bash
docker-compose up
```

This will start:
- 3 API server instances (ports 8081-8083)
- 1 Load balancer instance (port 8080)

### Running Manually

1. Start multiple API servers:
```bash
make start-backends
```

2. Start the load balancer:
```bash
make start-loadbalancer
```

To stop all services:
```bash
make stop-all
```

## Testing the Setup

1. Send a test request through the load balancer:
```bash
curl -X POST -H "Content-Type: application/json" \
-d '{"game":"Mobile Legends", "gamerID":"GYUTDTE", "points":20}' \
http://localhost:8080
```

2. Check health endpoint:
```bash
curl http://localhost:8080/healthz
```

## Configuration

### Load Balancer Configuration
- `port`: Port to listen on (default: 8080, configurable via command line flag)
- `BACKEND_SERVERS`: Comma-separated list of backend URLs (must be set as an environment variable)

### API Server Configuration
- `port`: Port to listen on (configurable via command line flag)

## Testing

Run the test suite:
```bash
go test ./...
```

The tests cover:
- Round Robin logic
- Backend health checking
- Failover scenarios
- API server functionality
- Load balancer behavior

## Future Improvements

Potential enhancements that could be added:
1. Metrics collection so we can use tools such as Prometheus
2. Circuit breaker implementation to handle failing backends
3. Weighted round robin support
4. TLS support
5. Dynamic backend registration/removal
6. More advanced health checks, now it always returns a HTTP OK 200 response