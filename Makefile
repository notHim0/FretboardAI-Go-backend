.PHONY: help build run test clean deps

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

deps: ## Install dependencies
	go mod download
	go mod tidy

build: ## Build the application
	go build -o bin/guitar-transcriber ./cmd/server

run: ## Run the application
	go run ./cmd/server/main.go

test: ## Run tests
	go test -v ./...

clean: ## Clean build artifacts and temporary files
	rm -rf bin/
	rm -rf uploads/
	rm -rf processed/
	rm -f *.db *.db-shm *.db-wal

dev: ## Run in development mode with auto-reload (requires air)
	air

docker-build: ## Build Docker image
	docker build -t guitar-transcriber:latest .

docker-run: ## Run Docker container
	docker run -p 8080:8080 --env-file .env guitar-transcriber:latest