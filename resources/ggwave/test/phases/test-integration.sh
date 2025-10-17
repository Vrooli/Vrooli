#!/bin/bash
# GGWave Integration Test Phase
# Full functionality validation (<120s)

set -euo pipefail

# Get directories
PHASE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="$(dirname "$PHASE_DIR")"
RESOURCE_DIR="$(dirname "$TEST_DIR")"

# Source configuration and libraries
source "${RESOURCE_DIR}/config/defaults.sh"
source "${RESOURCE_DIR}/lib/core.sh"
source "${RESOURCE_DIR}/lib/test.sh"

# Run integration tests
echo "========================================="
echo "GGWave Integration Test Phase"
echo "========================================="
echo "Timeout: 120 seconds"
echo "Purpose: End-to-end functionality validation"
echo ""

# Set timeout for entire integration test
timeout 120 bash -c 'ggwave::test::integration' || {
    echo "ERROR: Integration tests exceeded 120 second timeout"
    exit 1
}

exit $?