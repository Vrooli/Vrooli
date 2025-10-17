#!/bin/bash
# Traccar Test Library Functions

set -euo pipefail

# Test runner
test_run() {
    local test_type="${1:-all}"
    local test_dir="${RESOURCE_DIR}/test"
    
    case "$test_type" in
        smoke)
            bash "${test_dir}/phases/test-smoke.sh"
            ;;
        integration)
            bash "${test_dir}/phases/test-integration.sh"
            ;;
        unit)
            bash "${test_dir}/phases/test-unit.sh"
            ;;
        all)
            bash "${test_dir}/run-tests.sh" all
            ;;
        *)
            echo "Unknown test type: $test_type"
            exit 1
            ;;
    esac
}