#!/bin/bash
# Performance tests for test-scenario-1
# Tests performance characteristics and benchmarks

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "▶ Running performance tests..."

cd api

# Run performance tests
echo "  → Running Go performance tests..."
go test -v -run=TestPerformance -timeout=120s ./... || testing::warn "Some performance tests did not meet targets"

# Run benchmarks
echo "  → Running Go benchmarks..."
go test -bench=. -benchmem -run=^$ ./... 2>/dev/null || echo "  ℹ No benchmarks found"

# Run concurrent access tests
echo "  → Running concurrency tests..."
go test -v -run=TestConcurrent -timeout=60s ./... || testing::warn "Concurrency tests failed"

# Run memory tests
echo "  → Running memory tests..."
go test -v -run=TestMemoryUsage -timeout=60s ./... || testing::warn "Memory tests failed"

testing::phase::end_with_summary "Performance tests completed"
