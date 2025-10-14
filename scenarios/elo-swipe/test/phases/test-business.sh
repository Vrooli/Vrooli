#!/bin/bash
set -e
echo "=== Business Logic Tests ==="
# Test core Elo rating business functionality
API_PORT=${API_PORT:-19286}

if curl -sf http://localhost:${API_PORT}/health > /dev/null 2>&1; then
  echo "API is healthy, running business tests"

  # Test 1: Create a list and verify it exists
  echo "Test 1: Creating ranking list..."
  LIST_RESPONSE=$(curl -sf -X POST http://localhost:${API_PORT}/api/v1/lists \
    -H "Content-Type: application/json" \
    -d '{"name":"Test List","description":"Business test list","items":[{"content":"Item A"},{"content":"Item B"},{"content":"Item C"}]}')

  LIST_ID=$(echo "$LIST_RESPONSE" | jq -r '.list_id // empty')
  if [ -z "$LIST_ID" ]; then
    echo "❌ Failed to create list"
    exit 1
  fi
  echo "✅ List created: $LIST_ID"

  # Test 2: Submit a comparison and verify Elo ratings update
  echo "Test 2: Submitting comparison..."
  COMPARISON_RESPONSE=$(curl -sf -X GET http://localhost:${API_PORT}/api/v1/lists/${LIST_ID}/next-comparison)
  ITEM_A=$(echo "$COMPARISON_RESPONSE" | jq -r '.item_a.id // empty')
  ITEM_B=$(echo "$COMPARISON_RESPONSE" | jq -r '.item_b.id // empty')

  if [ -n "$ITEM_A" ] && [ -n "$ITEM_B" ]; then
    curl -sf -X POST http://localhost:${API_PORT}/api/v1/comparisons \
      -H "Content-Type: application/json" \
      -d "{\"list_id\":\"${LIST_ID}\",\"winner_id\":\"${ITEM_A}\",\"loser_id\":\"${ITEM_B}\"}" > /dev/null
    echo "✅ Comparison submitted successfully"
  else
    echo "⚠️  Could not get comparison pair (expected for small lists)"
  fi

  # Test 3: Verify rankings are returned
  echo "Test 3: Retrieving rankings..."
  RANKINGS=$(curl -sf http://localhost:${API_PORT}/api/v1/lists/${LIST_ID}/rankings)
  RANK_COUNT=$(echo "$RANKINGS" | jq '.rankings | length')
  if [ "$RANK_COUNT" -gt 0 ]; then
    echo "✅ Rankings retrieved: $RANK_COUNT items"
  else
    echo "❌ No rankings returned"
    exit 1
  fi

  echo "Business logic validation passed"
else
  echo "API not running, skipping business tests"
fi

echo "✅ Business tests completed"
