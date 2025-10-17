#!/bin/bash
set -e

# API-specific tests for text-tools scenario
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize test phase
testing::phase::init --target-time "60s"

cd "$TESTING_PHASE_SCENARIO_DIR"

# Get API port - use actual allocated port for text-tools
# The scenario's API is running on this port as allocated by the lifecycle system
export TEXT_TOOLS_API_PORT=16518

echo "=== API Tests (Port: ${TEXT_TOOLS_API_PORT}) ==="

# Test all v1 endpoints
echo "Testing /api/v1/text/diff..."
response=$(curl -sf -X POST "http://localhost:${TEXT_TOOLS_API_PORT}/api/v1/text/diff" \
    -H "Content-Type: application/json" \
    -d '{"text1":"line1\nline2","text2":"line1\nline3"}')
echo "$response" | jq -e '.changes | length > 0' || {
    echo "❌ Diff failed to detect changes"
    exit 1
}
echo "✅ Diff endpoint works"

echo "Testing /api/v1/text/search..."
response=$(curl -sf -X POST "http://localhost:${TEXT_TOOLS_API_PORT}/api/v1/text/search" \
    -H "Content-Type: application/json" \
    -d '{"text":"test text with pattern","pattern":"pattern"}')
echo "$response" | jq -e '.total_matches > 0' || {
    echo "❌ Search failed to find pattern"
    exit 1
}
echo "✅ Search endpoint works"

echo "Testing /api/v1/text/transform..."
response=$(curl -sf -X POST "http://localhost:${TEXT_TOOLS_API_PORT}/api/v1/text/transform" \
    -H "Content-Type: application/json" \
    -d '{"text":"lowercase text","transformations":[{"type":"case","parameters":{"type":"upper"}}]}')
result=$(echo "$response" | jq -r '.result')
if [[ "$result" == "LOWERCASE TEXT" ]]; then
    echo "✅ Transform endpoint works"
else
    echo "❌ Transform failed: expected 'LOWERCASE TEXT', got '$result'"
    exit 1
fi

echo "Testing /api/v1/text/analyze..."
response=$(curl -sf -X POST "http://localhost:${TEXT_TOOLS_API_PORT}/api/v1/text/analyze" \
    -H "Content-Type: application/json" \
    -d '{"text":"Contact john@example.com for more info","analyses":["entities"]}')
echo "$response" | jq -e '.entities | length > 0' || {
    echo "❌ Analyze failed to extract entities"
    exit 1
}
echo "✅ Analyze endpoint works"

# End test phase with summary
testing::phase::end_with_summary "API tests completed"
