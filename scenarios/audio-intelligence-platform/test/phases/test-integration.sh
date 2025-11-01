#!/bin/bash
# Integration testing phase for audio-intelligence-platform
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "240s" --require-runtime

SCENARIO_NAME="$TESTING_PHASE_SCENARIO_NAME"
SCENARIO_STARTED=false

stop_scenario_if_started() {
  if [ "$SCENARIO_STARTED" = true ]; then
    vrooli scenario stop "$SCENARIO_NAME" >/dev/null 2>&1 || true
  fi
}

testing::phase::register_cleanup stop_scenario_if_started

if ! command -v vrooli >/dev/null 2>&1; then
  testing::phase::add_error "vrooli CLI not available; cannot manage scenario runtime"
  testing::phase::end_with_summary "Integration tests incomplete"
fi

if ! testing::core::is_scenario_running "$SCENARIO_NAME"; then
  if vrooli scenario start "$SCENARIO_NAME" --clean-stale >/dev/null; then
    if testing::core::wait_for_scenario "$SCENARIO_NAME" 120 >/dev/null 2>&1; then
      SCENARIO_STARTED=true
    else
      testing::phase::add_error "Scenario failed to become healthy after auto-start"
      testing::phase::end_with_summary "Integration tests incomplete"
    fi
  else
    testing::phase::add_error "Unable to auto-start scenario $SCENARIO_NAME"
    testing::phase::end_with_summary "Integration tests incomplete"
  fi
fi

API_PORT_OUTPUT=$(vrooli scenario port "$SCENARIO_NAME" API_PORT 2>/dev/null || true)
API_PORT=$(echo "$API_PORT_OUTPUT" | awk -F= '/=/{print $2}' | tr -d ' ')
if [ -z "$API_PORT" ]; then
  API_PORT=$(echo "$API_PORT_OUTPUT" | tr -d '[:space:]')
fi

if [ -z "$API_PORT" ]; then
  testing::phase::add_error "Unable to determine API_PORT for $SCENARIO_NAME"
  testing::phase::end_with_summary "Integration tests incomplete"
fi

API_BASE="http://localhost:${API_PORT}"

if testing::phase::check "API health endpoint" curl -sf "$API_BASE/health"; then
  :
fi

if testing::phase::check "Transcriptions endpoint" curl -sf "$API_BASE/api/transcriptions"; then
  :
fi

if testing::phase::check "Analysis endpoint smoke" env API_PORT="$API_PORT" bash test/test-analysis-endpoint.sh; then
  :
fi

testing::phase::end_with_summary "Integration validation completed"
