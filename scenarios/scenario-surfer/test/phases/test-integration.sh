#!/bin/bash
# Connectivity checks for Scenario Surfer runtime components

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s" --skip-runtime-check

if command -v vrooli >/dev/null 2>&1 && testing::core::is_scenario_running "$TESTING_PHASE_SCENARIO_NAME"; then
  if ! command -v curl >/dev/null 2>&1; then
    testing::phase::add_warning "curl not available; cannot perform HTTP integration checks"
    testing::phase::add_test skipped
    testing::phase::end_with_summary "Integration validation completed"
  fi

  API_PORT_OUTPUT=$(vrooli scenario port "$TESTING_PHASE_SCENARIO_NAME" API_PORT 2>/dev/null || true)
  API_PORT=$(echo "$API_PORT_OUTPUT" | awk -F= '/=/{print $2}' | tr -d ' ')
  if [ -z "$API_PORT" ]; then
    API_PORT=$(echo "$API_PORT_OUTPUT" | tr -d '[:space:]')
  fi

  if [ -n "$API_PORT" ]; then
    testing::phase::check "API health endpoint responds" curl -sf "http://localhost:${API_PORT}/health"
    testing::phase::check "Scenario discovery endpoint returns data" curl -sf "http://localhost:${API_PORT}/api/v1/scenarios/healthy"
  else
    testing::phase::add_warning "Unable to resolve API_PORT; skipping HTTP checks"
    testing::phase::add_test skipped
  fi
else
  testing::phase::add_warning "Scenario Surfer runtime not detected; run with --manage-runtime to execute integration checks"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Integration validation completed"
