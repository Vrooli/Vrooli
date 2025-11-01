#!/bin/bash
# Run integration-focused validations for token-economy
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/runtime.sh"

testing::phase::init --target-time "180s"

# Go integration tests exercise end-to-end flows against database mocks
if [ -f "api/integration_test.go" ]; then
  testing::phase::check "Go integration suite" \
    bash -c 'cd api && go test -timeout 180s -run "Test(Token|Wallet|Transaction|Achievement|Analytics|Error|Concurrent|Cache)"'
else
  testing::phase::add_warning "api/integration_test.go missing; skipping Go integration suite"
  testing::phase::add_test skipped
fi

# Legacy shell integration, retained for smoke coverage
if [ -f "tests/integration-test.sh" ]; then
  testing::phase::check "Legacy integration shell" bash tests/integration-test.sh
else
  testing::phase::add_warning "Legacy integration shell absent"
  testing::phase::add_test skipped
fi

# If runtime is up, perform lightweight API checks
if command -v vrooli >/dev/null 2>&1 && testing::core::is_scenario_running "$TESTING_PHASE_SCENARIO_NAME"; then
  testing::runtime::discover_ports "$TESTING_PHASE_SCENARIO_NAME" 11080 11081 >/dev/null
  if [ -n "${API_PORT:-}" ]; then
    testing::phase::check "API health endpoint" curl -fsS "http://localhost:${API_PORT}/health"
  fi
  if [ -n "${UI_PORT:-}" ]; then
    testing::phase::check "UI root responds" curl -fsS "http://localhost:${UI_PORT}/"
  fi
else
  testing::phase::add_warning "Scenario runtime not detected; skipping live API/UI smoke checks"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Integration validation completed"
