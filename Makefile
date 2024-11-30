.PHONY: start-backends stop-backends

# Start all three backend API servers in the background
start-backends:
	@echo "Starting backend API servers..."
	@go build -o tmp_api cmd/api/main.go
	@./tmp_api --port=8080 & echo $$! > backend1.pid
	@./tmp_api --port=8081 & echo $$! > backend2.pid
	@./tmp_api --port=8082 & echo $$! > backend3.pid
	@rm tmp_api
	@echo "Backend servers started on ports 8080, 8081, and 8082"

# Stop all backend servers using their stored PIDs
stop-backends:
	@echo "Stopping backend API servers..."
	@-for pid in $$(cat backend*.pid 2>/dev/null); do \
		echo "Stopping PID: $$pid"; \
		kill -TERM $$pid 2>/dev/null || true; \
	done
	@rm -f backend*.pid
	@echo "Backend servers stopped"