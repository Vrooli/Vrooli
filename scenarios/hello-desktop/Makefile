# Hello Desktop - Minimal Desktop Deployment Test Scenario
# This scenario is designed to validate the full desktop deployment pipeline

SCENARIO_NAME := hello-desktop
VROOLI_ROOT ?= $(HOME)/Vrooli

.PHONY: help start stop status test logs clean build

help:
	@echo "Hello Desktop - Minimal Desktop Deployment Test"
	@echo ""
	@echo "Usage:"
	@echo "  make start    - Start the scenario"
	@echo "  make stop     - Stop the scenario"
	@echo "  make status   - Check scenario status"
	@echo "  make build    - Build API binary"
	@echo "  make test     - Run tests"
	@echo "  make clean    - Clean build artifacts"
	@echo ""
	@echo "Desktop Deployment:"
	@echo "  make deploy-desktop  - Full desktop deployment pipeline"

start:
	vrooli scenario start $(SCENARIO_NAME)

stop:
	vrooli scenario stop $(SCENARIO_NAME)

status:
	vrooli scenario status $(SCENARIO_NAME)

build:
	cd api && go build -o hello-desktop-api .

test:
	@echo "Testing API health..."
	@curl -sf http://localhost:$$(vrooli scenario port $(SCENARIO_NAME) API_PORT)/health || echo "API not running"

logs:
	vrooli scenario logs $(SCENARIO_NAME)

clean:
	rm -f api/hello-desktop-api
	rm -rf api/bin/
	rm -rf platforms/

# Desktop deployment workflow
deploy-desktop: build
	@echo "=== Desktop Deployment Pipeline ==="
	@echo ""
	@echo "1. Creating deployment profile..."
	deployment-manager profile create hello-desktop-profile $(SCENARIO_NAME) --tier 2 || true
	@echo ""
	@echo "2. Checking fitness..."
	deployment-manager fitness $(SCENARIO_NAME) --tier 2 || true
	@echo ""
	@echo "3. Running deploy-desktop..."
	deployment-manager deploy-desktop --profile hello-desktop-profile --dry-run
