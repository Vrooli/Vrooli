#!/bin/bash
# Runs API-level integration checks against a managed scenario instance.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "240s" --require-runtime

SCENARIO_NAME="${TESTING_PHASE_SCENARIO_NAME}"
TEST_DIR="${TESTING_PHASE_SCENARIO_DIR}/test"
ENDPOINT_SUITE="$TEST_DIR/test-api-endpoints.sh"

if [ ! -x "$ENDPOINT_SUITE" ]; then
  if [ -f "$ENDPOINT_SUITE" ]; then
    chmod +x "$ENDPOINT_SUITE"
  else
    testing::phase::add_error "Missing integration suite at $ENDPOINT_SUITE"
    testing::phase::end_with_summary "Integration tests incomplete"
  fi
fi

if ! command -v vrooli >/dev/null 2>&1; then
  testing::phase::add_error "vrooli CLI unavailable; cannot resolve API port"
  testing::phase::end_with_summary "Integration tests incomplete"
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

APP_BASE="http://localhost:${API_PORT}"

if APP_PERSONALIZER_API_BASE="$APP_BASE" "$ENDPOINT_SUITE"; then
  testing::phase::add_test passed
else
  testing::phase::add_error "API endpoint regression detected"
  testing::phase::add_test failed
fi


testing::phase::end_with_summary "Integration validation completed"
