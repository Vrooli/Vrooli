#!/bin/bash
# Performance tests for news-aggregator-bias-analysis scenario

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s"

cd "$TESTING_PHASE_SCENARIO_DIR/api" || testing::phase::end_with_summary "Performance phase skipped"

echo "⚡ Running performance tests..."

if [ -f "performance_test.go" ]; then
    echo "Running Go performance test suite"
    if go test -tags=testing -run=TestPerformance -v -timeout 180s; then
        testing::phase::add_test passed
    else
        testing::phase::add_test failed
    fi

    echo "Running Go benchmarks"
    if go test -tags=testing -bench=. -benchmem -run=^$ -timeout 180s 2>&1 | tee benchmark-results.txt >/dev/null; then
        testing::phase::add_test passed
    else
        testing::phase::add_test failed
    fi
else
    testing::phase::add_warning "⚠️  No performance tests found"
    testing::phase::add_test skipped
fi

cd "$TESTING_PHASE_SCENARIO_DIR"

testing::phase::end_with_summary "Performance tests completed"
