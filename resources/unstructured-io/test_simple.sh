#\!/usr/bin/env bash
set -e

echo "Testing increment issue"

tests_passed=0

echo "tests_passed before: $tests_passed"
((tests_passed++))
echo "tests_passed after: $tests_passed"

echo "Test completed"
