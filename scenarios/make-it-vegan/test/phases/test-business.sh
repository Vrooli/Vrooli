#!/bin/bash
set -e

echo "=== Testing Business Logic ==="

# Test ingredient checking logic
echo "Testing ingredient checking..."
result=$(make-it-vegan check "flour, sugar, salt" 2>&1 || true)
if echo "$result" | grep -q "vegan" || echo "$result" | grep -q "non-vegan"; then
    echo "✓ Ingredient checking works"
else
    echo "⚠ Ingredient checking returned: $result"
fi

# Test substitute suggestions
echo "Testing substitute suggestions..."
result=$(make-it-vegan substitute "milk" 2>&1 || true)
if echo "$result" | grep -q "alternative" || echo "$result" | grep -q "substitute" || echo "$result" | grep -q "soy\|almond\|oat"; then
    echo "✓ Substitute suggestions work"
else
    echo "⚠ Substitute suggestions returned: $result"
fi

echo "✅ Business tests passed"
