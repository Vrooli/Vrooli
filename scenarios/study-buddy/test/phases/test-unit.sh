#!/bin/bash
# Unit tests for study-buddy scenario
# Integrates with Vrooli's centralized testing infrastructure

set -euo pipefail

# Determine app root
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize test phase
testing::phase::init --target-time "60s"

# Source centralized testing runners
source "${APP_ROOT}/scripts/scenarios/testing/unit/run-all.sh"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "🧪 Running study-buddy unit tests..."

# Run all tests with coverage and record the outcome for phase reporting
if testing::unit::run_all_tests \
    --go-dir "api" \
    --skip-node \
    --skip-python \
    --coverage-warn 80 \
    --coverage-error 50; then
    testing::phase::add_test passed
else
    testing::phase::add_test failed
    testing::phase::end_with_summary "Unit test suite reported failures"
fi

testing::phase::end_with_summary "Unit tests completed"
