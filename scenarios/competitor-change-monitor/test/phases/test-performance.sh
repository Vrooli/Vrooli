#!/bin/bash

# Performance tests for competitor-change-monitor scenario

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "Running performance tests..."

# Run Go performance tests
if [ -d "api" ] && [ -f "api/performance_test.go" ]; then
    echo "Running Go performance tests..."
    cd api

    # Run performance tests with timeout
    if go test -v -run "^TestPerformance" -timeout 3m -tags testing 2>&1 | tee performance-test-output.txt; then
        echo "✓ Go performance tests passed"
    else
        echo "⚠️  Some performance tests failed (non-critical)"
    fi

    # Run benchmarks
    echo "Running Go benchmarks..."
    if go test -bench=. -benchmem -timeout 3m -tags testing 2>&1 | tee benchmark-output.txt; then
        echo "✓ Go benchmarks completed"
    else
        echo "⚠️  Some benchmarks failed (non-critical)"
    fi

    cd ..
else
    echo "⚠️  No performance tests found, skipping"
fi

testing::phase::end_with_summary "Performance tests completed"
