#!/bin/bash
# Exercises API/UI surface and CRUD interactions end to end
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s" --require-runtime

SCENARIO_NAME="$TESTING_PHASE_SCENARIO_NAME"
API_URL="$(testing::connectivity::get_api_url "$SCENARIO_NAME" || true)"
UI_URL="$(testing::connectivity::get_ui_url "$SCENARIO_NAME" || true)"

if [ -z "$API_URL" ]; then
  testing::phase::add_error "Unable to discover API URL"
  testing::phase::end_with_summary "Integration tests incomplete"
fi

if ! command -v curl >/dev/null 2>&1; then
  testing::phase::add_error "curl is required for integration checks"
  testing::phase::end_with_summary "Integration tests incomplete"
fi

testing::phase::check "API health endpoint" curl -sf "${API_URL}/health"

if command -v jq >/dev/null 2>&1; then
  testing::phase::check "Wheel listing returns data" \
    bash -c 'curl -sf "${API_URL}/api/wheels" | jq -e ". | length >= 1" >/dev/null'
else
  testing::phase::check "Wheel listing responds" curl -sf "${API_URL}/api/wheels"
  testing::phase::add_warning "jq unavailable; skipping wheel count assertion"
fi

SPIN_PAYLOAD='{"wheel_id":"yes-or-no"}'
if command -v jq >/dev/null 2>&1; then
  testing::phase::check "Spin API returns result" \
    bash -c 'curl -sf -X POST "${API_URL}/api/spin" -H "Content-Type: application/json" -d "${SPIN_PAYLOAD}" | jq -e ".result" >/dev/null'
else
  testing::phase::check "Spin API responds" \
    curl -sf -X POST "${API_URL}/api/spin" -H "Content-Type: application/json" -d "${SPIN_PAYLOAD}"
  testing::phase::add_warning "jq unavailable; result parsing skipped"
fi

if command -v jq >/dev/null 2>&1; then
  CUSTOM_WHEEL_RESPONSE=$(curl -sf -X POST "${API_URL}/api/wheels" \
    -H "Content-Type: application/json" \
    -d '{
      "name": "Integration Test Wheel",
      "description": "Created during integration testing",
      "options": [
        {"label": "Option A", "color": "#FF6B6B", "weight": 1},
        {"label": "Option B", "color": "#4ECDC4", "weight": 1}
      ],
      "theme": "integration"
    }' || true)

  if echo "$CUSTOM_WHEEL_RESPONSE" | jq -e '.id' >/dev/null 2>&1; then
    NEW_WHEEL_ID=$(echo "$CUSTOM_WHEEL_RESPONSE" | jq -r '.id')
    testing::phase::add_test passed
    testing::phase::check "New wheel retrievable" \
      curl -sf "${API_URL}/api/wheels/${NEW_WHEEL_ID}"
  else
    testing::phase::add_error "Custom wheel creation failed"
    testing::phase::add_test failed
  fi
else
  testing::phase::add_warning "jq unavailable; skipping custom wheel validation"
  testing::phase::add_test skipped
fi

if [ -n "$UI_URL" ]; then
  testing::phase::check "UI root responds" curl -sf "$UI_URL/"
else
  testing::phase::add_warning "Unable to discover UI URL; skipping UI connectivity"
  testing::phase::add_test skipped
fi

testing::phase::check "History endpoint responds" curl -sf "${API_URL}/api/history"

testing::phase::end_with_summary "Integration validation completed"
