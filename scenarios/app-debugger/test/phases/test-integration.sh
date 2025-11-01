#!/bin/bash
# Integration checks against running app-debugger services

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s" --require-runtime

SCENARIO_NAME="$TESTING_PHASE_SCENARIO_NAME"
API_URL=""

if API_URL=$(testing::connectivity::get_api_url "$SCENARIO_NAME"); then
  testing::phase::check "API health endpoint" curl -sf "${API_URL}/health"
  testing::phase::check "Apps listing endpoint" curl -sf "${API_URL}/api/apps"
  testing::phase::check "Errors listing endpoint" curl -sf "${API_URL}/api/errors"
else
  testing::phase::add_error "Unable to determine API URL for ${SCENARIO_NAME}"
fi

testing::phase::end_with_summary "Integration tests completed"
