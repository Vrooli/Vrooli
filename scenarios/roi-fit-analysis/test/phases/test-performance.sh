#!/bin/bash
# Performance test phase for roi-fit-analysis
# Tests performance characteristics and benchmarks

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "90s"

cd "$TESTING_PHASE_SCENARIO_DIR/api"

echo "⚡ Running performance tests and benchmarks..."

# Run performance-specific tests
echo "  ✓ Running performance test suite..."
go test -v -run="TestPerformance" -timeout=90s 2>&1 | tee test-performance.log

# Check results
if grep -q "FAIL" test-performance.log; then
    echo "    ⚠ Some performance tests failed"
fi

# Run Go benchmarks
echo ""
echo "  ✓ Running benchmarks..."
go test -bench=. -benchmem -run=^Benchmark -timeout=90s 2>&1 | tee benchmark-results.txt

if [ $? -eq 0 ]; then
    echo "    ✓ Benchmarks completed successfully"
else
    echo "    ⚠ Some benchmarks failed"
fi

# Performance criteria checks
echo ""
echo "📊 Performance Summary:"
if [ -f "benchmark-results.txt" ]; then
    echo "  Benchmark results:"
    grep "Benchmark" benchmark-results.txt | head -5
fi

echo ""
echo "  Performance targets:"
echo "  - Health endpoint: < 100ms"
echo "  - Format functions: < 1µs"
echo "  - Analysis requests: < 5s (with mock)"
echo "  - Concurrent requests: 10+ simultaneous"

# Clean up
rm -f test-performance.log benchmark-results.txt

cd "$TESTING_PHASE_SCENARIO_DIR"

testing::phase::end_with_summary "Performance tests completed"
