#!/bin/bash
# Performance baseline checks for travel-map-filler.

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

cd "$TESTING_PHASE_SCENARIO_DIR"

log::info "🚀 Running performance baselines"

if [ -f "api/performance_test.go" ]; then
  if (cd api && go test -v -run "TestPerformance" -timeout 60s -coverprofile=performance_coverage.out); then
    testing::phase::add_test passed
    log::success "✅ Performance tests passed"
  else
    testing::phase::add_error "Performance tests failed"
    testing::phase::add_test failed
  fi
else
  log::warning "ℹ️  No performance test suite; skipping"
  testing::phase::add_warning "Performance suite missing"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Performance validation completed"
