#!/bin/bash
# Performance baseline checks for Secrets Manager
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s" --require-runtime

SCENARIO_NAME="${TESTING_PHASE_SCENARIO_NAME}"
API_URL=$(testing::connectivity::get_api_url "$SCENARIO_NAME" || true)

if [ -z "$API_URL" ]; then
  testing::phase::add_warning "Unable to resolve API URL; skipping performance scan"
  testing::phase::add_test skipped
  testing::phase::end_with_summary "Performance checks skipped"
fi

API_PORT="${API_URL##*:}"
export API_PORT

CUSTOM_TESTS="${TESTING_PHASE_SCENARIO_DIR}/custom-tests.sh"
if [ -f "$CUSTOM_TESTS" ]; then
  source "$CUSTOM_TESTS"
  if test_performance_large_scan; then
    testing::phase::add_test passed
  else
    testing::phase::add_error "Large scan performance target not met"
    testing::phase::add_test failed
  fi
else
  testing::phase::add_warning "custom-tests.sh missing; performance baseline unavailable"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Performance validation completed"
