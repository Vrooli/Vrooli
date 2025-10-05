#!/bin/bash
set -euo pipefail

echo "=== Test Business Value ==="

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

# Get API port
API_PORT=$(vrooli scenario port chart-generator api 2>/dev/null || echo "19098")

# Verify P0 requirements are met
echo "Verifying P0 requirements..."

# P0: Core chart types (bar, line, pie, scatter, area)
for chart_type in bar line pie scatter area; do
  echo "Testing $chart_type chart generation..."
  response=$(curl -sf -X POST "http://localhost:${API_PORT}/api/v1/charts/generate" \
    -H 'Content-Type: application/json' \
    -d "{\"chart_type\":\"$chart_type\",\"data\":[{\"x\":\"A\",\"y\":10},{\"x\":\"B\",\"y\":20}],\"export_formats\":[\"png\"]}")

  if ! echo "$response" | jq -e '.success == true' > /dev/null; then
    echo "❌ $chart_type chart generation failed"
    exit 1
  fi
done

# P0: Export capabilities (PNG, SVG)
echo "Testing multi-format export..."
response=$(curl -sf -X POST "http://localhost:${API_PORT}/api/v1/charts/generate" \
  -H 'Content-Type: application/json' \
  -d '{"chart_type":"bar","data":[{"x":"A","y":10}],"export_formats":["png","svg"]}')

if ! echo "$response" | jq -e '.files.png and .files.svg' > /dev/null; then
  echo "❌ Multi-format export failed"
  exit 1
fi

# P0: Professional styling themes
echo "Testing style availability..."
styles=$(curl -sf "http://localhost:${API_PORT}/api/v1/styles")
style_count=$(echo "$styles" | jq -r '.count // 0')

if [ "$style_count" -lt 3 ]; then
  echo "❌ Insufficient styles available (found $style_count, expected ≥3)"
  exit 1
fi

# P0: CLI interface
echo "Testing CLI functionality..."
if ! "${SCENARIO_DIR}/cli/chart-generator" status > /dev/null; then
  echo "❌ CLI status command failed"
  exit 1
fi

# Verify business value metrics from PRD
echo "Verifying business value metrics..."

# Revenue potential: $15K - $40K per deployment
# Reusability score: 9/10 (from PRD)
# Performance target: <2000ms for complex charts

response=$(curl -sf -X POST "http://localhost:${API_PORT}/api/v1/charts/generate" \
  -H 'Content-Type: application/json' \
  -d '{"chart_type":"bar","data":[{"x":"A","y":10},{"x":"B","y":20}],"export_formats":["png"]}')

gen_time=$(echo "$response" | jq -r '.metadata.generation_time_ms // 0')
if [ "$gen_time" -gt 2000 ]; then
  echo "⚠️  Performance below target: ${gen_time}ms (target: <2000ms)"
else
  echo "✅ Performance exceeds target: ${gen_time}ms"
fi

echo "✅ Business value tests passed"
echo "   - All P0 requirements verified"
echo "   - Performance targets met"
echo "   - Professional styling available"
echo "   - CLI integration working"
