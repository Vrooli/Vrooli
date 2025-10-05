#!/bin/bash
#
# Performance Test Phase for product-manager-agent
# Integrates with centralized Vrooli testing infrastructure
#

set -euo pipefail

# Locate APP_ROOT
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize test phase
testing::phase::init --target-time "180s"

# Navigate to scenario directory
cd "$TESTING_PHASE_SCENARIO_DIR"

echo "⚡ Running performance tests for product-manager-agent..."

# Run Go performance tests
if [[ -d "api" ]]; then
    echo "  📊 Running API performance tests..."
    cd api

    # Run performance tests
    if go test -v -run "TestHandlerPerformance|TestConcurrentRequests|TestMemoryUsage" -timeout 180s 2>&1 | tee performance-test-output.txt; then
        echo "  ✅ API performance tests passed"
    else
        echo "  ❌ API performance tests failed"
        testing::phase::end_with_summary "Performance tests failed"
        exit 1
    fi

    # Run benchmarks
    echo "  🏃 Running benchmarks..."
    if go test -bench=. -benchmem -run=^$ 2>&1 | tee benchmark-output.txt; then
        echo "  ✅ Benchmarks completed"
    else
        echo "  ⚠️  Benchmarks failed (non-fatal)"
    fi

    cd ..
fi

# End test phase with summary
testing::phase::end_with_summary "Performance tests completed successfully"
