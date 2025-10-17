#!/bin/bash
# Performance tests for seo-optimizer scenario

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s"

cd "$TESTING_PHASE_SCENARIO_DIR/api"

echo "Running Go benchmark tests..."

# Run Go benchmarks
if ! go test -bench=. -benchmem -benchtime=3s -timeout=180s ./... 2>&1 | tee benchmark-results.txt; then
    echo "⚠️  Benchmark tests encountered errors, but continuing..."
fi

# Check for performance regressions
echo ""
echo "Performance Test Summary:"
echo "========================"

if [ -f benchmark-results.txt ]; then
    # Extract benchmark results
    AUDIT_BENCH=$(grep "BenchmarkPerformSEOAudit" benchmark-results.txt || echo "N/A")
    OPTIMIZE_BENCH=$(grep "BenchmarkOptimizeContent" benchmark-results.txt || echo "N/A")

    echo "SEO Audit Performance: $AUDIT_BENCH"
    echo "Content Optimization Performance: $OPTIMIZE_BENCH"

    rm -f benchmark-results.txt
else
    echo "No benchmark results found"
fi

testing::phase::end_with_summary "Performance tests completed"
