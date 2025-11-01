#!/bin/bash
# Exercises running services to verify health and critical endpoints
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s" --require-runtime

SCENARIO_NAME="${TESTING_PHASE_SCENARIO_NAME}"
SCENARIO_STARTED=false

stop_if_started() {
  if [ "$SCENARIO_STARTED" = true ]; then
    vrooli scenario stop "$SCENARIO_NAME" >/dev/null 2>&1 || true
  fi
}

testing::phase::register_cleanup stop_if_started

if ! command -v vrooli >/dev/null 2>&1; then
  testing::phase::add_error "vrooli CLI not available; cannot manage runtime"
  testing::phase::end_with_summary "Integration checks aborted"
fi

if ! testing::core::is_scenario_running "$SCENARIO_NAME"; then
  if vrooli scenario start "$SCENARIO_NAME" --clean-stale >/dev/null; then
    if testing::core::wait_for_scenario "$SCENARIO_NAME" 120 >/dev/null 2>&1; then
      SCENARIO_STARTED=true
    else
      testing::phase::add_error "Scenario failed to become healthy after auto-start"
      testing::phase::end_with_summary "Integration checks incomplete"
    fi
  else
    testing::phase::add_error "Unable to auto-start scenario $SCENARIO_NAME"
    testing::phase::end_with_summary "Integration checks incomplete"
  fi
fi

API_PORT_OUTPUT=$(vrooli scenario port "$SCENARIO_NAME" API_PORT 2>/dev/null || true)
API_PORT=$(echo "$API_PORT_OUTPUT" | awk -F= '/=/{print $2}' | tr -d ' ')
if [ -z "$API_PORT" ]; then
  API_PORT=$(echo "$API_PORT_OUTPUT" | tr -d '[:space:]')
fi

if [ -z "$API_PORT" ]; then
  testing::phase::add_error "Unable to resolve API_PORT for $SCENARIO_NAME"
  testing::phase::end_with_summary "Integration checks incomplete"
fi

BASE_URL="http://localhost:${API_PORT}"

if testing::phase::check "API health endpoint" curl -fsS "${BASE_URL}/health"; then
  :
fi

if testing::phase::check "Encryption status endpoint" curl -fsS "${BASE_URL}/api/v1/health/encryption"; then
  :
fi

if testing::phase::check "Authentication status endpoint" curl -fsS "${BASE_URL}/api/v1/auth/status"; then
  :
fi

testing::phase::end_with_summary "Integration checks completed"
