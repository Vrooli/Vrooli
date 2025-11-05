#!/bin/bash
# Runs telemetry smoke automation and workflow version history contract checks.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "240s" --require-runtime

SCENARIO_NAME="${TESTING_PHASE_SCENARIO_NAME}"
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

if ! testing::phase::run_bas_automation_validations --scenario "$SCENARIO_NAME" --manage-runtime skip; then
  bas_rc=$?
  if [ "$bas_rc" -ne 0 ] && [ "$bas_rc" -ne 200 ]; then
    testing::phase::add_error "Browser Automation Studio workflow validations failed"
    testing::phase::end_with_summary "Integration automation failed"
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

if command -v node >/dev/null 2>&1; then
  export BAS_INTEGRATION_API_URL="http://localhost:${API_PORT}/api/v1"
  if node tests/workflow-version-history.integration.mjs; then
    log::success "✅ Workflow version history contract validated"
    testing::phase::add_test passed
    testing::phase::add_requirement --id BAS-VERSION-RESTORE --status passed --evidence "Workflow version history integration"
  else
    testing::phase::add_error "Workflow version history integration failed"
    testing::phase::add_test failed
    testing::phase::add_requirement --id BAS-VERSION-RESTORE --status failed --evidence "Workflow version history integration"
  fi
else
  testing::phase::add_warning "Node runtime missing; skipping workflow version history integration"
  testing::phase::add_test skipped
  testing::phase::add_requirement --id BAS-VERSION-RESTORE --status skipped --evidence "Workflow version history integration"
fi

testing::phase::end_with_summary "Integration tests completed"
