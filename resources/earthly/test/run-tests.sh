#!/bin/bash
# Earthly Resource - Main Test Runner
# Delegates to test phases according to v2.0 contract

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Source test library
source "${SCRIPT_DIR}/lib/test.sh"

# Main test execution
main() {
    local test_type="${1:-all}"
    
    case "${test_type}" in
        smoke|integration|unit|all)
            handle_test "${test_type}"
            ;;
        *)
            echo "[ERROR] Unknown test type: ${test_type}"
            echo "Usage: $0 [smoke|integration|unit|all]"
            exit 1
            ;;
    esac
}

main "$@"