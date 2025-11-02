#!/bin/bash
set -euo pipefail

# Unit Testing Phase for personal-digital-twin
# This script runs all unit tests for the scenario

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

# Source the centralized unit test runner
source "${APP_ROOT}/scripts/scenarios/testing/unit/run-all.sh"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "🧪 Running unit tests for personal-digital-twin..."

# Run Go tests with coverage
testing::unit::run_all_tests \
    --go-dir "api" \
    --skip-node \
    --skip-python \
    --coverage-warn 80 \
    --coverage-error 50 \
    --scenario "personal-digital-twin"

testing::phase::end_with_summary "Unit tests completed"
