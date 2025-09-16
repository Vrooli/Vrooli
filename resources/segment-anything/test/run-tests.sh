#!/usr/bin/env bash
# Segment Anything Resource - Main Test Runner

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_DIR
readonly RESOURCE_DIR="$(dirname "${SCRIPT_DIR}")"
readonly PHASES_DIR="${SCRIPT_DIR}/phases"

# Load configuration
source "${RESOURCE_DIR}/config/defaults.sh"

# Determine which test to run
TEST_TYPE="${1:-all}"

echo "========================================"
echo "Segment Anything Resource Test Suite"
echo "========================================"
echo "Test Type: ${TEST_TYPE}"
echo "Port: ${SEGMENT_ANYTHING_PORT}"
echo "Model: ${SEGMENT_ANYTHING_MODEL_TYPE}-${SEGMENT_ANYTHING_MODEL_SIZE}"
echo "----------------------------------------"

# Run the appropriate test
case "${TEST_TYPE}" in
    smoke)
        bash "${PHASES_DIR}/test-smoke.sh"
        ;;
    integration)
        bash "${PHASES_DIR}/test-integration.sh"
        ;;
    unit)
        bash "${PHASES_DIR}/test-unit.sh"
        ;;
    all)
        # Run all tests in sequence
        EXIT_CODE=0
        
        echo -e "\n>>> Running Unit Tests..."
        bash "${PHASES_DIR}/test-unit.sh" || EXIT_CODE=$?
        
        echo -e "\n>>> Running Smoke Tests..."
        bash "${PHASES_DIR}/test-smoke.sh" || EXIT_CODE=$?
        
        echo -e "\n>>> Running Integration Tests..."
        bash "${PHASES_DIR}/test-integration.sh" || EXIT_CODE=$?
        
        echo -e "\n========================================"
        if [[ $EXIT_CODE -eq 0 ]]; then
            echo "✓ All test phases completed successfully"
        else
            echo "✗ Some test phases failed"
        fi
        exit $EXIT_CODE
        ;;
    *)
        echo "Error: Unknown test type: ${TEST_TYPE}" >&2
        echo "Valid types: smoke, integration, unit, all" >&2
        exit 1
        ;;
esac