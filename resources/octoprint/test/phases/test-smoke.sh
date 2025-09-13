#!/bin/bash

# OctoPrint Smoke Test
# Quick health validation (< 30 seconds)

set -euo pipefail

# Get the directory of this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$(dirname "${SCRIPT_DIR}")")"

# Source configuration and libraries
source "${RESOURCE_DIR}/config/defaults.sh"
source "${RESOURCE_DIR}/lib/core.sh"
source "${RESOURCE_DIR}/lib/test.sh"

# Run smoke test
echo "Starting OctoPrint smoke test..."
echo "================================"

# Ensure service is running for tests
if ! octoprint_is_running; then
    echo "Starting OctoPrint for testing..."
    OCTOPRINT_VIRTUAL_PRINTER=true octoprint_start --wait
fi

# Run the smoke test
if test_smoke; then
    echo ""
    echo "Smoke test: PASS"
    exit 0
else
    echo ""
    echo "Smoke test: FAIL"
    exit 1
fi