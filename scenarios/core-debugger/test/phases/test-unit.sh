#!/bin/bash
# Unit tests for core-debugger scenario
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

source "${APP_ROOT}/scripts/scenarios/testing/unit/run-all.sh"

cd "$TESTING_PHASE_SCENARIO_DIR"

if find api -name '*_test.go' -print -quit | grep -q '.'; then
  if testing::unit::run_all_tests \
      --go-dir "api" \
      --skip-node \
      --skip-python \
      --coverage-warn 0 \
      --coverage-error 0; then
    testing::phase::add_test passed
  else
    testing::phase::add_test failed
    testing::phase::end_with_summary "Unit tests failed"
  fi
else
  testing::phase::add_warning "No Go unit tests found; skipping unit phase"
  testing::phase::add_test skipped
  testing::phase::end_with_summary "Unit tests skipped"
fi

testing::phase::end_with_summary "Unit tests completed"
