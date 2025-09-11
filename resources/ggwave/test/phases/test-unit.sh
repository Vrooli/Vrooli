#!/bin/bash
# GGWave Unit Test Phase
# Library function validation (<60s)

set -euo pipefail

# Get directories
PHASE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="$(dirname "$PHASE_DIR")"
RESOURCE_DIR="$(dirname "$TEST_DIR")"

# Source configuration and libraries
source "${RESOURCE_DIR}/config/defaults.sh"
source "${RESOURCE_DIR}/lib/core.sh"
source "${RESOURCE_DIR}/lib/test.sh"

# Run unit tests
echo "========================================="
echo "GGWave Unit Test Phase"
echo "========================================="
echo "Timeout: 60 seconds"
echo "Purpose: Library function validation"
echo ""

# Set timeout for entire unit test
timeout 60 bash -c 'ggwave::test::unit' || {
    echo "ERROR: Unit tests exceeded 60 second timeout"
    exit 1
}

exit $?