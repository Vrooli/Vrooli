#!/bin/bash
# Main test runner for Airbyte resource

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Load core functions
source "${RESOURCE_DIR}/lib/core.sh"
source "${RESOURCE_DIR}/lib/test.sh"

# Run test based on argument
test_type="${1:-all}"

case "$test_type" in
    smoke)
        bash "${SCRIPT_DIR}/phases/test-smoke.sh"
        ;;
    integration)
        bash "${SCRIPT_DIR}/phases/test-integration.sh"
        ;;
    unit)
        bash "${SCRIPT_DIR}/phases/test-unit.sh"
        ;;
    all)
        failed=0
        bash "${SCRIPT_DIR}/phases/test-smoke.sh" || failed=$((failed + 1))
        bash "${SCRIPT_DIR}/phases/test-integration.sh" || failed=$((failed + 1))
        bash "${SCRIPT_DIR}/phases/test-unit.sh" || failed=$((failed + 1))
        
        if [[ $failed -gt 0 ]]; then
            echo "ERROR: $failed test suite(s) failed"
            exit 1
        fi
        echo "All tests passed"
        ;;
    *)
        echo "Unknown test type: $test_type"
        echo "Usage: $0 [smoke|integration|unit|all]"
        exit 1
        ;;
esac