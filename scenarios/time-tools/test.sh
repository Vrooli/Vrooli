#!/bin/bash
# ====================================================================
# Time Tools Integration Test
# ====================================================================
#
# This test validates time-tools scenario functionality
#
# ====================================================================

set -euo pipefail

# Get script directory (test.sh is in the scenario root, not test/)
SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="${SCENARIO_DIR}/test"

# Load scenario configuration
SERVICE_FILE="${SCENARIO_DIR}/service.json"

# Simple test runner - just run the API tests
echo "Running time-tools integration tests..."

# Run API endpoint tests
if [[ -f "${TEST_DIR}/test-api-endpoints.sh" ]]; then
    bash "${TEST_DIR}/test-api-endpoints.sh"
else
    echo "API endpoint tests not found at ${TEST_DIR}/test-api-endpoints.sh, skipping"
fi

exit 0
