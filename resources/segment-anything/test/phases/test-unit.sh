#!/usr/bin/env bash
# Segment Anything Resource - Unit Test Phase
# Test library functions and configuration (<60s)

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
echo "Unit Test - Library Functions"
echo "========================================"
echo "Expected Duration: <60 seconds"
echo "----------------------------------------"

# Set timeout for entire unit test
export UNIT_TEST_TIMEOUT=60

# Run with timeout
timeout "${UNIT_TEST_TIMEOUT}" bash -c "run_unit_test" || {
    EXIT_CODE=$?
    if [[ $EXIT_CODE -eq 124 ]]; then
        echo "✗ Unit test exceeded ${UNIT_TEST_TIMEOUT}s timeout"
    fi
    exit $EXIT_CODE
}