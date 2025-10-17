#!/usr/bin/env bash
# QwenCoder Test Runner

set -euo pipefail

# Test directory setup
readonly TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly RESOURCE_DIR="$(dirname "${TEST_DIR}")"
readonly PHASES_DIR="${TEST_DIR}/phases"

# Source configuration
source "${RESOURCE_DIR}/config/defaults.sh"
source "${RESOURCE_DIR}/lib/core.sh"
source "${RESOURCE_DIR}/lib/test.sh"

# Test phase selection
readonly TEST_PHASE="${1:-all}"

# Execute test phase
case "${TEST_PHASE}" in
    smoke)
        echo "=== Running Smoke Tests ==="
        bash "${PHASES_DIR}/test-smoke.sh"
        ;;
    integration)
        echo "=== Running Integration Tests ==="
        bash "${PHASES_DIR}/test-integration.sh"
        ;;
    unit)
        echo "=== Running Unit Tests ==="
        bash "${PHASES_DIR}/test-unit.sh"
        ;;
    all)
        echo "=== Running All Tests ==="
        failed=0
        
        # Run each phase
        for phase in smoke integration unit; do
            echo ""
            echo "--- ${phase} tests ---"
            if ! bash "${PHASES_DIR}/test-${phase}.sh"; then
                echo "✗ ${phase} tests failed"
                ((failed++))
            else
                echo "✓ ${phase} tests passed"
            fi
        done
        
        # Summary
        echo ""
        echo "=== Test Summary ==="
        if [[ ${failed} -eq 0 ]]; then
            echo "✓ All tests passed!"
            exit 0
        else
            echo "✗ ${failed} test phase(s) failed"
            exit 1
        fi
        ;;
    *)
        echo "Error: Unknown test phase '${TEST_PHASE}'"
        echo "Valid phases: smoke, integration, unit, all"
        exit 1
        ;;
esac