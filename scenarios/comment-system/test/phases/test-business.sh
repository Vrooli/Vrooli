#!/bin/bash
# Validate end-to-end comment workflows and CLI touchpoints
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "240s" --require-runtime

SCENARIO_NAME="$TESTING_PHASE_SCENARIO_NAME"
SCENARIO_DIR="$TESTING_PHASE_SCENARIO_DIR"
API_URL=$(testing::connectivity::get_api_url "$SCENARIO_NAME")
export API_URL

if [[ -z "$API_URL" ]]; then
  testing::phase::add_error "Unable to resolve API URL for $SCENARIO_NAME"
  testing::phase::add_test failed
  testing::phase::end_with_summary "Business workflow checks could not run"
fi

if [[ ! -x "$SCENARIO_DIR/test/test-threading.sh" ]]; then
  testing::phase::add_warning "Threading smoke script not found; skipping"
  testing::phase::add_test skipped
else
  if bash "$SCENARIO_DIR/test/test-threading.sh"; then
    testing::phase::add_test passed
  else
    testing::phase::add_error "Threading workflow tests failed"
    testing::phase::add_test failed
  fi
fi

CLI_BIN="$SCENARIO_DIR/cli/comment-system"
if [[ -x "$CLI_BIN" ]]; then
  if "$CLI_BIN" help >/dev/null 2>&1 && "$CLI_BIN" status --json >/dev/null 2>&1; then
    testing::phase::add_test passed
  else
    testing::phase::add_error "CLI smoke commands failed"
    testing::phase::add_test failed
  fi
else
  testing::phase::add_warning "CLI binary missing; skipping CLI smoke checks"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Business workflow validation completed"
