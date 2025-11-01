#!/bin/bash
# Integration testing phase for the symbol-search scenario

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/connectivity.sh"

testing::phase::init --target-time "180s" --require-runtime

SCENARIO_NAME="${TESTING_PHASE_SCENARIO_NAME}"
API_URL=$(testing::connectivity::get_api_url "$SCENARIO_NAME" || true)

if [ -z "$API_URL" ]; then
  testing::phase::add_error "Unable to resolve API URL for $SCENARIO_NAME"
  testing::phase::end_with_summary "Integration checks incomplete"
fi

wait_for_endpoint() {
  local url="$1"
  local retries=30
  while [ $retries -gt 0 ]; do
    if curl -sf "$url" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
    retries=$((retries - 1))
  done
  return 1
}

if wait_for_endpoint "${API_URL}/health"; then
  testing::phase::add_test passed
else
  testing::phase::add_error "API did not become healthy within timeout"
  testing::phase::add_test failed
  testing::phase::end_with_summary "Integration checks incomplete"
fi

USE_JQ=true
if ! command -v jq >/dev/null 2>&1; then
  USE_JQ=false
  testing::phase::add_warning "jq not available; falling back to simple string checks"
  testing::phase::add_test skipped
fi

if [ "$USE_JQ" = true ]; then
  testing::phase::check "Health endpoint reports healthy" bash -c "curl -sf '${API_URL}/health' | jq -e '.status==\"healthy\"' >/dev/null"
  testing::phase::check "Search endpoint returns characters" bash -c "curl -sf '${API_URL}/api/search?q=LATIN&limit=5' | jq -e '.characters | length > 0' >/dev/null"
  testing::phase::check "Category-filtered search returns results" bash -c "curl -sf '${API_URL}/api/search?category=So&limit=3' | jq -e '.characters | length > 0' >/dev/null"
  testing::phase::check "Categories endpoint returns array" bash -c "curl -sf '${API_URL}/api/categories' | jq -e '.categories | length > 0' >/dev/null"
  testing::phase::check "Blocks endpoint returns array" bash -c "curl -sf '${API_URL}/api/blocks' | jq -e '.blocks | length > 0' >/dev/null"
  testing::phase::check "Character detail endpoint resolves U+0041" bash -c "curl -sf '${API_URL}/api/character/U+0041' | jq -e '.character.codepoint == \"U+0041\"' >/dev/null"
  testing::phase::check "Bulk range endpoint returns symbols" bash -c "curl -sf -X POST '${API_URL}/api/bulk/range' -H 'Content-Type: application/json' -d '{\"ranges\":[{\"start\":\"U+0041\",\"end\":\"U+005A\"}]}' | jq -e '.characters | length > 0' >/dev/null"
  testing::phase::check "Health endpoint reports database connected" bash -c "curl -sf '${API_URL}/health' | jq -e '.database==\"connected\"' >/dev/null"
else
  testing::phase::check "Health endpoint responds" curl -sf "${API_URL}/health"
  testing::phase::check "Search endpoint returns payload" bash -c "curl -sf '${API_URL}/api/search?q=LATIN&limit=5' | grep -qi 'characters'"
  testing::phase::check "Category-filtered search returns payload" bash -c "curl -sf '${API_URL}/api/search?category=So&limit=3' | grep -qi 'characters'"
  testing::phase::check "Categories endpoint returns payload" bash -c "curl -sf '${API_URL}/api/categories' | grep -qi 'categories'"
  testing::phase::check "Blocks endpoint returns payload" bash -c "curl -sf '${API_URL}/api/blocks' | grep -qi 'blocks'"
  testing::phase::check "Character detail endpoint resolves U+0041" bash -c "curl -sf '${API_URL}/api/character/U+0041' | grep -qi 'U\\+0041'"
  testing::phase::check "Bulk range endpoint returns symbols" bash -c "curl -sf -X POST '${API_URL}/api/bulk/range' -H 'Content-Type: application/json' -d '{\"ranges\":[{\"start\":\"U+0041\",\"end\":\"U+005A\"}]}' | grep -qi 'characters'"
fi

# Summarise results
testing::phase::end_with_summary "Integration tests completed"
