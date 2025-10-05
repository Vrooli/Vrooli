#!/bin/bash

# Integration test phase for smart-shopping-assistant
# Tests API endpoints with running service

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"

SCENARIO_NAME="smart-shopping-assistant"
API_PORT="${SMART_SHOPPING_ASSISTANT_API_PORT:-3300}"
API_URL="http://localhost:${API_PORT}"

echo "Testing integration with smart-shopping-assistant API at ${API_URL}"

# Test health endpoint
testing::phase::log "Testing health endpoint..."
health_response=$(curl -s -f "${API_URL}/health" || echo "FAILED")
if [[ "$health_response" == "FAILED" ]]; then
    testing::phase::error "Health check failed - service may not be running"
    exit 1
fi

echo "✅ Health check passed"

# Test shopping research endpoint
testing::phase::log "Testing shopping research endpoint..."
research_response=$(curl -s -X POST "${API_URL}/api/v1/shopping/research" \
    -H "Content-Type: application/json" \
    -d '{
        "profile_id": "test-user",
        "query": "laptop",
        "budget_max": 1000.00,
        "include_alternatives": true
    }' || echo "FAILED")

if [[ "$research_response" == "FAILED" ]]; then
    testing::phase::error "Shopping research endpoint failed"
    exit 1
fi

echo "✅ Shopping research endpoint passed"

# Test tracking endpoint
testing::phase::log "Testing tracking endpoint..."
tracking_response=$(curl -s -f "${API_URL}/api/v1/shopping/tracking/test-user" || echo "FAILED")
if [[ "$tracking_response" == "FAILED" ]]; then
    testing::phase::error "Tracking endpoint failed"
    exit 1
fi

echo "✅ Tracking endpoint passed"

# Test pattern analysis endpoint
testing::phase::log "Testing pattern analysis endpoint..."
pattern_response=$(curl -s -X POST "${API_URL}/api/v1/shopping/pattern-analysis" \
    -H "Content-Type: application/json" \
    -d '{
        "profile_id": "test-user",
        "timeframe": "30d"
    }' || echo "FAILED")

if [[ "$pattern_response" == "FAILED" ]]; then
    testing::phase::error "Pattern analysis endpoint failed"
    exit 1
fi

echo "✅ Pattern analysis endpoint passed"

testing::phase::end_with_summary "Integration tests completed successfully"
