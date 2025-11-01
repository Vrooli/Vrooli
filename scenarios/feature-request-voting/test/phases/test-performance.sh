#!/bin/bash
# Executes performance-oriented Go tests and benchmarks

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s"

cd "$TESTING_PHASE_SCENARIO_DIR"

if [ -d "api" ]; then
  testing::phase::check "Go performance test suite" bash -c "cd api && GOFLAGS=\"\${GOFLAGS:-} -tags=testing\" go test -run 'TestPerformance|TestConcurrentRequests' -count=1 -timeout 5m ./..."
  testing::phase::check "Go benchmarks" bash -c "cd api && GOFLAGS=\"\${GOFLAGS:-} -tags=testing\" go test -run '^$' -bench . -benchmem -count=1 ./..."
else
  testing::phase::add_warning "API directory missing; skipping performance tests"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Performance checks completed"
