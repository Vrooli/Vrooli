#!/usr/bin/env bash
# LNbits Test Runner

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TEST_DIR="${SCRIPT_DIR}/test"

# Source test library
source "${SCRIPT_DIR}/lib/test.sh"

# Determine which test phase to run
PHASE="${1:-all}"

case "$PHASE" in
    smoke)
        "${TEST_DIR}/phases/test-smoke.sh"
        ;;
    integration)
        "${TEST_DIR}/phases/test-integration.sh"
        ;;
    unit)
        "${TEST_DIR}/phases/test-unit.sh"
        ;;
    all)
        echo "Running all test phases..."
        "${TEST_DIR}/phases/test-unit.sh"
        "${TEST_DIR}/phases/test-smoke.sh"
        "${TEST_DIR}/phases/test-integration.sh"
        echo "All tests completed successfully!"
        ;;
    *)
        echo "Usage: $0 [smoke|integration|unit|all]"
        exit 1
        ;;
esac