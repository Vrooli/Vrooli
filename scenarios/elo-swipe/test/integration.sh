#!/bin/bash

# Elo Swipe Integration Test

set -e

API_PORT=${API_PORT:-19294}
API_URL="http://localhost:$API_PORT/api/v1"

echo "Running Elo Swipe integration tests..."

# Test 1: Health check
echo -n "Testing API health... "
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL/health")
if [ "$STATUS" = "200" ]; then
    echo "✓"
else
    echo "✗ (HTTP $STATUS)"
    exit 1
fi

# Test 2: Create a test list
echo -n "Creating test list... "
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
    echo "✓ (ID: ${LIST_ID:0:8}...)"
else
    echo "✗"
    echo "Response: $RESPONSE"
    exit 1
fi

# Test 3: Get next comparison
echo -n "Getting next comparison... "
COMPARISON=$(curl -s "$API_URL/lists/$LIST_ID/next-comparison")
if echo "$COMPARISON" | grep -q "item_a"; then
    echo "✓"
else
    echo "✗"
    echo "Response: $COMPARISON"
    exit 1
fi

# Test 4: Submit a comparison
echo -n "Submitting comparison... "
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
    echo "✓"
else
    echo "✗"
    echo "Response: $COMP_RESPONSE"
    exit 1
fi

# Test 5: Get rankings
echo -n "Getting rankings... "
RANKINGS=$(curl -s "$API_URL/lists/$LIST_ID/rankings")
if echo "$RANKINGS" | grep -q "rankings"; then
    RANK_COUNT=$(echo "$RANKINGS" | grep -o '"rank"' | wc -l)
    echo "✓ ($RANK_COUNT items ranked)"
else
    echo "✗"
    echo "Response: $RANKINGS"
    exit 1
fi

# Test 6: CLI status
echo -n "Testing CLI status... "
if API_PORT=$API_PORT VROOLI_LIFECYCLE_MANAGED=true elo-swipe status &>/dev/null; then
    echo "✓"
else
    echo "✗"
    exit 1
fi

echo ""
echo "All integration tests passed! ✅"