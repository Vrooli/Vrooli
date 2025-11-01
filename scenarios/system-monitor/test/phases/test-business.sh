#!/bin/bash
# Business-layer validation ensures core workflows respond as expected.

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s" --require-runtime

cd "$TESTING_PHASE_SCENARIO_DIR"

scenario_name="$TESTING_PHASE_SCENARIO_NAME"
api_url="$(testing::connectivity::get_api_url "$scenario_name" 2>/dev/null || true)"
ui_url="$(testing::connectivity::get_ui_url "$scenario_name" 2>/dev/null || true)"

if [ -n "$api_url" ]; then
  testing::phase::check "API health endpoint" curl -sf "${api_url}/health"

  if command -v jq >/dev/null 2>&1; then
    testing::phase::check "Metrics endpoint returns telemetry" bash -c '
      api="$1"
      response=$(curl -sf "${api}/api/metrics/current")
      jq -e ".cpu_usage and .memory_usage and .timestamp" <<<"$response" >/dev/null
    ' _ "$api_url"
  else
    testing::phase::add_warning "jq not available; skipping metrics schema validation"
    testing::phase::add_test skipped
  fi

  testing::phase::check "Report generation succeeds" bash -c '
    api="$1"
    curl -sf -X POST "${api}/api/reports/generate" \
      -H "Content-Type: application/json" \
      -d "{\"type\":\"daily\"}"
  ' _ "$api_url"
else
  testing::phase::add_error "Unable to determine API URL for ${scenario_name}"
  testing::phase::add_test failed
fi

if [ -n "$ui_url" ]; then
  testing::phase::check "Dashboard responds" curl -sf "$ui_url"
else
  testing::phase::add_warning "Unable to determine UI URL; skipping dashboard check"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Business logic validation completed"
