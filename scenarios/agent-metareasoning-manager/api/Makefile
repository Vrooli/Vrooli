# Makefile for Metareasoning API

.PHONY: help build run test coverage clean lint fmt deps benchmark profile

# Variables
BINARY_NAME=agent-metareasoning-manager-api
GO_FILES=$(shell find . -name '*.go' -not -path "./vendor/*")
COVERAGE_TARGET=90
# Use dynamic port from environment or fall back to default
POSTGRES_PORT ?= 5432
API_PORT ?= 8090
TEST_DB_URL=postgres://postgres:postgres@localhost:$(POSTGRES_PORT)/metareasoning_test?sslmode=disable

# Default target
help: ## Show this help message
	@echo "Metareasoning API - Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build the API binary
	@echo "Building $(BINARY_NAME)..."
	@go build -o $(BINARY_NAME) main.go
	@echo "Build complete: ./$(BINARY_NAME)"

run: ## Run the API server
	@echo "Starting Metareasoning Coordinator API..."
	@go run main.go

test: ## Run all tests
	@echo "Running tests..."
	@TEST_DATABASE_URL=$(TEST_DB_URL) go test -v -race ./...

coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	@TEST_DATABASE_URL=$(TEST_DB_URL) go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	@echo "\nCoverage Report:"
	@go tool cover -func=coverage.out
	@echo "\nGenerating HTML report..."
	@go tool cover -html=coverage.out -o coverage.html
	@echo "HTML report saved to coverage.html"
	@echo "\nChecking coverage target ($(COVERAGE_TARGET)%)..."
	@coverage=$$(go tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	if [ $$(echo "$$coverage >= $(COVERAGE_TARGET)" | bc) -eq 1 ]; then \
		echo "✓ Coverage target met: $$coverage%"; \
	else \
		echo "✗ Coverage below target: $$coverage% < $(COVERAGE_TARGET)%"; \
	fi

coverage-html: coverage ## Open coverage report in browser
	@echo "Opening coverage report..."
	@if command -v xdg-open > /dev/null; then \
		xdg-open coverage.html; \
	elif command -v open > /dev/null; then \
		open coverage.html; \
	else \
		echo "Please open coverage.html manually"; \
	fi

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME) coverage.out coverage.html *.prof
	@echo "Clean complete"

lint: ## Run linters
	@echo "Running linters..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run ./...; \
	else \
		go vet ./...; \
		go fmt ./... | grep -v "^$$" && echo "Files need formatting" && exit 1 || echo "All files formatted"; \
	fi

fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...
	@echo "Format complete"

deps: ## Install/update dependencies
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy
	@echo "Dependencies installed"

benchmark: ## Run benchmarks
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem ./...

performance-test: ## Run performance monitoring
	@echo "Running performance tests..."
	@./monitor_performance.sh

benchmark-performance: ## Run performance benchmarks
	@echo "Running performance benchmarks..."
	@go test -bench=BenchmarkHealthHandler -benchtime=10s
	@go test -bench=BenchmarkWriteJSON -benchtime=5s
	@go test -bench=BenchmarkPerformanceMiddleware -benchtime=5s

profile: ## Generate CPU and memory profiles
	@echo "Generating profiles..."
	@go test -cpuprofile=cpu.prof -memprofile=mem.prof -bench=. ./...
	@echo "Profiles generated: cpu.prof, mem.prof"
	@echo "View with: go tool pprof cpu.prof"

test-verbose: ## Run tests with verbose output
	@TEST_DATABASE_URL=$(TEST_DB_URL) go test -v -race -count=1 ./...

test-short: ## Run short tests only
	@TEST_DATABASE_URL=$(TEST_DB_URL) go test -short ./...

test-integration: ## Run integration tests (requires running services)
	@echo "Running integration tests..."
	@TEST_DATABASE_URL=$(TEST_DB_URL) go test -v -tags=integration ./...

check: lint test ## Run linters and tests
	@echo "All checks passed!"

install: build ## Install the binary to GOPATH/bin
	@echo "Installing $(BINARY_NAME)..."
	@go install .
	@echo "Installed to $$(go env GOPATH)/bin/$(BINARY_NAME)"

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@docker build -t agent-metareasoning-manager-api:latest .

docker-run: ## Run API in Docker container
	@echo "Running in Docker..."
	@docker run -p $(SERVICE_PORT):$(SERVICE_PORT) --env-file .env agent-metareasoning-manager-api:latest

# Development helpers
dev: ## Run with hot reload (requires air)
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "Please install air: go install github.com/air-verse/air@latest"; \
		echo "Falling back to regular run..."; \
		$(MAKE) run; \
	fi

setup-test-db: ## Create test database
	@echo "Setting up test database..."
	@psql -U postgres -c "CREATE DATABASE metareasoning_test;" || true
	@echo "Test database ready"

teardown-test-db: ## Drop test database
	@echo "Dropping test database..."
	@psql -U postgres -c "DROP DATABASE IF EXISTS metareasoning_test;"
	@echo "Test database dropped"

# Performance testing
load-test: ## Run load tests (requires vegeta)
	@if command -v vegeta > /dev/null; then \
		echo "GET http://localhost:$(SERVICE_PORT)/health" | vegeta attack -duration=30s -rate=100 | vegeta report; \
	else \
		echo "Please install vegeta: go install github.com/tsenart/vegeta@latest"; \
	fi

# Documentation
docs: ## Generate API documentation
	@echo "Generating documentation..."
	@if command -v godoc > /dev/null; then \
		echo "Documentation server starting at http://localhost:6060"; \
		godoc -http=:6060; \
	else \
		echo "Please install godoc: go install golang.org/x/tools/cmd/godoc@latest"; \
	fi

# CI/CD targets
ci: deps lint test coverage ## Run CI pipeline
	@echo "CI pipeline complete"

.DEFAULT_GOAL := help