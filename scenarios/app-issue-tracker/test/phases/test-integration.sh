#!/bin/bash
# API integration checks for app-issue-tracker
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "240s" --require-runtime

SCENARIO_NAME="$TESTING_PHASE_SCENARIO_NAME"
API_URL=""

if ! command -v curl >/dev/null 2>&1 || ! command -v jq >/dev/null 2>&1; then
  testing::phase::add_warning "curl and jq are required for integration smoke tests"
  testing::phase::add_test skipped
  testing::phase::end_with_summary "Integration tests skipped"
fi

if API_URL=$(testing::connectivity::get_api_url "$SCENARIO_NAME"); then
  if testing::connectivity::test_api "$SCENARIO_NAME"; then
    testing::phase::add_test passed
  else
    testing::phase::add_error "API health check failed"
    testing::phase::add_test failed
  fi
else
  testing::phase::add_error "Unable to discover API port for $SCENARIO_NAME"
  testing::phase::add_test failed
  testing::phase::end_with_summary "Integration tests incomplete"
fi

API_BASE="${API_URL%/}/api"

if testing::phase::check "Semantic search endpoint" env ISSUE_TRACKER_API_URL="$API_BASE" bash "$TESTING_PHASE_SCENARIO_DIR/test/test-search-endpoint.sh"; then
  :
fi

testing::phase::end_with_summary "Integration tests completed"
