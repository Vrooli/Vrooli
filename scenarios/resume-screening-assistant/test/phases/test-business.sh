#!/bin/bash
# Validate end-to-end business workflow via custom tests.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

set -euo pipefail

testing::phase::init --target-time "180s" --require-runtime

SCENARIO_NAME="${TESTING_PHASE_SCENARIO_NAME}"
API_PORT_OUTPUT=$(vrooli scenario port "$SCENARIO_NAME" API_PORT 2>/dev/null || true)
API_PORT=$(echo "$API_PORT_OUTPUT" | awk -F= '/=/{print $2}' | tr -d ' ')
if [ -z "$API_PORT" ]; then
  API_PORT=$(echo "$API_PORT_OUTPUT" | tr -d '[:space:]')
fi

if [ -z "$API_PORT" ]; then
  testing::phase::add_error "Unable to resolve API_PORT for $SCENARIO_NAME"
  testing::phase::end_with_summary "Business checks incomplete"
fi

BUSINESS_TEST="custom-tests.sh"
if [ ! -f "$BUSINESS_TEST" ]; then
  testing::phase::add_error "custom-tests.sh missing"
  testing::phase::end_with_summary "Business checks incomplete"
fi

if testing::phase::check "Business workflow smoke" API_PORT="$API_PORT" bash -c 'source custom-tests.sh; test_resume_screening_assistant_workflow'; then
  :
fi

testing::phase::end_with_summary "Business workflow validation completed"
