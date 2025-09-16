#!/usr/bin/env bash
# Segment Anything Resource - Smoke Test Phase
# Quick validation that service is running and responsive (<30s)

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
echo "Smoke Test - Quick Health Validation"
echo "========================================"
echo "Expected Duration: <30 seconds"
echo "----------------------------------------"

# Set timeout for entire smoke test
export SMOKE_TEST_TIMEOUT=30

# Run with timeout
timeout "${SMOKE_TEST_TIMEOUT}" bash -c "run_smoke_test" || {
    EXIT_CODE=$?
    if [[ $EXIT_CODE -eq 124 ]]; then
        echo "✗ Smoke test exceeded ${SMOKE_TEST_TIMEOUT}s timeout"
    fi
    exit $EXIT_CODE
}