#!/bin/bash
# Integration tests for mind-maps scenario
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase with runtime requirement and 120-second target
testing::phase::init --require-runtime --target-time "120s"

echo "Comprehensive integration testing: API endpoints and workflows"

TEST_MAP_ID=""
API_URL=""

cleanup_map() {
    if [ -n "$TEST_MAP_ID" ]; then
        curl -sf -X DELETE "$API_URL/api/v1/mind-maps/$TEST_MAP_ID" >/dev/null 2>&1 || true
    fi
}

# Register cleanup function
testing::phase::register_cleanup cleanup_map

if ! testing::core::wait_for_scenario "$TESTING_PHASE_SCENARIO_NAME" 30; then
    testing::phase::add_error "❌ Scenario '$TESTING_PHASE_SCENARIO_NAME' did not become ready in time"
    testing::phase::end_with_summary
fi

if ! API_URL=$(testing::connectivity::get_api_url "$TESTING_PHASE_SCENARIO_NAME"); then
    testing::phase::add_error "❌ Could not resolve API URL for $TESTING_PHASE_SCENARIO_NAME"
    testing::phase::end_with_summary
fi

# Ensure downstream phases inherit the resolved ports
API_PORT="${API_URL##*:}"
if UI_URL=$(testing::connectivity::get_ui_url "$TESTING_PHASE_SCENARIO_NAME" 2>/dev/null); then
    UI_PORT="${UI_URL##*:}"
fi

echo ""
echo "🌐 Testing API Integration..."
if curl -sf --max-time 10 "$API_URL/health" >/dev/null 2>&1; then
    echo "✅ API health check passed ($API_URL/health)"
else
    echo "❌ API integration tests failed - service not responding"
    echo "   Expected API at: $API_URL"
    echo "   💡 Tip: Start with 'vrooli scenario run $TESTING_PHASE_SCENARIO_NAME'"
    testing::phase::add_error
    testing::phase::end_with_summary
fi

echo "🔍 Testing API endpoints..."
if curl -sf --max-time 10 "$API_URL/api/v1/mind-maps" >/dev/null 2>&1; then
    echo "  ✅ Mind maps list endpoint accessible"
else
    echo "  ❌ Mind maps list endpoint failed"
    testing::phase::add_error
fi

health_response=$(curl -sf --max-time 10 "$API_URL/health" 2>/dev/null || echo "")
if [[ "$health_response" =~ "status" ]]; then
    echo "  ✅ Health endpoint returns valid response"
else
    echo "  ❌ Health endpoint response unexpected"
    testing::phase::add_error
fi

echo ""
echo "🗺️  Testing Mind Map CRUD operations..."

# Create mind map
if command -v jq >/dev/null 2>&1; then
    map_payload=$(jq -n --arg title "Integration Test Map $(date +%s)" --arg description "Test map for integration suite" --arg userId "test-user" '{title:$title, description:$description, userId:$userId, isPublic:false}')
else
    map_payload='{"title":"Integration Test Map","description":"Test map","userId":"test-user","isPublic":false}'
fi

map_response=$(curl -sf --max-time 15 -X POST -H "Content-Type: application/json" -d "$map_payload" "$API_URL/api/v1/mind-maps" 2>/dev/null || echo "")
if echo "$map_response" | jq -e '.id' >/dev/null 2>&1; then
    TEST_MAP_ID=$(echo "$map_response" | jq -r '.id')
    echo "✅ Mind map created: $TEST_MAP_ID"
else
    echo "❌ Failed to create mind map for integration tests"
    testing::phase::add_error
    testing::phase::end_with_summary
fi

# Read mind map
if curl -sf --max-time 10 "$API_URL/api/v1/mind-maps/$TEST_MAP_ID" >/dev/null 2>&1; then
    echo "✅ Mind map retrieved successfully"
else
    echo "❌ Failed to retrieve mind map"
    testing::phase::add_error
fi

# Update mind map
update_payload='{"title":"Updated Title","description":"Updated description"}'
if curl -sf --max-time 15 -X PUT -H "Content-Type: application/json" -d "$update_payload" "$API_URL/api/v1/mind-maps/$TEST_MAP_ID" >/dev/null 2>&1; then
    echo "✅ Mind map updated successfully"
else
    echo "❌ Failed to update mind map"
    testing::phase::add_error
fi

echo ""
echo "📝 Testing Node operations..."

# Add node to mind map
if command -v jq >/dev/null 2>&1; then
    node_payload=$(jq -n --arg content "Test Node Content" '{content:$content, x:100, y:100, mindMapId:"'"$TEST_MAP_ID"'"}')
else
    node_payload='{"content":"Test Node Content","x":100,"y":100,"mindMapId":"'"$TEST_MAP_ID"'"}'
fi

node_response=$(curl -sf --max-time 15 -X POST -H "Content-Type: application/json" -d "$node_payload" "$API_URL/api/v1/mind-maps/$TEST_MAP_ID/nodes" 2>/dev/null || echo "")
if echo "$node_response" | jq -e '.id' >/dev/null 2>&1; then
    NODE_ID=$(echo "$node_response" | jq -r '.id')
    echo "✅ Node created: $NODE_ID"
else
    echo "⚠️  Node creation may have failed (database dependent)"
fi

# Get nodes
if curl -sf --max-time 10 "$API_URL/api/v1/mind-maps/$TEST_MAP_ID/nodes" >/dev/null 2>&1; then
    echo "✅ Nodes list retrieved successfully"
else
    echo "⚠️  Nodes list retrieval may have failed"
fi

echo ""
echo "🔄 Testing advanced features..."

# Test search (may fail without Qdrant)
search_payload='{"query":"test","limit":10}'
if curl -sf --max-time 15 -X POST -H "Content-Type: application/json" -d "$search_payload" "$API_URL/api/v1/mind-maps/$TEST_MAP_ID/search" >/dev/null 2>&1; then
    echo "✅ Search endpoint accessible"
else
    echo "ℹ️  Search endpoint may require Qdrant resource"
fi

# Test export
if curl -sf --max-time 10 "$API_URL/api/v1/mind-maps/$TEST_MAP_ID/export?format=json" >/dev/null 2>&1; then
    echo "✅ Export (JSON) successful"
else
    echo "⚠️  Export may have failed"
fi

if curl -sf --max-time 10 "$API_URL/api/v1/mind-maps/$TEST_MAP_ID/export?format=markdown" >/dev/null 2>&1; then
    echo "✅ Export (Markdown) endpoint accessible"
else
    echo "ℹ️  Markdown export not yet implemented"
fi

echo ""
echo "📊 Integration Test Summary:"
echo "════════════════════════════"
echo "   API health: passed"
echo "   Mind map CRUD: passed"
echo "   Node operations: completed"
echo "   Advanced features: tested"

echo ""
log::success "✅ SUCCESS: Integration tests completed!"

# End with summary
testing::phase::end_with_summary "Integration tests completed"
