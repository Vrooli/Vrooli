#!/bin/bash
# Unit test aggregation for app-issue-tracker
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "150s"
source "${APP_ROOT}/scripts/scenarios/testing/unit/run-all.sh"

cd "$TESTING_PHASE_SCENARIO_DIR"

testing::unit::run_all_tests \
  --go-dir "api" \
  --node-dir "ui" \
  --skip-python \
  --coverage-warn 60 \
  --coverage-error 45

testing::phase::end_with_summary "Unit tests completed"
