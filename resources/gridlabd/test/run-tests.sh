#!/usr/bin/env bash
# GridLAB-D Resource - Main Test Runner

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Source test library
source "${SCRIPT_DIR}/lib/test.sh"

# Determine which phase to run
PHASE="${1:-all}"

case "$PHASE" in
    smoke)
        test_smoke
        ;;
    integration)
        test_integration
        ;;
    unit)
        test_unit
        ;;
    all)
        test_all
        ;;
    *)
        echo "Usage: $0 [smoke|integration|unit|all]"
        echo "  smoke       - Quick health validation (<30s)"
        echo "  integration - End-to-end functionality tests"
        echo "  unit        - Library function tests"
        echo "  all         - Run all test phases"
        exit 1
        ;;
esac

exit $?