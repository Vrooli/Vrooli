#!/bin/bash
# Performance testing phase for react-component-library scenario

set -e

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "⚡ Running React Component Library performance tests..."
echo "======================================================="

cd api

echo "🚀 Running performance test suite..."
go test -v -timeout=180s -run "TestPerformance" ./... 2>&1 | tee ../test-performance.log

echo ""
echo "📊 Running benchmarks..."
go test -bench=. -benchmem -benchtime=3s ./... 2>&1 | tee -a ../test-performance.log

echo ""
echo "✅ Performance tests completed"

cd ..

testing::phase::end_with_summary "Performance tests completed"
