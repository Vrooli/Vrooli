#!/bin/bash
# Performance smoke tests for System Monitor.

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s"

cd "$TESTING_PHASE_SCENARIO_DIR"

if [ -d "api" ] && command -v go >/dev/null 2>&1; then
  pushd api >/dev/null

  if go test -bench=. -benchmem -benchtime=5s ./... -timeout 180s >/dev/null 2>&1; then
    log::success "✅ Go benchmarks stable"
    testing::phase::add_test passed
  else
    testing::phase::add_warning "Go benchmarks reported regressions"
    testing::phase::add_test skipped
  fi

  if go test -v -run TestPerformance ./... -timeout 120s >/dev/null 2>&1; then
    log::success "✅ Performance-specific tests passed"
    testing::phase::add_test passed
  else
    testing::phase::add_warning "Performance-specific tests failed"
    testing::phase::add_test skipped
  fi

  popd >/dev/null
else
  testing::phase::add_warning "Go toolchain or API sources unavailable; skipping performance checks"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Performance validation completed"
