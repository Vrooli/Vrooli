#!/bin/bash
# Runs API integration smoke tests and CLI validation against the running scenario.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "300s" --require-runtime

SCENARIO_NAME="${TESTING_PHASE_SCENARIO_NAME}"
SCENARIO_STARTED=false
AUTOMATION_SCRIPT="${TESTING_PHASE_SCENARIO_DIR}/test/test-generation-endpoint.sh"
CLI_BATS="${TESTING_PHASE_SCENARIO_DIR}/cli/image-generation-pipeline.bats"

stop_if_started() {
  if [ "$SCENARIO_STARTED" = true ]; then
    vrooli scenario stop "$SCENARIO_NAME" >/dev/null 2>&1 || true
  fi
}

testing::phase::register_cleanup stop_if_started

if ! command -v vrooli >/dev/null 2>&1; then
  testing::phase::add_error "vrooli CLI not available; cannot manage scenario runtime"
  testing::phase::end_with_summary "Integration tests incomplete"
fi

if ! testing::core::is_scenario_running "$SCENARIO_NAME"; then
  if vrooli scenario start "$SCENARIO_NAME" --clean-stale >/dev/null; then
    if testing::core::wait_for_scenario "$SCENARIO_NAME" 180 >/dev/null 2>&1; then
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

if [ ! -x "$AUTOMATION_SCRIPT" ]; then
  testing::phase::add_error "Automation script missing or not executable: $AUTOMATION_SCRIPT"
  testing::phase::end_with_summary "Integration tests incomplete"
fi

if ! command -v curl >/dev/null 2>&1; then
  testing::phase::add_error "curl not available; required for API smoke tests"
  testing::phase::end_with_summary "Integration tests incomplete"
fi

if ! command -v jq >/dev/null 2>&1; then
  testing::phase::add_warning "jq not found; API payloads will not be pretty-printed"
fi

API_BASE="http://localhost:${API_PORT}"
testing::phase::check "API smoke workflow" bash -c "API_PORT=$API_PORT '$AUTOMATION_SCRIPT' '$API_BASE'"

if [ -f "$CLI_BATS" ]; then
  if command -v bats >/dev/null 2>&1; then
    testing::phase::check "CLI BATS suite" bash -c "cd '${TESTING_PHASE_SCENARIO_DIR}/cli' && API_PORT=$API_PORT bats $(basename "$CLI_BATS")"
  else
    testing::phase::add_warning "bats command not available; skipping CLI tests"
    testing::phase::add_test skipped
  fi
else
  testing::phase::add_warning "CLI BATS suite not found; skipping CLI validation"
  testing::phase::add_test skipped
fi

# Basic health verification after tests for sanity.
testing::phase::check "API health endpoint" curl -sf "$API_BASE/health"

testing::phase::end_with_summary "Integration validation completed"
