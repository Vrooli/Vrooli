#!/bin/bash
# Performance tests for scenario-to-desktop

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "⚡ Running performance tests..."

if [[ -f "api/performance_test.go" ]]; then
    echo "🏃 Running API performance benchmarks..."
    cd api

    # Run performance tests (not in short mode)
    go test -v -run "Performance" -timeout 120s performance_test.go test_helpers.go test_patterns.go main.go 2>&1 | tee performance-results.log
    local test_result=$?

    # Extract performance metrics
    if grep -q "WARNING" performance-results.log; then
        echo "⚠️  Some performance targets not met (see warnings above)"
    else
        echo "✅ All performance targets met"
    fi

    cd ..

    if [[ $test_result -ne 0 ]]; then
        testing::phase::end_with_error "Performance tests failed"
    fi
else
    echo "⏭️  No performance tests found, skipping..."
fi

testing::phase::end_with_summary "Performance tests completed"
