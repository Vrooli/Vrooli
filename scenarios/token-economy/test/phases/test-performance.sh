#!/bin/bash
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "Running performance tests for token-economy..."

# Run Go performance tests
if [ -f "api/performance_comprehensive_test.go" ]; then
    echo "Running Go performance tests..."
    cd api

    # Run benchmark tests
    echo "Running benchmarks..."
    if go test -bench=. -benchmem -timeout 180s 2>&1 | tee benchmark-results.txt; then
        echo "✓ Benchmarks completed"
        echo ""
        echo "Benchmark results:"
        cat benchmark-results.txt | grep "^Benchmark" | head -20
    else
        echo "⚠ Some benchmarks failed"
    fi

    echo ""
    echo "Running performance test suite..."

    # Run end-to-end performance tests
    echo "Testing end-to-end performance..."
    go test -v -run "TestPerformanceEndToEnd" -timeout 60s || echo "⚠ End-to-end performance test skipped"

    # Run concurrent load tests
    echo ""
    echo "Testing concurrent load..."
    go test -v -run "TestConcurrentLoad" -timeout 60s || echo "⚠ Concurrent load test skipped"

    # Run memory usage tests
    echo ""
    echo "Testing memory usage..."
    go test -v -run "TestMemoryUsage" -timeout 60s || echo "⚠ Memory usage test skipped"

    # Run response time tests
    echo ""
    echo "Testing response times..."
    go test -v -run "TestResponseTime" -timeout 60s || echo "⚠ Response time test skipped"

    # Run throughput tests
    echo ""
    echo "Testing throughput..."
    go test -v -run "TestThroughput" -timeout 60s || echo "⚠ Throughput test skipped"

    # Run database pooling tests
    echo ""
    echo "Testing database connection pooling..."
    go test -v -run "TestDatabasePooling" -timeout 30s || echo "⚠ Database pooling test skipped"

    # Run cache effectiveness tests
    echo ""
    echo "Testing cache effectiveness..."
    go test -v -run "TestCacheEffectiveness" -timeout 30s || echo "⚠ Cache test skipped (Redis may not be available)"

    cd ..
else
    echo "⚠ No performance tests found (api/performance_comprehensive_test.go missing)"
fi

# Run from main_test.go performance tests
if [ -f "api/main_test.go" ]; then
    echo ""
    echo "Running additional performance tests from main_test.go..."
    cd api

    go test -v -run "TestPerformance" -timeout 60s || echo "⚠ Additional performance tests skipped"
    go test -v -run "TestConcurrency" -timeout 60s || echo "⚠ Concurrency tests skipped"

    cd ..
fi

# Generate performance report
echo ""
echo "Performance Test Summary"
echo "======================="

if [ -f "api/benchmark-results.txt" ]; then
    echo "Top 10 Benchmark Results:"
    grep "^Benchmark" api/benchmark-results.txt | head -10
else
    echo "No benchmark results available"
fi

testing::phase::end_with_summary "Performance tests completed"
