#!/bin/bash
# Integration test phase for visitor-intelligence scenario
# Tests database and Redis integration

set -euo pipefail

# Get project root
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"

# Source centralized testing utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize test phase
testing::phase::init --target-time "120s"

# Navigate to scenario directory
cd "$TESTING_PHASE_SCENARIO_DIR"

# Check if integration test script exists
if [ -f "test/integration-test.sh" ]; then
    echo "Running integration tests..."
    bash test/integration-test.sh
else
    echo "No integration tests found, skipping..."
fi

# End phase with summary
testing::phase::end_with_summary "Integration tests completed"
