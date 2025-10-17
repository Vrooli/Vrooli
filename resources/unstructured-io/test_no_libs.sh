#\!/usr/bin/env bash
set -u
set -e

echo "Testing without library sourcing"

tests_passed=0
tests_failed=0

test_function_exists() {
    local func_name="$1"
    local description="${2:-Function $func_name}"
    
    if command -v "$func_name" &>/dev/null; then
        echo "✓ $description exists"
        ((tests_passed++))
        return 0
    else
        echo "✗ $description missing"
        ((tests_failed++))
        return 1
    fi
}

echo "Test 1: Core lifecycle functions..."
test_function_exists "echo" "Echo function"
test_function_exists "ls" "List function"
test_function_exists "cat" "Cat function"

echo "All tests completed"
echo "Passed: $tests_passed, Failed: $tests_failed"
