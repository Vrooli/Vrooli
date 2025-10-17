#!/bin/bash
# Eclipse Ditto Test Runner

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
PHASES_DIR="${SCRIPT_DIR}/phases"

# Default test type
TEST_TYPE="${1:-all}"

# Source the CLI to get access to test functions
source "${RESOURCE_DIR}/cli.sh"

# Run the requested tests
case "$TEST_TYPE" in
    smoke)
        bash "${PHASES_DIR}/test-smoke.sh"
        ;;
    integration)
        bash "${PHASES_DIR}/test-integration.sh"
        ;;
    unit)
        bash "${PHASES_DIR}/test-unit.sh"
        ;;
    all)
        echo "Running all test phases for Eclipse Ditto..."
        echo "============================================="
        
        failed=0
        
        # Run each phase
        bash "${PHASES_DIR}/test-smoke.sh" || failed=$((failed + 1))
        echo ""
        
        bash "${PHASES_DIR}/test-integration.sh" || failed=$((failed + 1))
        echo ""
        
        bash "${PHASES_DIR}/test-unit.sh" || failed=$((failed + 1))
        
        # Report results
        if [[ $failed -gt 0 ]]; then
            echo ""
            echo "❌ Test suite failed: $failed phase(s) had errors"
            exit 1
        else
            echo ""
            echo "✅ All test phases completed successfully"
        fi
        ;;
    *)
        echo "Error: Unknown test type: $TEST_TYPE"
        echo "Usage: $0 [smoke|integration|unit|all]"
        exit 1
        ;;
esac