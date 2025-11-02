#!/bin/bash
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s"

scenario_name="$TESTING_PHASE_SCENARIO_NAME"
if ! testing::core::is_scenario_running "$scenario_name"; then
  testing::phase::add_warning "Scenario '$scenario_name' is not running; skipping integration checks"
  testing::phase::add_test skipped
  testing::phase::end_with_summary "Integration tests skipped"
fi

API_BASE_URL=$(testing::connectivity::get_api_url "$scenario_name")

if testing::phase::check "API health endpoint" curl -sf "$API_BASE_URL/health"; then
  :
fi

if testing::phase::check "Jobs listing responds" curl -sf "$API_BASE_URL/api/v1/jobs"; then
  :
fi

testing::phase::end_with_summary "Integration checks completed"
