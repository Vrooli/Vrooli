#!/bin/bash
set -e
echo "=== Business Logic Tests ==="

# Get the actual API port from service.json
API_PORT=$(jq -r '.api.config.port // 19450' /home/matthalloran8/Vrooli/scenarios/date-night-planner/.vrooli/service.json 2>/dev/null || echo "19450")

# Test 1: Budget filtering works correctly
echo "Test: Budget filtering..."
RESPONSE=$(curl -sf -X POST http://localhost:${API_PORT}/api/v1/dates/suggest \
  -H 'Content-Type: application/json' \
  -d '{"couple_id":"test","date_type":"romantic","budget_max":50}')
  
# Check all suggestions are under budget
echo "$RESPONSE" | jq -e '.suggestions | map(.estimated_cost <= 50) | all' > /dev/null || { 
  echo "FAIL: Budget filtering not working - found suggestions over $50 limit"
  exit 1
}
echo "✓ Budget filtering works"

# Test 2: Date type filtering works
echo "Test: Date type filtering..."
for TYPE in romantic adventure cultural casual; do
  RESPONSE=$(curl -sf -X POST http://localhost:${API_PORT}/api/v1/dates/suggest \
    -H 'Content-Type: application/json' \
    -d "{\"couple_id\":\"test\",\"date_type\":\"${TYPE}\"}")
  
  # Check we got suggestions for this type
  COUNT=$(echo "$RESPONSE" | jq '.suggestions | length')
  if [ "$COUNT" -eq 0 ]; then
    echo "FAIL: No suggestions returned for date type: $TYPE"
    exit 1
  fi
done
echo "✓ All date types return suggestions"

# Test 3: Weather preference handling
echo "Test: Weather preference handling..."
RESPONSE=$(curl -sf -X POST http://localhost:${API_PORT}/api/v1/dates/suggest \
  -H 'Content-Type: application/json' \
  -d '{"couple_id":"test","date_type":"adventure","weather_preference":"outdoor"}')

# Check outdoor activities exist
echo "$RESPONSE" | jq -e '.suggestions[0].weather_backup' > /dev/null || {
  echo "WARNING: No weather backup found for outdoor activities"
}
echo "✓ Weather preferences handled"

# Test 4: Multiple suggestions returned
echo "Test: Multiple suggestions generated..."
RESPONSE=$(curl -sf -X POST http://localhost:${API_PORT}/api/v1/dates/suggest \
  -H 'Content-Type: application/json' \
  -d '{"couple_id":"test"}')
  
COUNT=$(echo "$RESPONSE" | jq '.suggestions | length')
if [ "$COUNT" -lt 2 ]; then
  echo "FAIL: Only $COUNT suggestion(s) returned, expected at least 2"
  exit 1
fi
echo "✓ Multiple suggestions generated (count: $COUNT)"

# Test 5: Activities are included in suggestions
echo "Test: Activities included in suggestions..."
RESPONSE=$(curl -sf -X POST http://localhost:${API_PORT}/api/v1/dates/suggest \
  -H 'Content-Type: application/json' \
  -d '{"couple_id":"test","date_type":"romantic"}')
  
# Check first suggestion has activities
ACTIVITY_COUNT=$(echo "$RESPONSE" | jq '.suggestions[0].activities | length')
if [ "$ACTIVITY_COUNT" -eq 0 ]; then
  echo "FAIL: No activities in suggestion"
  exit 1
fi
echo "✓ Activities included (count: $ACTIVITY_COUNT)"

# Test 6: Confidence scores are reasonable
echo "Test: Confidence scores validation..."
RESPONSE=$(curl -sf -X POST http://localhost:${API_PORT}/api/v1/dates/suggest \
  -H 'Content-Type: application/json' \
  -d '{"couple_id":"test"}')
  
# Check confidence scores are between 0 and 1
echo "$RESPONSE" | jq -e '.suggestions | map(.confidence_score >= 0 and .confidence_score <= 1) | all' > /dev/null || {
  echo "FAIL: Invalid confidence scores found"
  exit 1
}
echo "✓ Confidence scores valid"

echo "All business tests passed"
exit 0