#!/bin/bash
# Quick test script to verify Judge0 is working correctly

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
MANAGE_SCRIPT="${SCRIPT_DIR}/../../manage.sh"

echo "🧪 Judge0 Quick Test Suite"
echo "========================="
echo

# Function to run a test
run_test() {
    local test_name="$1"
    local code="$2"
    local language="$3"
    local expected="$4"
    
    echo "📝 Test: $test_name"
    echo "   Language: $language"
    
    # Submit code
    local result=$("$MANAGE_SCRIPT" --action submit \
        --code "$code" \
        --language "$language" \
        2>&1)
    
    # Check if output contains expected result
    if echo "$result" | grep -q "$expected"; then
        echo "   ✅ PASSED"
    else
        echo "   ❌ FAILED"
        echo "   Expected: $expected"
        echo "   Got: $result"
    fi
    echo
}

# Check if Judge0 is running
echo "🔍 Checking Judge0 status..."
if ! "$MANAGE_SCRIPT" --action status >/dev/null 2>&1; then
    echo "❌ Judge0 is not running!"
    echo "   Run: $MANAGE_SCRIPT --action install"
    exit 1
fi
echo "✅ Judge0 is running"
echo

# Run tests
echo "🧪 Running basic tests..."
echo

# Test 1: JavaScript
run_test "JavaScript Hello World" \
    'console.log("Hello, Judge0!");' \
    "javascript" \
    "Hello, Judge0!"

# Test 2: Python
run_test "Python Math" \
    'print(2 + 2)' \
    "python" \
    "4"

# Test 3: Go
run_test "Go Output" \
    'package main; import "fmt"; func main() { fmt.Println("Go works!") }' \
    "go" \
    "Go works!"

# Test 4: Input/Output
run_test "Python with Input" \
    'name = input(); print(f"Hello, {name}!")' \
    "python" \
    "Hello," \
    --stdin "World"

# Test 5: Error handling
echo "📝 Test: Error Handling"
echo "   Language: python"
"$MANAGE_SCRIPT" --action submit \
    --code 'print(undefined_variable)' \
    --language "python" >/dev/null 2>&1

if [ $? -ne 0 ]; then
    echo "   ✅ PASSED (correctly reported error)"
else
    echo "   ❌ FAILED (should have reported error)"
fi
echo

# Summary
echo "🎯 Test Summary"
echo "==============="
echo "Basic functionality tests completed."
echo "Check the output above for any failures."
echo
echo "📚 Next steps:"
echo "  - Try more languages: $MANAGE_SCRIPT --action languages"
echo "  - Submit custom code: $MANAGE_SCRIPT --action submit --code 'your code' --language python"
echo "  - Check the examples directory for more complex tests"