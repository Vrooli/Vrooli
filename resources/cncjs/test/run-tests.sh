#!/usr/bin/env bash
################################################################################
# CNCjs Test Runner
# Main test orchestrator following v2.0 contract
################################################################################

set -euo pipefail

# Determine paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Source test library
source "${RESOURCE_DIR}/lib/test.sh"

# Parse arguments
TEST_TYPE="${1:-all}"

# Run tests
case "$TEST_TYPE" in
    smoke|integration|unit|all)
        cncjs::test "$TEST_TYPE"
        exit $?
        ;;
    *)
        echo "Usage: $0 [smoke|integration|unit|all]"
        echo ""
        echo "Test types:"
        echo "  smoke       - Quick health check (<30s)"
        echo "  integration - Full functionality test (<120s)"
        echo "  unit        - Library function tests (<60s)"
        echo "  all         - Run all tests"
        exit 1
        ;;
esac