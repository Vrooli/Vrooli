#!/bin/bash
# Unit testing phase for quiz-generator scenario
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"

source "${APP_ROOT}/scripts/scenarios/testing/unit/run-all.sh"

cd "$TESTING_PHASE_SCENARIO_DIR"

export SKIP_OLLAMA_TESTS=1

if testing::unit::run_all_tests --go-dir "api" --skip-node --skip-python --coverage-warn 80 --coverage-error 50; then
  testing::phase::add_test passed
else
  testing::phase::add_error "Unit tests failed"
  testing::phase::add_test failed
  testing::phase::end_with_summary "Unit tests failed"
fi

testing::phase::end_with_summary "Unit tests completed"
