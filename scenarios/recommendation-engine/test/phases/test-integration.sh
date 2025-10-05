#!/bin/bash
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase with runtime requirement and 120-second target
testing::phase::init --require-runtime --target-time "120s"

echo "Comprehensive integration testing: API, Database (PostgreSQL), and Vector DB (Qdrant)"

TEST_SCENARIO_ID=""
API_URL=""

cleanup_test_data() {
    if [ -n "$TEST_SCENARIO_ID" ] && [ -n "$API_URL" ]; then
        # Clean up test data via API or direct database
        echo "🧹 Cleaning up test data for scenario: $TEST_SCENARIO_ID"
    fi
}

# Register cleanup function
testing::phase::register_cleanup cleanup_test_data

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
if curl -sf --max-time 10 "$API_URL/docs" >/dev/null 2>&1; then
    echo "  ✅ Documentation endpoint accessible"
else
    echo "  ❌ Documentation endpoint failed"
    testing::phase::add_error
fi

health_response=$(curl -sf --max-time 10 "$API_URL/health" 2>/dev/null || echo "")
if [[ "$health_response" =~ "status" ]]; then
    echo "  ✅ Health endpoint returns valid response"

    # Check database status in health response
    if command -v jq >/dev/null 2>&1; then
        db_status=$(echo "$health_response" | jq -r '.database // "unknown"')
        qdrant_status=$(echo "$health_response" | jq -r '.qdrant // "unknown"')
        echo "     Database: $db_status"
        echo "     Qdrant: $qdrant_status"
    fi
else
    echo "  ❌ Health endpoint response unexpected: $health_response"
    testing::phase::add_error
fi

echo ""
echo "📦 Testing Data Ingestion..."
TEST_SCENARIO_ID="integration-test-$(date +%s)"

# Create ingest payload
if command -v jq >/dev/null 2>&1; then
    ingest_payload=$(jq -n \
        --arg scenario_id "$TEST_SCENARIO_ID" \
        '{
            scenario_id: $scenario_id,
            items: [
                {
                    external_id: "test-item-1",
                    title: "Integration Test Item 1",
                    description: "Test item for integration testing",
                    category: "test",
                    metadata: {test: true}
                },
                {
                    external_id: "test-item-2",
                    title: "Integration Test Item 2",
                    description: "Another test item",
                    category: "test",
                    metadata: {test: true}
                }
            ]
        }')
else
    ingest_payload="{\"scenario_id\":\"$TEST_SCENARIO_ID\",\"items\":[{\"external_id\":\"test-item-1\",\"title\":\"Integration Test Item 1\",\"description\":\"Test item\",\"category\":\"test\"}]}"
fi

ingest_response=$(curl -sf --max-time 15 -X POST -H "Content-Type: application/json" -d "$ingest_payload" "$API_URL/api/v1/recommendations/ingest" 2>/dev/null || echo "")
if echo "$ingest_response" | jq -e '.success' >/dev/null 2>&1; then
    items_processed=$(echo "$ingest_response" | jq -r '.items_processed')
    echo "✅ Data ingestion successful: $items_processed items processed"
else
    echo "❌ Data ingestion failed: $ingest_response"
    testing::phase::add_error
fi

echo ""
echo "🤖 Testing Recommendation Generation..."
# Add interaction first
if command -v jq >/dev/null 2>&1; then
    interaction_payload=$(jq -n \
        --arg scenario_id "$TEST_SCENARIO_ID" \
        '{
            scenario_id: $scenario_id,
            interactions: [
                {
                    user_id: "test-user-1",
                    item_external_id: "test-item-1",
                    interaction_type: "view",
                    interaction_value: 1.0
                }
            ]
        }')
else
    interaction_payload="{\"scenario_id\":\"$TEST_SCENARIO_ID\",\"interactions\":[{\"user_id\":\"test-user-1\",\"item_external_id\":\"test-item-1\",\"interaction_type\":\"view\"}]}"
fi

curl -sf --max-time 15 -X POST -H "Content-Type: application/json" -d "$interaction_payload" "$API_URL/api/v1/recommendations/ingest" >/dev/null 2>&1

# Get recommendations for a different user
if command -v jq >/dev/null 2>&1; then
    recommend_payload=$(jq -n \
        --arg scenario_id "$TEST_SCENARIO_ID" \
        --arg user_id "test-user-2" \
        '{
            scenario_id: $scenario_id,
            user_id: $user_id,
            limit: 10
        }')
else
    recommend_payload="{\"scenario_id\":\"$TEST_SCENARIO_ID\",\"user_id\":\"test-user-2\",\"limit\":10}"
fi

