.PHONY: start-backends stop-backends start-loadbalancer stop-loadbalancer start-all stop-all run

# Start all three backend API servers in the background
start-backends:
	@echo "Starting backend API servers..."
	@go build -o tmp_api cmd/api/main.go
	@./tmp_api --port=8081 > logs/backend1.log 2>&1 & echo $$! > backend1.pid
	@./tmp_api --port=8082 > logs/backend2.log 2>&1 & echo $$! > backend2.pid
	@./tmp_api --port=8083 > logs/backend3.log 2>&1 & echo $$! > backend3.pid
	@rm tmp_api
	@echo "Backend servers started on ports 8081, 8082, and 8083"

# Stop all backend servers using their stored PIDs
stop-backends:
	@echo "Stopping backend API servers..."
	@-for pid in $$(cat backend*.pid 2>/dev/null); do \
		echo "Stopping PID: $$pid"; \
		kill -TERM $$pid 2>/dev/null || true; \
	done
	@rm -f backend*.pid
	@echo "Backend servers stopped"

# Start the load balancer
start-loadbalancer:
	@echo "Starting load balancer..."
	@go build -o tmp_loadbalancer cmd/loadbalancer/main.go || (echo "Build failed"; exit 1)
	@echo "Build successful, attempting to start..."
	@export BACKEND_SERVERS=http://localhost:8081,http://localhost:8082,http://localhost:8083; \
		./tmp_loadbalancer --port=8080 > logs/loadbalancer.log 2>&1 & pid=$$!;  \
		sleep 2;
	@rm tmp_loadbalancer

# Stop the load balancer
stop-loadbalancer:
	@echo "Stopping load balancer..."
	@-if [ -f loadbalancer.pid ]; then \
		pid=$$(cat loadbalancer.pid); \
		echo "Stopping PID: $$pid"; \
		kill -TERM $$pid 2>/dev/null || true; \
		rm loadbalancer.pid; \
	fi
	@echo "Load balancer stopped"

# Start everything
start-all:
	@mkdir -p logs
	@make start-backends
	@make start-loadbalancer
	@echo "All services started"

# Stop everything
stop-all: stop-loadbalancer stop-backends
	@echo "All services stopped"