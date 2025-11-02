#!/bin/bash
# Exercises business workflows for smart-shopping-assistant against a running runtime.

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s" --require-runtime

API_URL=""
if API_URL=$(testing::connectivity::get_api_url "$TESTING_PHASE_SCENARIO_NAME"); then
  log::info "Targeting API at $API_URL"
else
  testing::phase::add_error "Unable to resolve API URL for $TESTING_PHASE_SCENARIO_NAME"
  testing::phase::end_with_summary "Business tests skipped"
fi

invoke_api() {
  local description="$1"
  local method="$2"
  local path="$3"
  local payload="${4:-}"
  local response

  testing::phase::add_test "$description"

  if [[ -n "$payload" ]]; then
    if response=$(curl -sf -X "$method" "$API_URL$path" \
      -H "Content-Type: application/json" \
      -d "$payload"); then
      log::success "✅ $description"
      printf '%s' "$response"
      return 0
    fi
  else
    if response=$(curl -sf -X "$method" "$API_URL$path"); then
      log::success "✅ $description"
      printf '%s' "$response"
      return 0
    fi
  fi

  testing::phase::add_error "$description failed"
  return 1
}

# Shopping research with budgeting and alternatives.
research_json=""
if research_json=$(invoke_api "Shopping research returns catalog data" POST \
  "/api/v1/shopping/research" \
  '{"profile_id":"test-user","query":"laptop","budget_max":1000.0,"include_alternatives":true}'); then
  [[ "$research_json" == *"products"* ]] || testing::phase::add_warning "Research response missing products"
  [[ "$research_json" == *"alternatives"* ]] || testing::phase::add_warning "Research response missing alternatives"
  [[ "$research_json" == *"price_analysis"* ]] || testing::phase::add_warning "Research response missing price analysis"
fi

# Price tracking lifecycle (create then list).
invoke_api "Create price tracking record" POST \
  "/api/v1/shopping/tracking" \
  '{"profile_id":"test-user","product_id":"test-product-123"}' >/dev/null || true
invoke_api "Retrieve price tracking records" GET \
  "/api/v1/shopping/tracking/test-user" >/dev/null || true

# Pattern analysis for behavioural insights.
pattern_json=""
if pattern_json=$(invoke_api "Pattern analysis returns insights" POST \
  "/api/v1/shopping/pattern-analysis" \
  '{"profile_id":"test-user","timeframe":"30d"}'); then
  [[ "$pattern_json" == *"patterns"* ]] || testing::phase::add_warning "Pattern analysis response missing patterns"
  [[ "$pattern_json" == *"predictions"* ]] || testing::phase::add_warning "Pattern analysis response missing predictions"
fi

# Gift recommendation workflow.
invoke_api "Gift recommendation research" POST \
  "/api/v1/shopping/research" \
  '{"profile_id":"test-user","query":"gift","budget_max":75.0,"gift_recipient_id":"recipient-123"}' >/dev/null || true

# Affiliate links in research results.
if [[ -n "$research_json" && "$research_json" != *"affiliate_links"* ]]; then
  testing::phase::add_warning "Affiliate links absent from research payload"
fi

# Multi-profile support and alert creation.
invoke_api "Fetch shopping profiles" GET "/api/v1/profiles" >/dev/null || true
invoke_api "Create price alert" POST \
  "/api/v1/alerts" \
  '{"profile_id":"test-user","product_id":"prod-123","target_price":99.99,"alert_type":"below_target"}' >/dev/null || true
invoke_api "List price alerts" GET "/api/v1/alerts" >/dev/null || true

testing::phase::end_with_summary "Business workflow validation completed"
