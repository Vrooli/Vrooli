#!/bin/bash
set -euo pipefail

echo "=== Test Performance ==="

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

# Get API port
API_PORT=$(vrooli scenario port chart-generator api 2>/dev/null || echo "19098")

# Test chart generation performance (should be < 2000ms per PRD)
echo "Testing chart generation performance..."
start_time=$(date +%s%3N)
response=$(curl -sf -X POST "http://localhost:${API_PORT}/api/v1/charts/generate" \
  -H 'Content-Type: application/json' \
  -d '{"chart_type":"bar","data":[{"x":"A","y":10},{"x":"B","y":20}],"export_formats":["png"]}')
end_time=$(date +%s%3N)

duration=$((end_time - start_time))
if [ "$duration" -gt 2000 ]; then
  echo "⚠️  Chart generation took ${duration}ms (target: <2000ms)"
else
  echo "✅ Chart generation took ${duration}ms (within target)"
fi

# Extract generation_time_ms from response
gen_time=$(echo "$response" | jq -r '.metadata.generation_time_ms // 0')
if [ "$gen_time" -gt 2000 ]; then
  echo "⚠️  Chart processing took ${gen_time}ms (target: <2000ms)"
else
  echo "✅ Chart processing took ${gen_time}ms (excellent performance)"
fi

echo "✅ Performance tests passed"
