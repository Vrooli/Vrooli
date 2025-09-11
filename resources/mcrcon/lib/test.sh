#!/usr/bin/env bash
# mcrcon test implementation

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_DIR
readonly RESOURCE_DIR="$(dirname "$SCRIPT_DIR")"
readonly TEST_DIR="${RESOURCE_DIR}/test"

# Source configuration
source "${RESOURCE_DIR}/config/defaults.sh"

# Execute test phase
run_test() {
    local test_type="${1:-all}"
    
    case "$test_type" in
        smoke)
            echo "Running smoke tests..."
            bash "${TEST_DIR}/phases/test-smoke.sh"
            ;;
        integration)
            echo "Running integration tests..."
            bash "${TEST_DIR}/phases/test-integration.sh"
            ;;
        unit)
            echo "Running unit tests..."
            bash "${TEST_DIR}/phases/test-unit.sh"
            ;;
        all)
            echo "Running all tests..."
            bash "${TEST_DIR}/run-tests.sh"
            ;;
        *)
            echo "Unknown test type: $test_type" >&2
            echo "Valid types: smoke, integration, unit, all" >&2
            exit 1
            ;;
    esac
}

# Execute if called directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    run_test "$@"
fi