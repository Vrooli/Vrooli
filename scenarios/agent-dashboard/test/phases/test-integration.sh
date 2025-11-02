#!/bin/bash
# Exercise core API/UI integration points while the scenario is running
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "240s" --require-runtime

SCENARIO_NAME="$TESTING_PHASE_SCENARIO_NAME"
API_BASE_URL=$(testing::connectivity::get_api_url "$SCENARIO_NAME" || true)
UI_BASE_URL=$(testing::connectivity::get_ui_url "$SCENARIO_NAME" || true)
JQ_AVAILABLE=false

if [ -z "$API_BASE_URL" ]; then
  testing::phase::add_error "Unable to discover API URL for $SCENARIO_NAME"
  testing::phase::end_with_summary "Integration tests incomplete"
fi

if command -v jq >/dev/null 2>&1; then
  JQ_AVAILABLE=true
else
  testing::phase::add_warning "jq not available; JSON payload validation will be skipped"
fi

testing::phase::check "API health endpoint responds" curl -fsS "${API_BASE_URL}/health"

testing::phase::check "Capabilities endpoint reachable" curl -fsS "${API_BASE_URL}/api/v1/capabilities"

testing::phase::check "Search endpoint handles query" curl -fsS "${API_BASE_URL}/api/v1/agents/search?capability=analysis"

if [ "$JQ_AVAILABLE" = true ]; then
  testing::phase::check "Version endpoint returns success payload" env API_BASE_URL="$API_BASE_URL" bash -c 'curl -fsS "$API_BASE_URL/api/v1/version" | jq -e ".success == true"'
  testing::phase::check "Agents endpoint returns list" env API_BASE_URL="$API_BASE_URL" bash -c 'curl -fsS "$API_BASE_URL/api/v1/agents" | jq -e ".success == true and (.data.total >= 0)"'
else
  testing::phase::add_test skipped
  testing::phase::add_test skipped
fi

if [ -n "$UI_BASE_URL" ]; then
  testing::phase::check "UI health endpoint responds" curl -fsS "${UI_BASE_URL}/health"
else
  testing::phase::add_warning "UI port not published; skipping UI connectivity checks"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Integration validation completed"
