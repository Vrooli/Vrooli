# Vrooli Resource Integration Tests - Quick Commands
# =================================================
#
# Convenient shortcuts for common test operations
#
# Usage:
#   make test                # Run all tests
#   make test-single         # Run single resource tests only
#   make test-scenarios      # Run business scenarios only
#   make test-resource R=qdrant  # Test specific resource
#   make test-debug R=qdrant     # Debug specific resource
#   make test-quick          # Quick test (reduced timeout)
#   make clean               # Clean test artifacts
#   make help                # Show this help

.PHONY: help test test-single test-scenarios test-resource test-debug test-quick clean status config

# Default target
.DEFAULT_GOAL := help

# Test runner location
TEST_RUNNER := ./run.sh

# Help target
help:
	@echo "Vrooli Resource Integration Test Commands"
	@echo "========================================"
	@echo ""
	@echo "Basic Commands:"
	@echo "  make test                 Run all tests"
	@echo "  make test-single          Run single resource tests only"
	@echo "  make test-scenarios       Run business scenarios only"
	@echo "  make test-resource R=name Test specific resource (e.g., R=qdrant)"
	@echo "  make test-debug R=name    Debug specific resource with verbose output"
	@echo "  make test-quick           Quick test run (30s timeout)"
	@echo ""
	@echo "Utility Commands:"
	@echo "  make clean                Clean test artifacts and logs"
	@echo "  make status               Show resource and test status"
	@echo "  make config               Show/validate configuration"
	@echo "  make list-scenarios       List available business scenarios"
	@echo ""
	@echo "Examples:"
	@echo "  make test-resource R=ollama"
	@echo "  make test-debug R=qdrant"
	@echo "  make test-scenarios FILTER=\"category=ai\""

# Run all tests
test:
	@echo "🧪 Running all integration tests..."
	@$(TEST_RUNNER)

# Run single resource tests only
test-single:
	@echo "📦 Running single resource tests..."
	@$(TEST_RUNNER) --single-only

# Run business scenario tests only
test-scenarios:
	@echo "🎯 Running business scenario tests..."
	@$(TEST_RUNNER) --scenarios-only $(if $(FILTER),--scenarios "$(FILTER)")

# Test specific resource
test-resource:
	@if [ -z "$(R)" ]; then \
		echo "❌ Error: Resource not specified. Use: make test-resource R=<resource>"; \
		exit 1; \
	fi
	@echo "🔍 Testing resource: $(R)"
	@$(TEST_RUNNER) --resource $(R)

# Debug specific resource
test-debug:
	@if [ -z "$(R)" ]; then \
		echo "❌ Error: Resource not specified. Use: make test-debug R=<resource>"; \
		exit 1; \
	fi
	@echo "🔍 Debug testing resource: $(R)"
	@$(TEST_RUNNER) --debug --resource $(R)

# Quick test run with reduced timeout
test-quick:
	@echo "⚡ Running quick tests (30s timeout)..."
	@$(TEST_RUNNER) --timeout 30 --fail-fast

# Clean test artifacts
clean:
	@echo "🧹 Cleaning test artifacts..."
	@rm -f /tmp/vrooli_test_*.log 2>/dev/null || true
	@rm -f /tmp/vrooli_http_*.log 2>/dev/null || true
	@rm -f /tmp/test-collection-* 2>/dev/null || true
	@echo "✅ Test artifacts cleaned"

# Show resource and test status
status:
	@echo "📊 Resource Status"
	@echo "=================="
	@../index.sh --action discover || echo "Resource discovery failed"
	@echo ""
	@echo "📋 Configuration Status"
	@echo "======================"
	@./framework/helpers/config-manager.sh status 2>/dev/null || echo "Config manager not available"

# Show/validate configuration
config:
	@echo "⚙️  Configuration Management"
	@echo "========================="
	@./framework/helpers/config-manager.sh ensure 2>/dev/null || echo "Config validation failed"

# List available scenarios
list-scenarios:
	@$(TEST_RUNNER) --list-scenarios

# Common test combinations
test-ai:
	@echo "🤖 Testing AI resources..."
	@$(TEST_RUNNER) --resource ollama --resource whisper --resource comfyui

test-storage:
	@echo "💾 Testing storage resources..."
	@$(TEST_RUNNER) --resource minio --resource qdrant --resource vault

test-automation:
	@echo "⚙️  Testing automation resources..."
	@$(TEST_RUNNER) --resource n8n --resource node-red --resource windmill

# Verbose test with business report
test-full:
	@echo "📊 Running full test suite with business report..."
	@$(TEST_RUNNER) --verbose --business-report