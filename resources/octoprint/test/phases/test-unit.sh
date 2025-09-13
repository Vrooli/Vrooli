#!/bin/bash

# OctoPrint Unit Test
# Library function validation (< 60 seconds)

set -euo pipefail

# Get the directory of this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$(dirname "${SCRIPT_DIR}")")"

# Source configuration and libraries
source "${RESOURCE_DIR}/config/defaults.sh"
source "${RESOURCE_DIR}/lib/core.sh"
source "${RESOURCE_DIR}/lib/test.sh"

# Run unit test
echo "Starting OctoPrint unit test..."
echo "================================"

# Run the unit test
if test_unit; then
    echo ""
    echo "Unit test: PASS"
    exit 0
else
    echo ""
    echo "Unit test: FAIL"
    exit 1
fi