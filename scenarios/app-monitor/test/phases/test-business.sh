#!/bin/bash
# Exercise business workflows such as CLI orchestration and alert endpoints.

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s" --require-runtime

SCENARIO_NAME="${TESTING_PHASE_SCENARIO_NAME}"
API_URL=$(testing::connectivity::get_api_url "$SCENARIO_NAME" || true)

if [ -z "$API_URL" ]; then
  testing::phase::add_error "Unable to resolve API port for $SCENARIO_NAME"
  testing::phase::end_with_summary "Business workflow validation incomplete"
fi

if command -v jq >/dev/null 2>&1; then
  summary_url="$API_URL/api/v1/apps/summary"
  testing::phase::check "Apps summary endpoint returns JSON" env SUMMARY_URL="$summary_url" bash -c 'curl -fsS "$SUMMARY_URL" | jq . >/dev/null'
else
  testing::phase::add_warning "jq not available; skipping JSON validation on apps summary"
  testing::phase::add_test skipped
fi

# App issue listing is central to monitoring incidents.
testing::phase::check "Issues endpoint responds" curl -fsS "$API_URL/api/v1/issues"

# CLI should surface high-level status without manual HTTP calls.
CLI_BIN="${TESTING_PHASE_SCENARIO_DIR}/cli/app-monitor"
if [ -x "$CLI_BIN" ]; then
  API_PORT_OUTPUT=$(vrooli scenario port "$SCENARIO_NAME" API_PORT 2>/dev/null || true)
  API_PORT=$(echo "$API_PORT_OUTPUT" | awk -F= '/=/{print $2}' | tr -d ' ')
  if [ -z "$API_PORT" ]; then
    API_PORT=$(echo "$API_PORT_OUTPUT" | tr -d '[:space:]')
  fi
  if [ -n "$API_PORT" ]; then
    testing::phase::check "CLI status command" env API_PORT="$API_PORT" CLI_BIN="$CLI_BIN" bash -c '"$CLI_BIN" status >/dev/null'
  else
    testing::phase::add_warning "Could not determine API_PORT for CLI validation"
    testing::phase::add_test skipped
  fi
else
  testing::phase::add_warning "app-monitor CLI not found; skipping CLI workflow validation"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Business workflow validation completed"
