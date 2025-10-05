#!/bin/bash
# Email Outreach Manager - Unit Tests
# This script integrates with Vrooli's centralized testing infrastructure

set -euo pipefail

# Locate Vrooli root directory
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"

# Source centralized testing libraries
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize test phase
testing::phase::init --target-time "60s"

# Source centralized unit test runners
source "${APP_ROOT}/scripts/scenarios/testing/unit/run-all.sh"

# Navigate to scenario directory
cd "$TESTING_PHASE_SCENARIO_DIR"

echo "======================================"
echo "Email Outreach Manager - Unit Tests"
echo "======================================"

# Run all unit tests with centralized infrastructure
testing::unit::run_all_tests \
    --go-dir "api" \
    --skip-node \
    --skip-python \
    --coverage-warn 80 \
    --coverage-error 50

# End test phase with summary
testing::phase::end_with_summary "Unit tests completed"
