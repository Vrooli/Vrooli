#!/bin/bash
# GGWave Smoke Test Phase
# Quick validation that service is operational (<30s)

set -euo pipefail

# Get directories
PHASE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="$(dirname "$PHASE_DIR")"
RESOURCE_DIR="$(dirname "$TEST_DIR")"

# Source configuration and libraries
source "${RESOURCE_DIR}/config/defaults.sh"
source "${RESOURCE_DIR}/lib/core.sh"
source "${RESOURCE_DIR}/lib/test.sh"

# Run smoke tests
echo "========================================="
echo "GGWave Smoke Test Phase"
echo "========================================="
echo "Timeout: 30 seconds"
echo "Purpose: Quick health validation"
echo ""

# Set timeout for entire smoke test
timeout 30 bash -c 'ggwave::test::smoke' || {
    echo "ERROR: Smoke tests exceeded 30 second timeout"
    exit 1
}

exit $?