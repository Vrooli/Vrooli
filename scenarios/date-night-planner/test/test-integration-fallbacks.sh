#!/bin/bash
# Test integration fallback mechanisms for date-night-planner

set -e

echo "Testing integration fallback mechanisms..."

# Test API fallback mode
echo -n "Testing API fallback response... "
RESPONSE=$(curl -sf -X POST http://localhost:${API_PORT}/api/v1/dates/suggest \
  -H 'Content-Type: application/json' \
  -d '{"couple_id":"test-fallback","date_type":"casual","budget_max":50}' 2>/dev/null)

if echo "$RESPONSE" | grep -q "suggestions"; then
  echo "✅ API returns suggestions in fallback mode"
else
  echo "❌ API failed to return suggestions"
  exit 1
fi

# Test that API gracefully handles missing scenarios
echo -n "Testing missing scenario handling... "
# Since local-info-scout is not running, API should still work
if echo "$RESPONSE" | grep -q "personalization_factors"; then
  echo "✅ API handles missing scenarios gracefully"
else
  echo "❌ API does not handle missing scenarios"
  exit 1
fi

echo "✅ All integration fallback tests passed!"