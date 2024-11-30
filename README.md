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

Running this should respond with the exact same JSON payload.

## Running tests
Tests are included and can be run as follows: `go test ./...`