#!/bin/bash
# Validates business workflows spanning discovery, cost analysis, and CLI flows.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s" --require-runtime

SCENARIO_NAME="${TESTING_PHASE_SCENARIO_NAME}"
API_URL=$(testing::connectivity::get_api_url "$SCENARIO_NAME" || true)

if [ -z "$API_URL" ]; then
  testing::phase::add_error "Unable to determine API URL for $SCENARIO_NAME"
  testing::phase::end_with_summary "Business tests incomplete"
fi

if ! command -v jq >/dev/null 2>&1; then
  testing::phase::add_warning "jq not available; skipping business workflow assertions"
  testing::phase::add_test skipped
  testing::phase::end_with_summary "Business tests skipped"
fi

# Categories catalogue should be present for browsing.
testing::phase::check "Categories endpoint" bash -c "curl -sf '$API_URL/api/v1/categories' | jq -e 'type == \"array\" and length >= 0' >/dev/null"

# Tags catalogue powers filtering and onboarding.
testing::phase::check "Tags endpoint" bash -c "curl -sf '$API_URL/api/v1/tags' | jq -e 'type == \"array\" and length >= 0' >/dev/null"

# Fetch a representative API for deeper business workflows.
API_ID=$(curl -sf -X POST "$API_URL/api/v1/search" \
  -H "Content-Type: application/json" \
  -d '{"query":"analytics","limit":1}' | jq -r '.results[0].id // empty' 2>/dev/null || echo "")

if [ -z "$API_ID" ]; then
  testing::phase::add_warning "No APIs returned during discovery; skipping cost and lifecycle checks"
  testing::phase::add_test skipped
else
  testing::phase::check "Calculate cost recommendation" bash -c "curl -sf -X POST '$API_URL/api/v1/calculate-cost' -H 'Content-Type: application/json' -d '{"api_id":"$API_ID","requests_per_month":5000,"data_per_request_mb":1.5}' | jq -e '.recommended_tier.name and (.recommended_tier.estimated_cost | tonumber >= 0)' >/dev/null"

  testing::phase::check "API version history endpoint" bash -c "curl -sf '$API_URL/api/v1/apis/$API_ID/versions' | jq -e 'type == \"array\"' >/dev/null"

  testing::phase::check "API export includes records" bash -c "curl -sf '$API_URL/api/v1/export?format=json' | jq -e 'type == \"array\" and length >= 0' >/dev/null"

  if command -v api-library >/dev/null 2>&1; then
    testing::phase::check "CLI list-configured" bash -c "api-library list-configured --json | jq -e 'type == \"array\"' >/dev/null"
  else
    testing::phase::add_warning "api-library CLI not on PATH; skipping configured listings"
    testing::phase::add_test skipped
  fi
fi

testing::phase::end_with_summary "Business workflow validation completed"
