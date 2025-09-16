#!/bin/bash

# GeoNode Test Functions

set -euo pipefail

RESOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Handle test command
handle_test() {
    local phase="${1:-all}"
    
    case "$phase" in
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
            echo "Usage: test [smoke|integration|unit|all]" >&2
            exit 1
            ;;
    esac
}

# Run smoke tests
run_smoke_tests() {
    echo "Running smoke tests..."
    "${RESOURCE_DIR}/test/phases/test-smoke.sh"
}

# Run integration tests
run_integration_tests() {
    echo "Running integration tests..."
    "${RESOURCE_DIR}/test/phases/test-integration.sh"
}

# Run unit tests
run_unit_tests() {
    echo "Running unit tests..."
    "${RESOURCE_DIR}/test/phases/test-unit.sh"
}

# Run all tests
run_all_tests() {
    echo "Running all tests..."
    "${RESOURCE_DIR}/test/run-tests.sh"
}