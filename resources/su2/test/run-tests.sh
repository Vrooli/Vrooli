#!/bin/bash
# SU2 Test Runner

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$SCRIPT_DIR")"

# Source CLI for test functions
source "${RESOURCE_DIR}/cli.sh"

# Run tests based on argument
phase="${1:-all}"

case "$phase" in
    smoke)
        "${RESOURCE_DIR}/test/phases/test-smoke.sh"
        ;;
    integration)
        "${RESOURCE_DIR}/test/phases/test-integration.sh"
        ;;
    unit)
        "${RESOURCE_DIR}/test/phases/test-unit.sh"
        ;;
    all)
        failed=0
        
        echo "Running all test phases..."
        echo
        
        "${RESOURCE_DIR}/test/phases/test-unit.sh" || ((failed++))
        echo
        
        "${RESOURCE_DIR}/test/phases/test-smoke.sh" || ((failed++))
        echo
        
        "${RESOURCE_DIR}/test/phases/test-integration.sh" || ((failed++))
        
        if [[ $failed -eq 0 ]]; then
            echo -e "\n✓ All tests passed"
            exit 0
        else
            echo -e "\n✗ $failed test phase(s) failed"
            exit 1
        fi
        ;;
    *)
        echo "Usage: $0 [smoke|integration|unit|all]"
        exit 1
        ;;
esac