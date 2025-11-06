#!/bin/bash
# Executes end-to-end smoke checks against the running scenario services.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "240s" --require-runtime

SCENARIO_NAME="$TESTING_PHASE_SCENARIO_NAME"
SCENARIO_DIR="$TESTING_PHASE_SCENARIO_DIR"

source "${APP_ROOT}/scripts/scenarios/testing/shell/connectivity.sh"

API_URL=""
if API_URL=$(testing::connectivity::get_api_url "$SCENARIO_NAME" 2>/dev/null); then
  testing::phase::check "API health endpoint" curl -fsS "$API_URL/health"
  testing::phase::check "Documents API endpoint" curl -fsS "$API_URL/api/documents"
  testing::phase::check "Jobs API endpoint" curl -fsS "$API_URL/api/jobs"
  testing::phase::check "Workflows API endpoint" curl -fsS "$API_URL/api/workflows"
else
  testing::phase::add_error "Unable to resolve API URL for $SCENARIO_NAME"
  testing::phase::add_test failed
fi

# Validate that the CLI is installed and responsive when available.
if command -v secure-document-processing >/dev/null 2>&1; then
  testing::phase::check "CLI status command" secure-document-processing status --format json >/dev/null
else
  testing::phase::add_warning "CLI binary not on PATH; skipping CLI validation"
  testing::phase::add_test skipped
fi

# Run custom business workflow checks if available.
CUSTOM_TESTS="$SCENARIO_DIR/custom-tests.sh"
if [ -f "$CUSTOM_TESTS" ]; then
  testing::phase::check "Custom workflow validation" bash -c 'source "$1" && { declare -f run_custom_tests >/dev/null && run_custom_tests; }' _ "$CUSTOM_TESTS"
else
  testing::phase::add_warning "Custom workflow script missing; skipping custom validation"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Integration tests completed"
