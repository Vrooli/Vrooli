#!/usr/bin/env bash
# farmOS Test Library Functions

# Test all phases
farmos::test::all() {
    echo "Running all farmOS tests..."
    
    # Delegate to test runner
    if [[ -f "${RESOURCE_DIR}/test/run-tests.sh" ]]; then
        "${RESOURCE_DIR}/test/run-tests.sh" all
    else
        echo "Error: Test runner not found"
        return 1
    fi
}

# Smoke test - quick health check
farmos::test::smoke() {
    echo "Running farmOS smoke tests..."
    
    # Delegate to smoke test script
    if [[ -f "${RESOURCE_DIR}/test/phases/test-smoke.sh" ]]; then
        "${RESOURCE_DIR}/test/phases/test-smoke.sh"
    else
        echo "Error: Smoke test script not found"
        return 1
    fi
}

# Integration test - full functionality
farmos::test::integration() {
    echo "Running farmOS integration tests..."
    
    # Delegate to integration test script
    if [[ -f "${RESOURCE_DIR}/test/phases/test-integration.sh" ]]; then
        "${RESOURCE_DIR}/test/phases/test-integration.sh"
    else
        echo "Error: Integration test script not found"
        return 1
    fi
}

# Unit test - library functions
farmos::test::unit() {
    echo "Running farmOS unit tests..."
    
    # Delegate to unit test script
    if [[ -f "${RESOURCE_DIR}/test/phases/test-unit.sh" ]]; then
        "${RESOURCE_DIR}/test/phases/test-unit.sh"
    else
        echo "No unit tests available"
        return 2
    fi
}