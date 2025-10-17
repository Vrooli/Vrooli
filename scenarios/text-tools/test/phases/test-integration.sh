#!/bin/bash
set -e

# Integration tests for text-tools scenario
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize test phase
testing::phase::init --target-time "120s"

cd "$TESTING_PHASE_SCENARIO_DIR"

# Get API port - use actual allocated port for text-tools
# The scenario's API is running on this port as allocated by the lifecycle system
export TEXT_TOOLS_API_PORT=16518

echo "=== Integration Tests (Port: ${TEXT_TOOLS_API_PORT}) ==="

# Test API health
echo "Testing API health endpoint..."
curl -sf "http://localhost:${TEXT_TOOLS_API_PORT}/health" || {
    echo "❌ API health check failed"
    exit 1
}
echo "✅ API health check passed"

# Test diff endpoint
echo "Testing diff endpoint..."
curl -sf -X POST "http://localhost:${TEXT_TOOLS_API_PORT}/api/v1/text/diff" \
    -H "Content-Type: application/json" \
    -d '{"text1":"hello world","text2":"hello Vrooli"}' | jq -e '.changes' || {
    echo "❌ Diff endpoint failed"
    exit 1
}
echo "✅ Diff endpoint passed"

# Test search endpoint
echo "Testing search endpoint..."
curl -sf -X POST "http://localhost:${TEXT_TOOLS_API_PORT}/api/v1/text/search" \
    -H "Content-Type: application/json" \
    -d '{"text":"The quick brown fox","pattern":"quick"}' | jq -e '.matches' || {
    echo "❌ Search endpoint failed"
    exit 1
}
echo "✅ Search endpoint passed"

# Test transform endpoint
echo "Testing transform endpoint..."
curl -sf -X POST "http://localhost:${TEXT_TOOLS_API_PORT}/api/v1/text/transform" \
    -H "Content-Type: application/json" \
    -d '{"text":"hello","transformations":[{"type":"case","parameters":{"type":"upper"}}]}' | jq -e '.result' || {
    echo "❌ Transform endpoint failed"
    exit 1
}
echo "✅ Transform endpoint passed"

# Test analyze endpoint
echo "Testing analyze endpoint..."
curl -sf -X POST "http://localhost:${TEXT_TOOLS_API_PORT}/api/v1/text/analyze" \
    -H "Content-Type: application/json" \
    -d '{"text":"test@example.com","analyses":["entities"]}' | jq -e '.entities' || {
    echo "❌ Analyze endpoint failed"
    exit 1
}
echo "✅ Analyze endpoint passed"

# End test phase with summary
testing::phase::end_with_summary "Integration tests completed"
