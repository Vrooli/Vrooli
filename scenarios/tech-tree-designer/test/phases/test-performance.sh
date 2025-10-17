#!/bin/bash
# Performance testing phase for tech-tree-designer
# Benchmarks API endpoints and database operations

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"

cd "$TESTING_PHASE_SCENARIO_DIR"

# Run Go performance tests
testing::phase::log "Running performance tests..."
cd api
go test -tags testing -run "^TestPerformance_" -v -timeout 90s

if [ $? -ne 0 ]; then
    testing::phase::error "Performance tests failed"
    testing::phase::end_with_summary "Performance tests failed" 1
fi

# Run Go benchmarks
testing::phase::log "Running benchmarks..."
go test -tags testing -bench=. -benchtime=1s -run=^$ -timeout 90s

testing::phase::end_with_summary "Performance tests completed"
