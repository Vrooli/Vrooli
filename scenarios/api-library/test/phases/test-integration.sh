#!/bin/bash
# Runs API + CLI integration checks for api-library.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "240s" --require-runtime

SCENARIO_NAME="${TESTING_PHASE_SCENARIO_NAME}"
API_URL=$(testing::connectivity::get_api_url "$SCENARIO_NAME" || true)
UI_URL=$(testing::connectivity::get_ui_url "$SCENARIO_NAME" || true)

if [ -z "$API_URL" ]; then
  testing::phase::add_error "Unable to determine API URL for $SCENARIO_NAME"
  testing::phase::end_with_summary "Integration tests incomplete"
fi

if ! command -v curl >/dev/null 2>&1; then
  testing::phase::add_error "curl is required for integration validation"
  testing::phase::end_with_summary "Integration tests incomplete"
fi

if ! command -v jq >/dev/null 2>&1; then
  testing::phase::add_warning "jq not available; skipping JSON response assertions"
  testing::phase::add_test skipped
  testing::phase::end_with_summary "Integration tests skipped (jq missing)"
fi

# 1. API health endpoint responds with healthy status.
testing::phase::check "API health endpoint" bash -c "curl -sf '$API_URL/health' | jq -e '.status == \"healthy\"' >/dev/null"

# 2. Semantic search surfaces results.
testing::phase::check "Semantic search returns results" bash -c "curl -sf -X POST '$API_URL/api/v1/search' -H 'Content-Type: application/json' -d '{"query":"payment processing","limit":5}' | jq -e '.results | length > 0' >/dev/null"

# 3. Filtered search honours configured flag.
testing::phase::check "Search filter for configured=false" bash -c "curl -sf -X POST '$API_URL/api/v1/search' -H 'Content-Type: application/json' -d '{"query":"api","limit":10,"filters":{"configured":false}}' | jq -e '.results | length >= 0 and (.results | all(.configured == false))' >/dev/null"

# 4. Request research endpoint produces an identifier.
testing::phase::check "Request research endpoint" bash -c "curl -sf -X POST '$API_URL/api/v1/request-research' -H 'Content-Type: application/json' -d '{"capability":"video transcription"}' | jq -e '.research_id | length > 0' >/dev/null"

# Resolve an API id for downstream CLI + mutations.
API_ID=$(curl -sf -X POST "$API_URL/api/v1/search" \
  -H "Content-Type: application/json" \
  -d '{"query":"pricing","limit":1}' | jq -r '.results[0].id // empty' 2>/dev/null || echo "")

if [ -z "$API_ID" ]; then
  testing::phase::add_warning "No API records returned; skipping CLI detail and note tests"
  testing::phase::add_test skipped
else
  # 5. CLI search returns JSON results.
  if command -v api-library >/dev/null 2>&1; then
    testing::phase::check "CLI search command" bash -c "api-library search 'payment processing' --json | jq -e '.results | length > 0' >/dev/null"

    testing::phase::check "CLI show command" bash -c "api-library show '$API_ID' --json | jq -e '.api.id == \"$API_ID\"' >/dev/null"
  else
    testing::phase::add_warning "api-library CLI not on PATH; skipping CLI commands"
    testing::phase::add_test skipped
  fi

  # 6. Add note mutation succeeds.
  testing::phase::check "Add note to API" bash -c "curl -sf -X POST '$API_URL/api/v1/apis/$API_ID/notes' -H 'Content-Type: application/json' -d '{"content":"Integration test note","type":"tip"}' | jq -e '.id | length > 0' >/dev/null"
fi

# 7. UI responds with content marker when available.
if [ -n "$UI_URL" ]; then
  testing::phase::check "UI root responds" bash -c "curl -sf '$UI_URL' | grep -q 'API Library'"
else
  testing::phase::add_warning "UI URL not configured; skipping UI availability check"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Integration validation completed"
