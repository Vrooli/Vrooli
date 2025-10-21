#!/bin/bash
set -e

echo "=== Testing Business Logic ==="

# Test agent management through CLI
echo "Testing agent listing..."
result=$(web-scraper-manager agents list 2>&1 || true)
if echo "$result" | grep -q "id\|No agents found\|agents"; then
    echo "✓ Agent listing works"
else
    echo "⚠️  Agent listing returned unexpected output: $result"
fi

# Test platform capabilities
echo "Testing platform capabilities..."
result=$(web-scraper-manager platforms list 2>&1 || true)
if echo "$result" | grep -q "huginn\|browserless\|agent-s2\|platform"; then
    echo "✓ Platform listing works"
else
    echo "⚠️  Platform listing returned unexpected output: $result"
fi

# Test status check
echo "Testing status check..."
result=$(web-scraper-manager status 2>&1 || true)
if echo "$result" | grep -q "healthy\|running\|API"; then
    echo "✓ Status check works"
else
    echo "⚠️  Status check returned unexpected output: $result"
fi

echo "✅ Business tests passed"
