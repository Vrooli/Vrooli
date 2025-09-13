#!/bin/bash
# Simple benchmark test

echo "Testing benchmark functionality..."

# Test 1: Check help includes benchmark
echo -n "1. Benchmark in help... "
if resource-openrouter help 2>/dev/null | grep -q benchmark; then
    echo "✓"
else
    echo "✗"
fi

# Test 2: Test benchmark command exists
echo -n "2. Benchmark command... "
if resource-openrouter benchmark list >/dev/null 2>&1; then
    echo "✓"
else
    echo "✗"
fi

# Test 3: Test single model benchmark
echo -n "3. Single model test... "
output=$(resource-openrouter benchmark test "openai/gpt-3.5-turbo" 2>/dev/null)
if echo "$output" | grep -q '"success": true'; then
    echo "✓"
else
    echo "✗"
fi

echo "Benchmark tests complete!"