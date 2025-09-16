#!/bin/bash

# OpenEMS Test Functions
# Implements smoke, integration, and unit tests

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$SCRIPT_DIR")"
TEST_DIR="${RESOURCE_DIR}/test"

# Source configuration
source "${RESOURCE_DIR}/config/defaults.sh"

# ============================================
# Test Execution Functions
# ============================================

test::smoke() {
    echo "🔥 Running OpenEMS smoke tests..."
    
    # Execute smoke test script
    if [[ -f "${TEST_DIR}/phases/test-smoke.sh" ]]; then
        bash "${TEST_DIR}/phases/test-smoke.sh"
        local result=$?
        
        if [[ $result -eq 0 ]]; then
            echo "✅ Smoke tests passed"
        else
            echo "❌ Smoke tests failed"
        fi
        
        return $result
    else
        echo "❌ Smoke test script not found"
        return 1
    fi
}

test::integration() {
    echo "🔗 Running OpenEMS integration tests..."
    
    # Execute integration test script
    if [[ -f "${TEST_DIR}/phases/test-integration.sh" ]]; then
        bash "${TEST_DIR}/phases/test-integration.sh"
        local result=$?
        
        if [[ $result -eq 0 ]]; then
            echo "✅ Integration tests passed"
        else
            echo "❌ Integration tests failed"
        fi
        
        return $result
    else
        echo "❌ Integration test script not found"
        return 1
    fi
}

test::unit() {
    echo "🧪 Running OpenEMS unit tests..."
    
    # Execute unit test script
    if [[ -f "${TEST_DIR}/phases/test-unit.sh" ]]; then
        bash "${TEST_DIR}/phases/test-unit.sh"
        local result=$?
        
        if [[ $result -eq 0 ]]; then
            echo "✅ Unit tests passed"
        else
            echo "❌ Unit tests failed"
        fi
        
        return $result
    else
        echo "❌ Unit test script not found"
        return 1
    fi
}

test::all() {
    echo "🎯 Running all OpenEMS tests..."
    
    local failed=0
    
    # Run smoke tests
    test::smoke || ((failed++))
    
    # Run integration tests
    test::integration || ((failed++))
    
    # Run unit tests
    test::unit || ((failed++))
    
    if [[ $failed -eq 0 ]]; then
        echo "✅ All tests passed"
        return 0
    else
        echo "❌ $failed test suites failed"
        return 1
    fi
}