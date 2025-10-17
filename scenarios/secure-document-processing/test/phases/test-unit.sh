#!/bin/bash
set -euo pipefail

# Get APP_ROOT - navigate up from test/phases to project root
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"

# Source utilities and phase helpers
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize test phase
testing::phase::init --target-time "60s"

# Source centralized unit test runner
source "${APP_ROOT}/scripts/scenarios/testing/unit/run-all.sh"

# Navigate to scenario directory
cd "$TESTING_PHASE_SCENARIO_DIR"

echo "🧪 Running Secure Document Processing unit tests"

# Run all unit tests using centralized testing infrastructure
testing::unit::run_all_tests \
    --go-dir "api" \
    --node-dir "ui" \
    --skip-python \
    --coverage-warn 80 \
    --coverage-error 50 \
    --verbose

# End test phase with summary
testing::phase::end_with_summary "Unit tests completed"
