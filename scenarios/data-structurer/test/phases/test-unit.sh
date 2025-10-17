#!/bin/bash
# Data Structurer Unit Tests
# Integrates with centralized Vrooli testing infrastructure

set -euo pipefail

# Initialize phase with centralized testing library
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

# Source centralized unit testing runner
source "${APP_ROOT}/scripts/scenarios/testing/unit/run-all.sh"

cd "$TESTING_PHASE_SCENARIO_DIR"

# Run all unit tests with centralized runner
testing::unit::run_all_tests \
    --go-dir "api" \
    --skip-node \
    --skip-python \
    --coverage-warn 80 \
    --coverage-error 50

# End phase with summary
testing::phase::end_with_summary "Unit tests completed"
