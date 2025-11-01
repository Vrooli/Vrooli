#!/bin/bash
# Business workflow validation for Fall Foliage Explorer

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s" --require-runtime

API_BASE_URL=""
if ! API_BASE_URL=$(testing::connectivity::get_api_url "${TESTING_PHASE_SCENARIO_NAME}"); then
  testing::phase::add_error "Unable to determine API base URL"
  testing::phase::end_with_summary "Business tests incomplete"
fi
export FALL_FOLIAGE_API_BASE_URL="$API_BASE_URL"

UI_BASE_URL=""
if ! UI_BASE_URL=$(testing::connectivity::get_ui_url "${TESTING_PHASE_SCENARIO_NAME}"); then
  testing::phase::add_warning "Unable to determine UI base URL; UI checks skipped"
else
  export FALL_FOLIAGE_UI_BASE_URL="$UI_BASE_URL"
fi

HAS_JQ=false
if command -v jq >/dev/null 2>&1; then
  HAS_JQ=true
else
  testing::phase::add_warning "jq not available; detailed JSON assertions limited"
fi

if [ -n "$UI_BASE_URL" ]; then
  testing::phase::check "UI root accessible" bash -c 'curl -sf "$FALL_FOLIAGE_UI_BASE_URL/"'
else
  testing::phase::add_test skipped
fi

testing::phase::check "API health reports" bash -c '
  response=$(curl -sf "$FALL_FOLIAGE_API_BASE_URL/health") || exit 1
  echo "$response" | grep -q '"status"' || exit 1
  if command -v jq >/dev/null 2>&1; then
    echo "$response" | jq -e '.database == "healthy" and .redis == "healthy"' >/dev/null
  else
    echo "$response" | grep -q '"database":"healthy"'
  fi
'

if [ -x "./cli/foliage-explorer" ]; then
  testing::phase::check "CLI lists regions" bash -c 'API_URL="$FALL_FOLIAGE_API_BASE_URL" ./cli/foliage-explorer regions >/dev/null 2>&1'
  testing::phase::check "CLI status command succeeds" bash -c 'API_URL="$FALL_FOLIAGE_API_BASE_URL" ./cli/foliage-explorer status 1 >/dev/null 2>&1'
  testing::phase::check "CLI predict command succeeds" bash -c 'API_URL="$FALL_FOLIAGE_API_BASE_URL" ./cli/foliage-explorer predict 1 >/dev/null 2>&1'
else
  testing::phase::add_warning "CLI executable missing; skipping CLI checks"
  testing::phase::add_test skipped
  testing::phase::add_test skipped
  testing::phase::add_test skipped
fi

if [ "$HAS_JQ" = true ]; then
  testing::phase::check "Timeline API returns entries" bash -c 'curl -sf "$FALL_FOLIAGE_API_BASE_URL/api/timeline" | jq -e ".data | length > 0" >/dev/null'
else
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Business workflow validation completed"
