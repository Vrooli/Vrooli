#!/bin/bash
# Exercise integration flows requiring the running API
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
  testing::phase::end_with_summary "Integration checks could not run"
fi

if ! command -v curl >/dev/null 2>&1; then
  testing::phase::add_error "curl is required for integration validation"
  testing::phase::add_test failed
  testing::phase::end_with_summary "Integration checks could not run"
fi

if [[ ! -x "$SCENARIO_DIR/test/test-comment-crud.sh" ]]; then
  testing::phase::add_error "Missing test/test-comment-crud.sh"
  testing::phase::add_test failed
else
  if bash "$SCENARIO_DIR/test/test-comment-crud.sh"; then
    testing::phase::add_test passed
  else
    testing::phase::add_error "Comment CRUD regression detected"
    testing::phase::add_test failed
  fi
fi

if [[ ! -x "$SCENARIO_DIR/test/test-integration.sh" ]]; then
  testing::phase::add_warning "integration smoke script not found; skipping"
  testing::phase::add_test skipped
else
  if command -v jq >/dev/null 2>&1; then
    if bash "$SCENARIO_DIR/test/test-integration.sh"; then
      testing::phase::add_test passed
    else
      testing::phase::add_error "Cross-scenario integration checks failed"
      testing::phase::add_test failed
    fi
  else
    testing::phase::add_warning "jq unavailable; skipping integration dependency checks"
    testing::phase::add_test skipped
  fi
fi

testing::phase::end_with_summary "Integration validation completed"
