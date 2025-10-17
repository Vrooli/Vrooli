#!/bin/bash
#
# Unit Test Phase for product-manager-agent
# Integrates with centralized Vrooli testing infrastructure
#

set -euo pipefail

# Locate APP_ROOT
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize test phase
testing::phase::init --target-time "60s"

# Source centralized unit test runner
source "${APP_ROOT}/scripts/scenarios/testing/unit/run-all.sh"

# Navigate to scenario directory
cd "$TESTING_PHASE_SCENARIO_DIR"

echo "🧪 Running unit tests for product-manager-agent..."

# Run all unit tests with coverage
testing::unit::run_all_tests \
    --go-dir "api" \
    --skip-node \
    --skip-python \
    --coverage-warn 80 \
    --coverage-error 50

# End test phase with summary
testing::phase::end_with_summary "Unit tests completed"
