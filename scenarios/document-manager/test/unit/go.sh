#!/bin/bash
# Run Go unit tests with coverage
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$SCENARIO_DIR/api"

echo "Running Go unit tests..."

if ! ls *_test.go >/dev/null 2>&1; then
  log::warning "⚠️  No Go unit tests found (*_test.go)"
  exit 0
fi

# Run tests with coverage
if go test -v -timeout=60s -coverprofile=coverage.out ./...; then
  # Extract coverage percentage
  if [ -f coverage.out ]; then
    coverage_percent=$(go tool cover -func=coverage.out 2>/dev/null | grep total | awk '{print $3}' | sed 's/%//' || echo "0")
    log::success "✅ Go tests passed with ${coverage_percent}% coverage"

    # Check coverage thresholds
    if (( $(echo "$coverage_percent < 70" | bc -l 2>/dev/null || echo "0") )); then
      log::error "❌ Coverage below 70%: ${coverage_percent}%"
      exit 1
    elif (( $(echo "$coverage_percent < 80" | bc -l 2>/dev/null || echo "0") )); then
      log::warning "⚠️  Coverage below 80%: ${coverage_percent}%"
    fi

    # Generate HTML coverage report
    go tool cover -html=coverage.out -o coverage.html 2>/dev/null || true
    log::info "Coverage report: api/coverage.html"
  fi
  exit 0
else
  log::error "❌ Go tests failed"
  exit 1
fi
