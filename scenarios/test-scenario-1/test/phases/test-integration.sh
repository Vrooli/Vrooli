#!/bin/bash
# Integration tests for test-scenario-1
# Tests the full HTTP API integration

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "▶ Running integration tests..."

# Check if API is available
if ! cd api; then
    testing::fail "API directory not found"
fi

# Run integration tests (if any specific integration test files exist)
if ls *integration*_test.go 2>/dev/null; then
    echo "  → Running Go integration tests..."
    go test -v -tags=integration -timeout=60s ./... || testing::fail "Integration tests failed"
else
    echo "  ℹ No specific integration test files found"
    echo "  → Running all tests as integration check..."
    go test -v -timeout=60s ./... &> /dev/null || testing::fail "Integration tests failed"
fi

testing::phase::end_with_summary "Integration tests completed"
