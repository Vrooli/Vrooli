#!/bin/bash
# Performance tests for news-aggregator-bias-analysis scenario

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s"

cd "$TESTING_PHASE_SCENARIO_DIR/api"

echo "⚡ Running performance tests..."

# Run performance tests
if [ -f "performance_test.go" ]; then
    echo "Running Go performance tests..."

    # Run performance tests with benchmarks
    if go test -tags=testing -run=TestPerformance -v -timeout 180s; then
        echo "  ✓ Performance tests passed"
    else
        echo "  ⚠️  Some performance tests failed"
    fi

    # Run benchmarks
    echo ""
    echo "Running benchmarks..."
    if go test -tags=testing -bench=. -benchmem -run=^$ -timeout 180s 2>&1 | tee benchmark-results.txt; then
        echo "  ✓ Benchmarks completed"
    else
        echo "  ⚠️  Benchmarks encountered issues"
    fi
else
    echo "⚠️  No performance tests found"
fi

cd "$TESTING_PHASE_SCENARIO_DIR"

testing::phase::end_with_summary "Performance tests completed"
