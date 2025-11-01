#!/bin/bash
# Validates core API integrations for resource-experimenter
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s" --require-runtime

SCENARIO_NAME="$TESTING_PHASE_SCENARIO_NAME"
API_BASE=""

if API_BASE=$(testing::connectivity::get_api_url "$SCENARIO_NAME"); then
  testing::phase::add_test passed
else
  testing::phase::add_error "Failed to resolve API port for $SCENARIO_NAME"
  testing::phase::end_with_summary "Integration tests incomplete"
fi

if ! command -v curl >/dev/null 2>&1; then
  testing::phase::add_error "curl is required for integration checks"
  testing::phase::end_with_summary "Integration tests incomplete"
fi

check_endpoint() {
  local path="$1"
  testing::phase::check "${path} returns 200" curl -sf --max-time 10 "${API_BASE}${path}" >/dev/null
}

check_endpoint /health
check_endpoint /api/experiments
check_endpoint /api/scenarios
check_endpoint /api/templates

testing::phase::end_with_summary "Integration tests completed"
