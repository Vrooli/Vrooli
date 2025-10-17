#!/bin/bash
set -euo pipefail

echo "=== Test Integration ==="

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

# Get API port
API_PORT=$(vrooli scenario port chart-generator api 2>/dev/null || echo "19098")

# Test API health endpoint
echo "Testing API health endpoint..."
if ! curl -sf "http://localhost:${API_PORT}/health" > /dev/null; then
  echo "❌ API health check failed"
  exit 1
fi

# Test chart generation endpoint
echo "Testing chart generation endpoint..."
response=$(curl -sf -X POST "http://localhost:${API_PORT}/api/v1/charts/generate" \
  -H 'Content-Type: application/json' \
  -d '{"chart_type":"bar","data":[{"x":"A","y":10},{"x":"B","y":20}],"export_formats":["png"]}')

if ! echo "$response" | jq -e '.success == true' > /dev/null; then
  echo "❌ Chart generation test failed"
  echo "Response: $response"
  exit 1
fi

# Test styles endpoint
echo "Testing styles listing endpoint..."
if ! curl -sf "http://localhost:${API_PORT}/api/v1/styles" | jq -e '.count > 0' > /dev/null; then
  echo "❌ Styles listing test failed"
  exit 1
fi

# Test CLI integration
echo "Testing CLI chart generation..."
if ! echo '[{"x":"Test","y":42}]' | "${SCENARIO_DIR}/cli/chart-generator" generate bar --data - --format png --output /tmp > /dev/null; then
  echo "❌ CLI integration test failed"
  exit 1
fi

echo "✅ Integration tests passed"
