#!/bin/bash
set -e

# Get the application root directory
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"

# Source required libraries
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize the test phase
testing::phase::init --target-time "60s"

# Source centralized testing runners
source "${APP_ROOT}/scripts/scenarios/testing/unit/run-all.sh"

# Change to scenario directory
cd "$TESTING_PHASE_SCENARIO_DIR"

# Run all unit tests with coverage thresholds
testing::unit::run_all_tests \
    --go-dir "api" \
    --skip-node \
    --skip-python \
    --coverage-warn 80 \
    --coverage-error 50

# End the test phase with summary
testing::phase::end_with_summary "Unit tests completed"
