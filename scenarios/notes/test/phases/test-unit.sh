#!/usr/bin/env bash
# Unit test orchestration using centralized testing infrastructure
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"
source "${APP_ROOT}/scripts/scenarios/testing/unit/run-all.sh"

testing::phase::init --target-time "120s"

cd "$TESTING_PHASE_SCENARIO_DIR"

testing::unit::run_all_tests \
  --scenario "notes" \
  --go-dir "api" \
  --node-dir "ui" \
  --skip-python \
  --coverage-warn 70 \
  --coverage-error 50

testing::phase::add_test passed

testing::phase::end_with_summary "Unit tests completed"
