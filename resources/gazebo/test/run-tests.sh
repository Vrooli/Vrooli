#!/usr/bin/env bash
# Gazebo Test Runner

set -euo pipefail

# Get the directory of this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Source test library
source "${SCRIPT_DIR}/lib/test.sh"

# Main test runner
main() {
    local suite="${1:-all}"
    
    echo "================================"
    echo "Gazebo Resource Test Suite"
    echo "================================"
    echo ""
    
    case "$suite" in
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
            echo "Unknown test suite: $suite"
            echo "Valid suites: smoke, integration, unit, all"
            exit 1
            ;;
    esac
    
    local exit_code=$?
    
    echo ""
    echo "================================"
    if [[ $exit_code -eq 0 ]]; then
        echo "Test Suite: PASSED"
    else
        echo "Test Suite: FAILED"
    fi
    echo "================================"
    
    exit $exit_code
}

# Run tests
main "$@"