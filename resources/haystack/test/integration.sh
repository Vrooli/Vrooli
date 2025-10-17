#!/usr/bin/env bash
# Haystack integration tests

set -euo pipefail

# Test directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
TEST_DIR="${APP_ROOT}/resources/haystack/test"
HAYSTACK_DIR="${APP_ROOT}/resources/haystack"

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

# Simple test function
run_test() {
    local test_name="$1"
    shift
    
    echo -n "Testing $test_name... "
    
    if "$@" >/dev/null 2>&1; then
        echo "✓"
        ((TESTS_PASSED++))
    else
        echo "✗"
        ((TESTS_FAILED++))
    fi
}

# Main test suite
main() {
    echo "🧪 Haystack Integration Tests"
    echo "============================"
    
    # Test 1: Check if Haystack CLI exists
    run_test "CLI exists" which resource-haystack
    
    # Test 2: Check if Haystack is healthy via status
    run_test "Status check" resource-haystack status
    
    # Test 3: Health check endpoint
    run_test "Health endpoint" curl -s -f http://localhost:8075/health
    
    # Test 4: Check API availability
    run_test "API endpoint" curl -s -f http://localhost:8075/
    
    # Test 5: Check stats endpoint
    run_test "Stats endpoint" curl -s -f http://localhost:8075/stats
    
    # Test 6: Check data directory
    run_test "Data directory" test -d "$HOME/.vrooli/haystack"
    
    # Test 7: Help command works
    run_test "Help command" resource-haystack help
    
    # Test 8: Check Vrooli registration
    run_test "Vrooli integration" vrooli resource list
    
    # Test 9: Simple document test via API
    run_test "Document API" curl -s -X POST http://localhost:8075/documents \
        -H "Content-Type: application/json" \
        -d '{"text": "Test document"}'
    
    # Summary
    echo ""
    echo "📊 Test Results:"
    echo "   ✅ Passed: $TESTS_PASSED"
    
    if [ "$TESTS_FAILED" -gt 0 ]; then
        echo "   ❌ Failed: $TESTS_FAILED"
        exit 1
    else
        echo "   All tests passed!"
        
        # Update last test timestamp
        echo "$(date -Iseconds)" > "$HAYSTACK_DIR/test/.last_test_run"
    fi
}

# Run tests
main "$@"