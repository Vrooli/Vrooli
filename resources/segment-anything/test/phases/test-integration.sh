#!/usr/bin/env bash
# Segment Anything Resource - Integration Test Phase
# Full functionality testing including API endpoints (<120s)

set -euo pipefail

# Get directories
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_DIR
readonly TEST_DIR="$(dirname "${SCRIPT_DIR}")"
readonly RESOURCE_DIR="$(dirname "${TEST_DIR}")"

# Load libraries
source "${RESOURCE_DIR}/config/defaults.sh"
source "${RESOURCE_DIR}/lib/test.sh"

echo "========================================"
echo "Integration Test - Full Functionality"
echo "========================================"
echo "Expected Duration: <120 seconds"
echo "----------------------------------------"

# Set timeout for entire integration test
export INTEGRATION_TEST_TIMEOUT=120

# Ensure service is started for integration testing
if ! docker ps --format "table {{.Names}}" | grep -q "^${SEGMENT_ANYTHING_CONTAINER}$"; then
    echo "Starting service for integration testing..."
    "${RESOURCE_DIR}/cli.sh" manage start --wait || {
        echo "Error: Failed to start service for testing" >&2
        exit 1
    }
fi

# Run with timeout
timeout "${INTEGRATION_TEST_TIMEOUT}" bash -c "run_integration_test" || {
    EXIT_CODE=$?
    if [[ $EXIT_CODE -eq 124 ]]; then
        echo "✗ Integration test exceeded ${INTEGRATION_TEST_TIMEOUT}s timeout"
    fi
    exit $EXIT_CODE
}