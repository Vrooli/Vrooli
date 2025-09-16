#!/usr/bin/env bash
# PaperMC test runner

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_DIR
readonly RESOURCE_DIR="$(dirname "${SCRIPT_DIR}")"

# Source configurations
source "${RESOURCE_DIR}/config/defaults.sh"
source "${RESOURCE_DIR}/lib/core.sh"
source "${RESOURCE_DIR}/lib/test.sh"

# Test type from argument
TEST_TYPE="${1:-all}"

# Run the tests
echo "PaperMC Resource Test Suite"
echo "============================"
echo "Test type: ${TEST_TYPE}"
echo ""

# Execute tests
run_tests "${TEST_TYPE}"
exit_code=$?

echo ""
echo "Test run complete!"
exit ${exit_code}