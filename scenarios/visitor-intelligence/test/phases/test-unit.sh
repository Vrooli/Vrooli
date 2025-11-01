#!/bin/bash
# Unit tests for visitor-intelligence scenario leveraging centralized runners

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "150s"
source "${APP_ROOT}/scripts/scenarios/testing/unit/run-all.sh"

cd "$TESTING_PHASE_SCENARIO_DIR"

if testing::unit::run_all_tests \
    --scenario "visitor-intelligence" \
    --go-dir "api" \
    --skip-node \
    --skip-python \
    --coverage-warn 30 \
    --coverage-error 10; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

testing::phase::end_with_summary "Unit tests completed"
