#!/bin/bash
# Runs lightweight performance smoke checks when benchmarks are available.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

if [ -d "api" ] && command -v go >/dev/null 2>&1 && find api -maxdepth 1 -name "*_test.go" | grep -q .; then
  if testing::phase::timed_exec "Go benchmarks" bash -c 'cd api && go test -bench=. -benchmem -run=^$'; then
    testing::phase::add_test passed
  else
    testing::phase::add_error "Go benchmarks failed"
    testing::phase::add_test failed
  fi
else
  testing::phase::add_warning "No Go benchmarks detected or Go toolchain unavailable; performance phase skipped"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Performance checks completed"
