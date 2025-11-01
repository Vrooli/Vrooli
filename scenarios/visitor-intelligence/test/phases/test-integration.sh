#!/bin/bash
# Integration validation for visitor-intelligence scenario

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "300s" --require-runtime

SCENARIO_NAME="${TESTING_PHASE_SCENARIO_NAME}"
INTEGRATION_SCRIPT="test/integration-test.sh"

if [ ! -x "$INTEGRATION_SCRIPT" ]; then
  testing::phase::add_error "Integration harness missing or not executable at $INTEGRATION_SCRIPT"
  testing::phase::add_test failed
  testing::phase::end_with_summary "Integration tests incomplete"
fi

if ! command -v vrooli >/dev/null 2>&1; then
  testing::phase::add_error "vrooli CLI unavailable; cannot resolve scenario ports"
  testing::phase::add_test failed
  testing::phase::end_with_summary "Integration tests incomplete"
fi

API_PORT_OUTPUT=$(vrooli scenario port "$SCENARIO_NAME" API_PORT 2>/dev/null || true)
API_PORT=$(echo "$API_PORT_OUTPUT" | awk -F= '/=/{print $2}' | tr -d ' ')
if [ -z "$API_PORT" ]; then
  API_PORT=$(echo "$API_PORT_OUTPUT" | tr -d '[:space:]')
fi

if [ -z "$API_PORT" ]; then
  testing::phase::add_error "Unable to determine API_PORT for $SCENARIO_NAME"
  testing::phase::add_test failed
  testing::phase::end_with_summary "Integration tests incomplete"
fi

export API_PORT

if "$INTEGRATION_SCRIPT"; then
  testing::phase::add_test passed
else
  testing::phase::add_error "Integration harness reported failures"
  testing::phase::add_test failed
fi

testing::phase::end_with_summary "Integration tests completed"
