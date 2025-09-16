#!/usr/bin/env bash
# OpenTripPlanner Test Functions

set -euo pipefail

# Test commands delegation

opentripplanner::test::all() {
    echo "Running all OpenTripPlanner tests..."
    
    # Delegate to test runner
    "${SCRIPT_DIR}/test/run-tests.sh" all
}

opentripplanner::test::smoke() {
    echo "Running smoke tests..."
    
    # Delegate to smoke test phase
    "${SCRIPT_DIR}/test/phases/test-smoke.sh"
}

opentripplanner::test::integration() {
    echo "Running integration tests..."
    
    # Delegate to integration test phase
    "${SCRIPT_DIR}/test/phases/test-integration.sh"
}

opentripplanner::test::unit() {
    echo "Running unit tests..."
    
    # Delegate to unit test phase
    "${SCRIPT_DIR}/test/phases/test-unit.sh"
}