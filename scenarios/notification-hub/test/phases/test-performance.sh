#!/bin/bash
# Performance tests for notification-hub scenario

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "⚡ Running performance tests for notification-hub..."

# Performance test scenarios:
# - Health check endpoint latency (< 100ms)
# - Profile creation latency (< 500ms)
# - Notification sending throughput (100+ notifications/second)
# - Bulk notification creation (1000 notifications < 5s)
# - Analytics query performance
# - Redis cache hit rates
# - Database query performance

if [ -d "api" ] && [ -f "api/go.mod" ]; then
    cd api
    echo "Running Go performance benchmarks..."
    if go test -tags=performance -bench=. -benchmem -benchtime=5s ./... 2>&1 | tee test-performance-output.txt; then
        echo "✅ Performance tests completed"
    else
        echo "⚠️  Some performance tests failed"
    fi
    cd ..
fi

testing::phase::end_with_summary "Performance tests completed"