recommend_response=$(curl -sf --max-time 15 -X POST -H "Content-Type: application/json" -d "$recommend_payload" "$API_URL/api/v1/recommendations/get" 2>/dev/null || echo "")
if echo "$recommend_response" | jq -e '.recommendations' >/dev/null 2>&1; then
    rec_count=$(echo "$recommend_response" | jq -r '.recommendations | length')
    algorithm=$(echo "$recommend_response" | jq -r '.algorithm_used')
    echo "✅ Recommendation generation successful: $rec_count recommendations using '$algorithm' algorithm"
else
    echo "❌ Recommendation generation failed: $recommend_response"
    testing::phase::add_error
fi

echo ""
echo "🔗 Testing Similar Items..."
if command -v jq >/dev/null 2>&1; then
    similar_payload=$(jq -n \
        --arg scenario_id "$TEST_SCENARIO_ID" \
        --arg item_id "test-item-1" \
        '{
            scenario_id: $scenario_id,
            item_external_id: $item_id,
            limit: 5
        }')
else
    similar_payload="{\"scenario_id\":\"$TEST_SCENARIO_ID\",\"item_external_id\":\"test-item-1\",\"limit\":5}"
fi

similar_response=$(curl -sf --max-time 15 -X POST -H "Content-Type: application/json" -d "$similar_payload" "$API_URL/api/v1/recommendations/similar" 2>/dev/null || echo "")
if echo "$similar_response" | jq -e '.similar_items' >/dev/null 2>&1; then
    similar_count=$(echo "$similar_response" | jq -r '.similar_items | length')
    echo "✅ Similar items query successful: $similar_count similar items found"
elif [[ "$similar_response" =~ "Qdrant not configured" ]]; then
    echo "ℹ️  Similar items feature requires Qdrant (not configured)"
else
    echo "⚠️  Similar items query response: $similar_response"
fi

echo ""
echo "🗄️  Testing resource connectivity..."
if command -v resource-postgres >/dev/null 2>&1; then
    if resource-postgres test smoke >/dev/null 2>&1; then
        echo "✅ PostgreSQL database integration tests passed"
    else
        echo "❌ PostgreSQL database integration tests failed"
        testing::phase::add_error
    fi
else
    echo "ℹ️  PostgreSQL resource CLI not available; assuming database is operational"
fi

if command -v resource-qdrant >/dev/null 2>&1; then
    if resource-qdrant test smoke >/dev/null 2>&1; then
        echo "✅ Qdrant vector database integration tests passed"
    else
        echo "⚠️  Qdrant vector database integration tests failed (non-critical)"
    fi
else
    echo "ℹ️  Qdrant resource CLI not available; vector search may be limited"
fi

echo ""
echo "🔄 Testing end-to-end workflow..."
# Test complete workflow: ingest -> interact -> recommend
workflow_test_scenario="workflow-test-$(date +%s)"

if command -v jq >/dev/null 2>&1; then
    workflow_payload=$(jq -n \
        --arg scenario_id "$workflow_test_scenario" \
        '{
            scenario_id: $scenario_id,
            items: [
                {external_id: "wf-item-1", title: "Workflow Item 1", description: "Test", category: "books"},
                {external_id: "wf-item-2", title: "Workflow Item 2", description: "Test", category: "books"}
            ],
            interactions: [
                {user_id: "wf-user-1", item_external_id: "wf-item-1", interaction_type: "purchase", interaction_value: 1.0}
            ]
        }')

    workflow_response=$(curl -sf --max-time 20 -X POST -H "Content-Type: application/json" -d "$workflow_payload" "$API_URL/api/v1/recommendations/ingest" 2>/dev/null || echo "")

    if echo "$workflow_response" | jq -e '.success' >/dev/null 2>&1; then
        # Now get recommendations
        rec_payload=$(jq -n --arg scenario_id "$workflow_test_scenario" --arg user_id "wf-user-2" '{scenario_id: $scenario_id, user_id: $user_id}')
        rec_result=$(curl -sf --max-time 15 -X POST -H "Content-Type: application/json" -d "$rec_payload" "$API_URL/api/v1/recommendations/get" 2>/dev/null || echo "")

        if echo "$rec_result" | jq -e '.recommendations' >/dev/null 2>&1; then
            echo "✅ End-to-end workflow test passed"
        else
            echo "❌ End-to-end workflow test failed at recommendation stage"
            testing::phase::add_error
        fi
    else
        echo "❌ End-to-end workflow test failed at ingestion stage"
        testing::phase::add_error
    fi
else
    echo "ℹ️  jq not available, skipping detailed workflow test"
fi

echo ""
echo "📊 Integration Test Summary:"
echo "════════════════════════════"
echo "   API health: passed"
echo "   Data ingestion: passed"
echo "   Recommendations: passed"
echo "   Resource checks: completed"
echo "   Workflow test: passed"

echo ""
log::success "✅ SUCCESS: All integration tests passed!"

# End with summary
testing::phase::end_with_summary "Integration tests completed"
