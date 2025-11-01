#!/bin/bash
# Integration validation for Fall Foliage Explorer

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "240s" --require-runtime

API_BASE_URL=""
if ! API_BASE_URL=$(testing::connectivity::get_api_url "${TESTING_PHASE_SCENARIO_NAME}"); then
  testing::phase::add_error "Unable to determine API base URL"
  testing::phase::end_with_summary "Integration tests incomplete"
fi

HAS_JQ=false
if command -v jq >/dev/null 2>&1; then
  HAS_JQ=true
else
  testing::phase::add_warning "jq not available; response-content checks will be skipped"
fi

testing::phase::check "API health endpoint" curl -sf "$API_BASE_URL/health"

testing::phase::check "Regions endpoint reachable" curl -sf "$API_BASE_URL/api/regions"

if [ "$HAS_JQ" = true ]; then
  testing::phase::check "Regions endpoint returns success" bash -lc 'curl -sf "$0" | jq -r ".status" | grep -q "success"' "$API_BASE_URL/api/regions"
else
  testing::phase::add_test skipped
fi

if [ "$HAS_JQ" = true ]; then
  testing::phase::check "Foliage endpoint returns success" bash -lc 'curl -sf "$0" | jq -r ".status" | grep -q "success"' "$API_BASE_URL/api/foliage?region_id=1"
else
  testing::phase::add_test skipped
fi

if [ "$HAS_JQ" = true ]; then
  testing::phase::check "Weather endpoint returns success" bash -lc 'curl -sf "$0" | jq -r ".status" | grep -q "success"' "$API_BASE_URL/api/weather?region_id=1"
else
  testing::phase::add_test skipped
fi

if [ "$HAS_JQ" = true ]; then
  testing::phase::check "Report submission persists" bash -lc '
    payload="{\"region_id\":1,\"foliage_status\":\"peak\",\"description\":\"Integration test\"}"
    curl -sf -X POST "$0" -H "Content-Type: application/json" -d "$payload" | jq -r ".status" | grep -q "success"
  ' "$API_BASE_URL/api/reports"
  testing::phase::check "Report listing returns success" bash -lc 'curl -sf "$0" | jq -r ".status" | grep -q "success"' "$API_BASE_URL/api/reports?region_id=1"
else
  testing::phase::add_test skipped
  testing::phase::add_test skipped
fi

if [ "$HAS_JQ" = true ]; then
  testing::phase::check "Prediction endpoint returns success" bash -lc '
    payload="{\"region_id\":1}"
    curl -sf -X POST "$0" -H "Content-Type: application/json" -d "$payload" | jq -r ".status" | grep -q "success"
  ' "$API_BASE_URL/api/predict"
else
  testing::phase::add_test skipped
fi

if [ "$HAS_JQ" = true ]; then
  testing::phase::check "Foliage intensity within expected range" bash -lc '
    response=$(curl -sf "$0") || exit 1
    intensity=$(echo "$response" | jq -r ".data.color_intensity")
    [[ "$intensity" =~ ^[0-9]+$ ]] && [ "$intensity" -ge 0 ] && [ "$intensity" -le 10 ]
  ' "$API_BASE_URL/api/foliage?region_id=2"
else
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Integration validation completed"
