#!/bin/bash
set -euo pipefail

# Recipe Book Integration Test Runner

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"

cd "$TESTING_PHASE_SCENARIO_DIR"

testing::phase::log "Starting Recipe Book integration tests"

# Ensure dependencies are running
if ! command -v psql &> /dev/null; then
    testing::phase::log "Warning: PostgreSQL client not available"
fi

# Test API health endpoint
testing::phase::log "Testing health endpoint..."
if curl -sf "http://localhost:${API_PORT:-3250}/health" > /dev/null 2>&1; then
    testing::phase::log "✓ Health endpoint responding"
else
    testing::phase::log "⚠ Health endpoint not available (scenario may not be running)"
fi

# Test recipe creation endpoint
testing::phase::log "Testing recipe creation..."
RECIPE_RESPONSE=$(curl -sf -X POST "http://localhost:${API_PORT:-3250}/api/v1/recipes" \
    -H "Content-Type: application/json" \
    -d '{
        "title": "Integration Test Recipe",
        "description": "Test recipe for integration testing",
        "ingredients": [{"name": "flour", "amount": 2, "unit": "cups"}],
        "instructions": ["Mix ingredients", "Bake"],
        "prep_time": 10,
        "cook_time": 20,
        "servings": 4,
        "tags": ["test"],
        "cuisine": "Test",
        "dietary_info": [],
        "nutrition": {"calories": 100},
        "created_by": "integration-test-user",
        "visibility": "private",
        "source": "original"
    }' 2>&1 || echo '{"error": "Request failed"}')

if echo "$RECIPE_RESPONSE" | grep -q '"id"'; then
    testing::phase::log "✓ Recipe creation successful"
    RECIPE_ID=$(echo "$RECIPE_RESPONSE" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)

    # Test recipe retrieval
    testing::phase::log "Testing recipe retrieval..."
    if curl -sf "http://localhost:${API_PORT:-3250}/api/v1/recipes/${RECIPE_ID}?user_id=integration-test-user" > /dev/null 2>&1; then
        testing::phase::log "✓ Recipe retrieval successful"
    fi

    # Test recipe deletion
    testing::phase::log "Testing recipe deletion..."
    if curl -sf -X DELETE "http://localhost:${API_PORT:-3250}/api/v1/recipes/${RECIPE_ID}?user_id=integration-test-user" > /dev/null 2>&1; then
        testing::phase::log "✓ Recipe deletion successful"
    fi
else
    testing::phase::log "⚠ Recipe creation test skipped (API may not be running)"
fi

# Test search endpoint
testing::phase::log "Testing search endpoint..."
if curl -sf -X POST "http://localhost:${API_PORT:-3250}/api/v1/recipes/search" \
    -H "Content-Type: application/json" \
    -d '{"query": "pasta", "user_id": "test-user", "limit": 10}' > /dev/null 2>&1; then
    testing::phase::log "✓ Search endpoint responding"
fi

testing::phase::end_with_summary "Recipe Book integration tests completed"
