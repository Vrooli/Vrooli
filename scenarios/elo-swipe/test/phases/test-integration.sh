#!/bin/bash
# test-integration.sh - Full integration tests

set -e

# Detect actual API port from running process
if DETECTED_PORT=$(lsof -i -P -n | grep "elo-swipe.*LISTEN" | awk '{print $9}' | cut -d: -f2 | head -1); then
    API_PORT=${DETECTED_PORT:-19304}
else
    API_PORT=${API_PORT:-19304}
fi

API_URL="http://localhost:$API_PORT/api/v1"

echo "🔗 Integration Tests"
echo "==================="

# Test 1: Create list
echo -n "✓ Create list... "
RESPONSE=$(curl -s -X POST "$API_URL/lists" \
    -H "Content-Type: application/json" \
    -d '{
        "name": "Integration Test List",
        "description": "Automated test",
        "items": [
            {"content": "Item A"},
            {"content": "Item B"},
            {"content": "Item C"}
        ]
    }')

LIST_ID=$(echo "$RESPONSE" | grep -o '"list_id":"[^"]*' | cut -d'"' -f4)
if [ -n "$LIST_ID" ]; then
    echo "PASS (ID: ${LIST_ID:0:8}...)"
else
    echo "FAIL"
    echo "Response: $RESPONSE"
    exit 1
fi

# Test 2: Get comparison
echo -n "✓ Get next comparison... "
COMPARISON=$(curl -s "$API_URL/lists/$LIST_ID/next-comparison")
if echo "$COMPARISON" | grep -q "item_a"; then
    echo "PASS"
else
    echo "FAIL"
    exit 1
fi

# Test 3: Submit comparison
echo -n "✓ Submit comparison... "
ITEM_A_ID=$(echo "$COMPARISON" | grep -o '"item_a":{"id":"[^"]*' | cut -d'"' -f6)
ITEM_B_ID=$(echo "$COMPARISON" | grep -o '"item_b":{"id":"[^"]*' | cut -d'"' -f6)

COMP_RESPONSE=$(curl -s -X POST "$API_URL/comparisons" \
    -H "Content-Type: application/json" \
    -d "{
        \"list_id\": \"$LIST_ID\",
        \"winner_id\": \"$ITEM_A_ID\",
        \"loser_id\": \"$ITEM_B_ID\"
    }")

if echo "$COMP_RESPONSE" | grep -q "winner_rating_after"; then
    echo "PASS"
else
    echo "FAIL"
    exit 1
fi

# Test 4: Get rankings
echo -n "✓ Get rankings... "
RANKINGS=$(curl -s "$API_URL/lists/$LIST_ID/rankings")
if echo "$RANKINGS" | grep -q "rankings"; then
    RANK_COUNT=$(echo "$RANKINGS" | grep -o '"rank"' | wc -l)
    echo "PASS ($RANK_COUNT items ranked)"
else
    echo "FAIL"
    exit 1
fi

# Test 5: Export CSV
echo -n "✓ Export CSV... "
CSV_EXPORT=$(curl -s "$API_URL/lists/$LIST_ID/rankings?format=csv")
if echo "$CSV_EXPORT" | grep -iq "Rank.*Item.*Elo"; then
    echo "PASS"
else
    echo "FAIL"
    exit 1
fi

# Test 6: Export JSON
echo -n "✓ Export JSON... "
JSON_EXPORT=$(curl -s "$API_URL/lists/$LIST_ID/rankings?format=json")
if echo "$JSON_EXPORT" | grep -q "rankings"; then
    echo "PASS"
else
    echo "FAIL"
    exit 1
fi

echo ""
echo "All integration tests passed! ✅"
