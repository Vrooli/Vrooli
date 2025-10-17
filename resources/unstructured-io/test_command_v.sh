#\!/usr/bin/env bash
set -e

tests_passed=0

test_function_exists() {
    local func_name="$1"
    echo "Testing: $func_name"
    
    if command -v "$func_name" &>/dev/null; then
        echo "✓ $func_name exists"
        ((tests_passed++))
        echo "Incremented, now: $tests_passed"
        return 0
    else
        echo "✗ $func_name missing"
        return 1
    fi
}

echo "About to test echo command"
test_function_exists "echo"
echo "Test completed, final count: $tests_passed"
