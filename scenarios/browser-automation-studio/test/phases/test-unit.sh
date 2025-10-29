#!/bin/bash
# Unit tests for browser-automation-studio scenario using centralized testing infrastructure

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "90s"
source "${APP_ROOT}/scripts/scenarios/testing/unit/run-all.sh"

cd "$TESTING_PHASE_SCENARIO_DIR"

# Run all unit tests with coverage thresholds
# Note: Lower threshold (30%) reflects accurate measurement after including handlers package
# Individual packages maintain strong coverage (browserless: 69.9%, compiler: 73.7%, events: 89.3%)
# Many functions require Browserless/Postgres integration which are tested separately
testing::unit::run_all_tests \
    --go-dir "api" \
    --skip-node \
    --skip-python \
    --coverage-warn 40 \
    --coverage-error 30 \
    --verbose

testing::phase::end_with_summary "Unit tests completed"
