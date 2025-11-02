#!/bin/bash
# Execute unit tests for accessibility-compliance-hub using shared runners.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "90s"
source "${APP_ROOT}/scripts/scenarios/testing/unit/run-all.sh"

cd "$TESTING_PHASE_SCENARIO_DIR"

if testing::unit::run_all_tests \
  --go-dir "api" \
  --skip-node \
  --skip-python \
  --coverage-warn 70 \
  --coverage-error 50 \
  --scenario "$TESTING_PHASE_SCENARIO_NAME"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
  testing::phase::end_with_summary "Unit tests failed"
fi

testing::phase::end_with_summary "Unit tests completed"
