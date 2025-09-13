#!/bin/bash

# OctoPrint Integration Test
# Full functionality validation (< 120 seconds)

set -euo pipefail

# Get the directory of this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$(dirname "${SCRIPT_DIR}")")"

# Source configuration and libraries
source "${RESOURCE_DIR}/config/defaults.sh"
source "${RESOURCE_DIR}/lib/core.sh"
source "${RESOURCE_DIR}/lib/test.sh"

# Run integration test
echo "Starting OctoPrint integration test..."
echo "======================================"

# Enable virtual printer for testing
export OCTOPRINT_VIRTUAL_PRINTER=true

# Run the integration test
if test_integration; then
    echo ""
    echo "Integration test: PASS"
    exit 0
else
    echo ""
    echo "Integration test: FAIL"
    exit 1
fi