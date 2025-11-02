#!/bin/bash
# Run Go/Node unit tests using centralized runner

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s"
source "${APP_ROOT}/scripts/scenarios/testing/unit/run-all.sh"

cd "$TESTING_PHASE_SCENARIO_DIR"

testing::unit::run_all_tests \
  --scenario "$TESTING_PHASE_SCENARIO_NAME" \
  --go-dir "api" \
  --node-dir "ui" \
  --skip-python \
  --coverage-warn 75 \
  --coverage-error 50

testing::phase::end_with_summary "Unit tests completed"
