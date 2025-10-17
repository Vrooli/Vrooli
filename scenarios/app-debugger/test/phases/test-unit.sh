#!/bin/bash
# Unit test phase for app-debugger
# Integrates with centralized testing infrastructure

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase with target time
testing::phase::init --target-time "60s"

# Source centralized unit test runner
source "${APP_ROOT}/scripts/scenarios/testing/unit/run-all.sh"

# Change to scenario directory
cd "$TESTING_PHASE_SCENARIO_DIR"

# Run all unit tests with coverage
testing::unit::run_all_tests \
    --go-dir "api" \
    --skip-node \
    --skip-python \
    --coverage-warn 80 \
    --coverage-error 50

# End phase with summary
testing::phase::end_with_summary "Unit tests completed"