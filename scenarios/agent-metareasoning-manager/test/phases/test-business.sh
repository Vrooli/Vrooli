#!/bin/bash
# Validates runtime business workflows and critical endpoints.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

TESTING_PHASE_REQUIREMENT_TAG="business"

testing::phase::init --target-time "120s" --require-runtime

SCENARIO_NAME="$TESTING_PHASE_SCENARIO_NAME"
API_BASE_URL=""

if API_BASE_URL=$(testing::connectivity::get_api_url "$SCENARIO_NAME"); then
  testing::phase::add_test passed
else
  testing::phase::add_error "Unable to resolve API URL for $SCENARIO_NAME"
  testing::phase::add_test failed
  testing::phase::end_with_summary "Business validation incomplete"
fi

if testing::phase::check "API health endpoint" curl -sf "${API_BASE_URL}/health"; then
  :
fi

if testing::phase::check "Workflows endpoint responds" curl -sf "${API_BASE_URL}/workflows"; then
  :
fi

analyze_script="test/test-analyze-endpoint.sh"
if [ -x "$analyze_script" ] || [ -f "$analyze_script" ]; then
  api_port="${API_BASE_URL##*:}"
  api_port="${api_port%%/*}"
  if [ -z "$api_port" ]; then
    testing::phase::add_warning "Unable to derive API_PORT for pros/cons smoke test"
    testing::phase::add_test skipped
  else
    if API_PORT="$api_port" testing::phase::check "Pros/cons reasoning endpoint" bash "$analyze_script"; then
      :
    fi
  fi
else
  testing::phase::add_warning "Pros/cons reasoning smoke script missing"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Business validation completed"
