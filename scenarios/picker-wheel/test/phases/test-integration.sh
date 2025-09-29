#!/bin/bash
set -e

echo "=== Integration Tests for Picker Wheel ==="

API_PORT="${API_PORT:-19899}"
UI_PORT="${UI_PORT:-37193}"
API_URL="http://localhost:${API_PORT}"
UI_URL="http://localhost:${UI_PORT}"

# Check API health
echo "Testing API health endpoint..."
curl -sf "${API_URL}/health" || { echo "❌ API health check failed"; exit 1; }
echo "✅ API health check passed"

# Test wheel listing
echo "Testing wheel listing..."
WHEELS=$(curl -sf "${API_URL}/api/wheels")
if [ -z "$WHEELS" ]; then
  echo "❌ Failed to get wheels list"
  exit 1
fi
echo "✅ Wheels retrieved: $(echo $WHEELS | jq '. | length') wheels found"

# Test spin functionality
echo "Testing wheel spin..."
SPIN_RESULT=$(curl -sf -X POST "${API_URL}/api/spin" \
  -H "Content-Type: application/json" \
  -d '{"wheel_id": "yes-or-no"}')
if [ -z "$SPIN_RESULT" ]; then
  echo "❌ Spin test failed"
  exit 1
fi
echo "✅ Spin result: $(echo $SPIN_RESULT | jq -r '.result')"

# Test custom wheel creation
echo "Testing custom wheel creation..."
CUSTOM_WHEEL=$(curl -sf -X POST "${API_URL}/api/wheels" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Wheel",
    "description": "Integration test wheel",
    "options": [
      {"label": "Option A", "color": "#FF0000", "weight": 1},
      {"label": "Option B", "color": "#00FF00", "weight": 1}
    ],
    "theme": "test"
  }')
if [ -z "$CUSTOM_WHEEL" ]; then
  echo "❌ Custom wheel creation failed"
  exit 1
fi
echo "✅ Custom wheel created with ID: $(echo $CUSTOM_WHEEL | jq -r '.id')"

# Test UI availability
echo "Testing UI server..."
curl -sf "${UI_URL}/" -o /dev/null || { echo "❌ UI server not responding"; exit 1; }
echo "✅ UI server is accessible"

# Test history endpoint
echo "Testing history endpoint..."
curl -sf "${API_URL}/api/history" || { echo "❌ History endpoint failed"; exit 1; }
echo "✅ History endpoint working"

echo "✅ All integration tests passed"