#!/bin/bash
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"

cd "$TESTING_PHASE_SCENARIO_DIR"

if command -v go >/dev/null 2>&1; then
  testing::phase::check "Go performance tests" bash -c 'cd api && go test -run "^TestPerformance" -timeout=120s ./...'
  testing::phase::check "Go benchmarks" bash -c 'cd api && go test -bench . -benchmem -run ^$ ./...'
else
  testing::phase::add_warning "Go toolchain not found; skipping performance suite"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Performance checks completed"
