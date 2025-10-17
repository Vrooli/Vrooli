#!/bin/bash
#
# Integration Test Phase for product-manager-agent
# Integrates with centralized Vrooli testing infrastructure
#

set -euo pipefail

# Locate APP_ROOT
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize test phase
testing::phase::init --target-time "120s"

# Navigate to scenario directory
cd "$TESTING_PHASE_SCENARIO_DIR"

echo "🔗 Running integration tests for product-manager-agent..."

# Run Go integration tests
if [[ -d "api" ]]; then
    echo "  📦 Running API integration tests..."
    cd api

    # Run integration tests with timeout
    if go test -v -run "TestFeatureToRoadmapWorkflow|TestSprintPlanningWorkflow|TestCompleteProductPlanningCycle|TestDataConsistencyWorkflow|TestErrorRecoveryWorkflow" -timeout 120s 2>&1 | tee integration-test-output.txt; then
        echo "  ✅ API integration tests passed"
    else
        echo "  ❌ API integration tests failed"
        testing::phase::end_with_summary "Integration tests failed"
        exit 1
    fi

    cd ..
fi

# End test phase with summary
testing::phase::end_with_summary "Integration tests completed successfully"
