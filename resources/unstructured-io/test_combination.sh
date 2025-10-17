#\!/usr/bin/env bash
set -e

tests_passed=0

test_function() {
    echo "In test function"
    ((tests_passed++))
    echo "Increment done"
    return 0
}

echo "About to call function"
test_function
echo "Function call completed"
echo "Final tests_passed: $tests_passed"
