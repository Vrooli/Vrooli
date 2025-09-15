#\!/usr/bin/env bash
set -e

echo "Starting minimal test"

# Test function
test_function_exists() {
    local func_name="$1"
    echo "Checking: $func_name"
    if command -v "$func_name" &>/dev/null; then
        echo "✓ $func_name exists"
        return 0
    else
        echo "✗ $func_name missing"
        return 1
    fi
}

echo "About to call first test"
test_function_exists "echo" 
echo "First test completed"

echo "About to call second test"  
test_function_exists "ls"
echo "Second test completed"

echo "Both tests completed successfully"
