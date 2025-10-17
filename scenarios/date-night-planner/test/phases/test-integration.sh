#!/bin/bash
set -e
echo "=== Integration Tests ==="

# Get the actual API port from service.json
API_PORT=$(jq -r '.api.config.port // 19450' /home/matthalloran8/Vrooli/scenarios/date-night-planner/.vrooli/service.json 2>/dev/null || echo "19450")

# Test health endpoint
echo "Testing health endpoint on port $API_PORT..."
curl -f http://localhost:${API_PORT}/health || { echo "API health check failed"; exit 1; }

# Test date suggestion endpoint
echo "Testing date suggestion API..."
curl -sf -X POST http://localhost:${API_PORT}/api/v1/dates/suggest \
  -H 'Content-Type: application/json' \
  -d '{"couple_id":"test","date_type":"romantic","budget_max":150}' \
  | jq -e '.suggestions | length > 0' > /dev/null || { echo "Date suggestion API failed"; exit 1; }

# Test CLI commands
echo "Testing CLI suggest command..."
/home/matthalloran8/Vrooli/scenarios/date-night-planner/cli/date-night-planner suggest test --type casual --budget 50 --json \
  | jq -e '.suggestions | length > 0' > /dev/null || { echo "CLI suggest command failed"; exit 1; }

echo "Testing CLI status command..."
/home/matthalloran8/Vrooli/scenarios/date-night-planner/cli/date-night-planner status || { echo "CLI status command failed"; exit 1; }

# Test surprise mode endpoint
echo "Testing surprise date creation..."
SURPRISE_RESPONSE=$(curl -sf -X POST http://localhost:${API_PORT}/api/v1/dates/surprise \
  -H 'Content-Type: application/json' \
  -d '{
    "couple_id": "test-couple",
    "planned_by": "partner-1",
    "date_suggestion": {
      "title": "Test Surprise",
      "description": "Test surprise date",
      "activities": [{"type": "romantic", "name": "Test Activity", "duration": "1 hour"}],
      "estimated_cost": 100,
      "estimated_duration": "2 hours"
    },
    "planned_date": "2025-02-14T19:00:00Z"
  }')

echo "$SURPRISE_RESPONSE" | jq -e '.status == "surprise_created"' > /dev/null || { 
  echo "Surprise date creation failed"
  exit 1
}
echo "✓ Surprise mode working"

echo "Integration tests passed"
exit 0