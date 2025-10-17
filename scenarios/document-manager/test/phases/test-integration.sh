#!/bin/bash
# Phase 4: Integration tests - <120 seconds
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"

echo "=== Phase 4: Integration Tests ==="
start_time=$(date +%s)
error_count=0
test_count=0

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$SCENARIO_DIR"

# Get scenario name
scenario_name="document-manager"

# Get dynamic ports
log::info "Discovering service ports..."
API_PORT=$(vrooli scenario port "$scenario_name" API_PORT 2>/dev/null || echo "17810")
UI_PORT=$(vrooli scenario port "$scenario_name" UI_PORT 2>/dev/null || echo "38106")

export API_PORT
export UI_PORT

log::info "Using ports - API: $API_PORT, UI: $UI_PORT"

# Test API health
((test_count++))
log::info "Testing API health..."
if curl -sf "http://localhost:$API_PORT/health" >/dev/null 2>&1; then
  log::success "✅ API health check passed"
else
  log::error "❌ API health check failed"
  ((error_count++))
fi

# Test database connectivity
((test_count++))
log::info "Testing database connectivity..."
if curl -sf "http://localhost:$API_PORT/api/system/db-status" >/dev/null 2>&1; then
  log::success "✅ Database connectivity verified"
else
  log::error "❌ Database connectivity failed"
  ((error_count++))
fi

# Test vector database
((test_count++))
log::info "Testing vector database..."
if curl -sf "http://localhost:$API_PORT/api/system/vector-status" >/dev/null 2>&1; then
  log::success "✅ Vector database connectivity verified"
else
  log::warning "⚠️  Vector database connectivity issues"
fi

# Test AI integration
((test_count++))
log::info "Testing AI integration..."
if curl -sf "http://localhost:$API_PORT/api/system/ai-status" >/dev/null 2>&1; then
  log::success "✅ AI integration verified"
else
  log::warning "⚠️  AI integration issues"
fi

# Test UI health
((test_count++))
log::info "Testing UI health..."
if curl -sf "http://localhost:$UI_PORT/ui-health" >/dev/null 2>&1; then
  log::success "✅ UI health check passed"
else
  log::warning "⚠️  UI health check failed"
fi

# Run CLI tests with BATS
if [ -f "cli/document-manager.bats" ]; then
  ((test_count++))
  log::info "Running CLI integration tests..."
  if bats cli/document-manager.bats 2>&1; then
    log::success "✅ CLI tests passed"
  else
    log::error "❌ CLI tests failed"
    ((error_count++))
  fi
else
  log::info "No CLI BATS tests found, skipping"
fi

# Run existing integration test script if it exists
if [ -f "test/integration-test.sh" ]; then
  ((test_count++))
  log::info "Running integration test suite..."
  if bash test/integration-test.sh 2>&1; then
    log::success "✅ Integration test suite passed"
  else
    log::error "❌ Integration test suite failed"
    ((error_count++))
  fi
fi

# Performance check
end_time=$(date +%s)
duration=$((end_time - start_time))

if [ $error_count -eq 0 ]; then
  log::success "✅ Integration tests completed in ${duration}s"
  log::success "   Tests run: $test_count, Errors: $error_count"
  exit 0
else
  log::error "❌ Integration tests failed with $error_count errors in ${duration}s"
  exit 1
fi
