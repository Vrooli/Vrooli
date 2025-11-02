#!/bin/bash
# Validates smart-shopping-assistant API endpoints when the scenario is running.

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s" --require-runtime

SCENARIO_NAME="$TESTING_PHASE_SCENARIO_NAME"
API_URL=""

if API_URL=$(testing::connectivity::get_api_url "$SCENARIO_NAME"); then
  log::info "Using API endpoint at $API_URL"
else
  testing::phase::add_error "Unable to discover API port for $SCENARIO_NAME"
  testing::phase::end_with_summary "Integration tests incomplete"
fi

execute_request() {
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

# Health endpoint must respond with expected keys.
health_json=""
if health_json=$(execute_request "API health endpoint" GET "/health"); then
  if command -v jq >/dev/null 2>&1; then
    if ! echo "$health_json" | jq -e '.status == "healthy"' >/dev/null; then
      testing::phase::add_warning "Health payload missing expected status field"
    fi
  elif [[ "$health_json" != *"healthy"* ]]; then
    testing::phase::add_warning "Health response does not mention 'healthy'"
  fi
fi

# Shopping research endpoint should return product data.
research_json=""
if research_json=$(execute_request "POST /api/v1/shopping/research" POST \
  "/api/v1/shopping/research" \
  '{"profile_id":"test-user","query":"laptop","budget_max":1000.0,"include_alternatives":true}'); then
  if [[ "$research_json" != *"products"* ]]; then
    testing::phase::add_warning "Research response missing products field"
  fi
  if [[ "$research_json" != *"alternatives"* ]]; then
    testing::phase::add_warning "Research response missing alternatives field"
  fi
fi

# Tracking endpoint exposes tracked items.
execute_request "GET /api/v1/shopping/tracking" GET \
  "/api/v1/shopping/tracking/test-user"

# Pattern analysis endpoint returns insights.
pattern_json=""
if pattern_json=$(execute_request "POST /api/v1/shopping/pattern-analysis" POST \
  "/api/v1/shopping/pattern-analysis" \
  '{"profile_id":"test-user","timeframe":"30d"}'); then
  if [[ "$pattern_json" != *"patterns"* ]]; then
    testing::phase::add_warning "Pattern analysis response missing patterns field"
  fi
fi

testing::phase::end_with_summary "Integration API tests completed"
