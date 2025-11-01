#!/bin/bash
# Run calendar unit tests via the shared unit test orchestration.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "240s"
source "${APP_ROOT}/scripts/scenarios/testing/unit/run-all.sh"

cd "$TESTING_PHASE_SCENARIO_DIR"

testing::unit::run_all_tests \
  --go-dir "api" \
  --skip-node \
  --skip-python \
  --coverage-warn 10 \
  --coverage-error 1 \
  --verbose

testing::phase::end_with_summary "Unit tests completed"
