#!/bin/bash
# Execute API integration tests while the scenario runtime is available.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "300s" --require-runtime

SCENARIO_NAME="${TESTING_PHASE_SCENARIO_NAME}"
SCENARIO_DIR="${TESTING_PHASE_SCENARIO_DIR}"
INTEGRATION_SCRIPT="$SCENARIO_DIR/test/integration-test.sh"

if [ ! -x "$INTEGRATION_SCRIPT" ]; then
  testing::phase::add_warning "Integration script missing at test/integration-test.sh"
  testing::phase::add_test skipped
  testing::phase::end_with_summary "Integration tests skipped"
fi

if ! command -v jq >/dev/null 2>&1; then
  testing::phase::add_warning "jq not installed; integration tests require jq"
  testing::phase::add_test skipped
  testing::phase::end_with_summary "Integration tests skipped"
fi

API_PORT_OUTPUT=$(vrooli scenario port "$SCENARIO_NAME" API_PORT 2>/dev/null || true)
API_PORT=$(echo "$API_PORT_OUTPUT" | awk -F= '/=/{print $2}' | tr -d ' ')
if [ -z "$API_PORT" ]; then
  API_PORT=$(echo "$API_PORT_OUTPUT" | tr -d '[:space:]')
fi

if [ -z "$API_PORT" ]; then
  testing::phase::add_error "Unable to resolve API port for $SCENARIO_NAME"
  testing::phase::end_with_summary "Integration tests incomplete"
fi

testing::phase::check "Calendar integration suite" bash -c "API_PORT=$API_PORT '$INTEGRATION_SCRIPT'"

testing::phase::end_with_summary "Integration tests completed"
