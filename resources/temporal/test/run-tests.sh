#!/usr/bin/env bash
# Temporal Resource - Main Test Runner

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$SCRIPT_DIR")"

# Source test library
source "${RESOURCE_DIR}/lib/test.sh"

# Main test execution
main() {
    local test_type="${1:-all}"
    
    case "$test_type" in
        smoke)
            run_smoke_tests
            ;;
        integration)
            run_integration_tests
            ;;
        unit)
            run_unit_tests
            ;;
        all)
            run_all_tests
            ;;
        *)
            echo "Unknown test type: $test_type"
            echo "Valid types: smoke, integration, unit, all"
            exit 1
            ;;
    esac
}

main "$@"