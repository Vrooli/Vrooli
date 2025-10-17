#!/bin/bash
# Performance test phase for travel-map-filler
# Tests API endpoint performance and scalability

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "⚡ Running performance tests..."

# Run Go performance tests
if [ -d "api" ] && [ -f "api/performance_test.go" ]; then
    echo "Running API performance tests..."
    cd api
    go test -v -run "TestHandlerPerformance|TestThroughput|TestMemoryUsage|TestScalability" \
        -timeout 180s \
        -coverprofile=performance_coverage.out 2>&1 | tee performance_test.log

    PERFORMANCE_EXIT_CODE=${PIPESTATUS[0]}

    if [ $PERFORMANCE_EXIT_CODE -eq 0 ]; then
        echo "✅ Performance tests passed"
    else
        echo "❌ Performance tests failed with exit code $PERFORMANCE_EXIT_CODE"
    fi

    # Extract performance metrics
    echo ""
    echo "📈 Performance Summary:"
    grep -E "req/s|completed in|throughput" performance_test.log || echo "No performance metrics found"

    cd ..
else
    echo "⚠️  No performance tests found"
    PERFORMANCE_EXIT_CODE=0
fi

testing::phase::end_with_summary "Performance tests completed"

exit $PERFORMANCE_EXIT_CODE
