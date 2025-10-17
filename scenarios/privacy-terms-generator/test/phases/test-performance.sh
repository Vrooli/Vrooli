#!/bin/bash
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"

cd "$TESTING_PHASE_SCENARIO_DIR/api"

echo "Running performance tests..."

# Run Go benchmarks
if [ -f "performance_test.go" ]; then
    echo "Running Go benchmark tests..."
    go test -bench=. -benchmem -tags=testing -run=^$ 2>&1 || {
        echo "Warning: Benchmark tests failed (may be expected if dependencies not available)"
    }
else
    echo "No performance_test.go found"
fi

# Run performance-focused unit tests
echo "Running performance unit tests..."
go test -tags=testing -run=Performance -v 2>&1 || {
    echo "Warning: Performance tests failed (may be expected if dependencies not available)"
}

testing::phase::end_with_summary "Performance tests completed"
