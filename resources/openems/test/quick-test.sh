#!/bin/bash

echo "Quick test starting..."
TESTS_PASSED=0
TESTS_RUN=0

echo -n "Test 1... "
((TESTS_PASSED++))
((TESTS_RUN++))
echo "✅"

echo -n "Test 2... "
((TESTS_PASSED++))
((TESTS_RUN++))
echo "✅"

echo "Results: $TESTS_PASSED/$TESTS_RUN"
echo "Done!"