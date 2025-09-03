#!/bin/bash

# Test the issue capture endpoint

API_URL="${API_URL:-http://localhost:${API_PORT:-3250}}"

echo "Testing issue capture endpoint..."

# Test data
TEST_ISSUE=$(cat <<EOF
{
  "description": "Test issue from automated test",
  "screenshot_path": "/tmp/test-screenshot.png",
  "scenario_name": "test-scenario",
  "url": "http://localhost:3000",
  "context": {
    "test": true,
    "timestamp": "$(date -Iseconds)"
  }
}
EOF
)

# Send request
response=$(curl -s -X POST "$API_URL/api/v1/assistant/capture" \
  -H "Content-Type: application/json" \
  -d "$TEST_ISSUE")

# Check response
if echo "$response" | grep -q "issue_id"; then
  echo "✓ Issue capture successful"
  echo "Response: $response"
  exit 0
else
  echo "✗ Issue capture failed"
  echo "Response: $response"
  exit 1
fi