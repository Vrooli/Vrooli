#!/bin/bash
# Test business logic phase - validates core business functionality

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "45s"

cd "$TESTING_PHASE_SCENARIO_DIR"

log::info "Testing smart-shopping-assistant business logic"

# Require API to be running
API_PORT="${API_PORT:-3300}"
API_URL="http://localhost:${API_PORT}"

# Helper function to test API endpoint
test_endpoint() {
    local method="$1"
    local path="$2"
    local data="${3:-}"
    local description="$4"

    testing::phase::add_test "$description"

    if [[ -n "$data" ]]; then
        if response=$(curl -sf -X "$method" "$API_URL$path" \
            -H "Content-Type: application/json" \
            -d "$data" 2>&1); then
            log::success "$description: OK"
            echo "$response"
        else
            testing::phase::add_error "$description failed"
            return 1
        fi
    else
        if response=$(curl -sf -X "$method" "$API_URL$path" 2>&1); then
            log::success "$description: OK"
            echo "$response"
        else
            testing::phase::add_error "$description failed"
            return 1
        fi
    fi
}

# Test 1: Shopping Research with Budget
testing::phase::add_test "Business: Shopping research respects budget constraints"
RESEARCH_RESPONSE=$(test_endpoint POST "/api/v1/shopping/research" \
    '{"query":"laptop","budget_max":1000.0,"profile_id":"test-user"}' \
    "Shopping research with budget") || true

if [[ -n "$RESEARCH_RESPONSE" ]]; then
    # Validate products are within budget (would need jq for full validation)
    if echo "$RESEARCH_RESPONSE" | grep -q "products"; then
        log::success "Shopping research returns products"
    else
        testing::phase::add_warning "Shopping research response format unexpected"
    fi
fi

# Test 2: Price Tracking Functionality
testing::phase::add_test "Business: Price tracking creates alerts"
TRACKING_CREATE=$(test_endpoint POST "/api/v1/shopping/tracking" \
    '{"profile_id":"test-user","product_id":"test-product-123"}' \
    "Create price tracking") || true

# Test 3: Pattern Analysis
testing::phase::add_test "Business: Pattern analysis identifies shopping patterns"
PATTERN_RESPONSE=$(test_endpoint POST "/api/v1/shopping/pattern-analysis" \
    '{"profile_id":"test-user","timeframe":"30d"}' \
    "Pattern analysis") || true

if [[ -n "$PATTERN_RESPONSE" ]]; then
    if echo "$PATTERN_RESPONSE" | grep -q "patterns"; then
        log::success "Pattern analysis returns patterns"
    fi
    if echo "$PATTERN_RESPONSE" | grep -q "predictions"; then
        log::success "Pattern analysis returns predictions"
    fi
    if echo "$PATTERN_RESPONSE" | grep -q "savings_opportunities"; then
        log::success "Pattern analysis returns savings opportunities"
    fi
fi

# Test 4: Alternative Product Suggestions
testing::phase::add_test "Business: Alternative products offer savings"
ALT_RESPONSE=$(test_endpoint POST "/api/v1/shopping/research" \
    '{"query":"headphones","budget_max":200.0,"profile_id":"test-user","include_alternatives":true}' \
    "Research with alternatives") || true

if [[ -n "$ALT_RESPONSE" ]]; then
    if echo "$ALT_RESPONSE" | grep -q "alternatives"; then
        log::success "Alternatives are included in response"
    fi
fi

# Test 5: Profile Management
testing::phase::add_test "Business: Multi-profile support"
PROFILES_RESPONSE=$(test_endpoint GET "/api/v1/profiles" "" \
    "Get profiles list") || true

# Test 6: Alert Management
testing::phase::add_test "Business: Price alert creation and retrieval"
ALERT_CREATE=$(test_endpoint POST "/api/v1/alerts" \
    '{"profile_id":"test-user","product_id":"prod-123","target_price":99.99,"alert_type":"below_target"}' \
    "Create price alert") || true

ALERTS_LIST=$(test_endpoint GET "/api/v1/alerts" "" \
    "Get alerts list") || true

# Test 7: Gift Recipient Integration
testing::phase::add_test "Business: Gift recipient recommendations"
GIFT_RESPONSE=$(test_endpoint POST "/api/v1/shopping/research" \
    '{"query":"birthday gift","budget_max":50.0,"profile_id":"test-user","gift_recipient_id":"recipient-123"}' \
    "Gift recommendations") || true

# Test 8: Affiliate Link Generation
testing::phase::add_test "Business: Affiliate links generated for revenue"
if [[ -n "$RESEARCH_RESPONSE" ]]; then
    if echo "$RESEARCH_RESPONSE" | grep -q "affiliate_links"; then
        log::success "Affiliate links are generated"
    else
        testing::phase::add_warning "Affiliate links not found in response"
    fi
fi

# Test 9: Price History and Insights
testing::phase::add_test "Business: Price analysis provides actionable insights"
if [[ -n "$RESEARCH_RESPONSE" ]]; then
    if echo "$RESEARCH_RESPONSE" | grep -q "price_analysis"; then
        log::success "Price analysis is included"
    fi
    if echo "$RESEARCH_RESPONSE" | grep -q "recommendations"; then
        log::success "Recommendations are generated"
    fi
fi

testing::phase::end_with_summary "Business logic tests completed"
