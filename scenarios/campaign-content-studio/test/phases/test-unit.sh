#!/bin/bash
# Unit test phase for campaign-content-studio
# This script integrates with Vrooli's centralized testing infrastructure

set -euo pipefail

# Get the root directory of the Vrooli project
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"

# Source utilities and phase helpers
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase with target time
testing::phase::init --target-time "60s"

# Source centralized unit test runners
source "${APP_ROOT}/scripts/scenarios/testing/unit/run-all.sh"

# Change to scenario directory
cd "$TESTING_PHASE_SCENARIO_DIR"

# Run all unit tests with coverage thresholds
# - Go tests in api/ directory
# - Coverage warning at 80%, error at 50%
testing::unit::run_all_tests \
    --go-dir "api" \
    --skip-node \
    --skip-python \
    --coverage-warn 80 \
    --coverage-error 50

# End phase with summary
testing::phase::end_with_summary "Unit tests completed"
