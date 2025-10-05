#!/bin/bash
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "90s"

cd "$TESTING_PHASE_SCENARIO_DIR/api"

echo "Running performance tests..."

# Run Go performance tests
go test -v -run "TestPerformance" -timeout 5m

# Run benchmarks
echo ""
echo "Running benchmarks..."
go test -bench=. -benchtime=1s -timeout 5m

testing::phase::end_with_summary "Performance tests completed"
