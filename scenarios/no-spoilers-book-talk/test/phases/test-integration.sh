#!/bin/bash
# Executes end-to-end book upload smoke tests
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/connectivity.sh"

testing::phase::init --target-time "300s" --require-runtime

SCENARIO_NAME="${TESTING_PHASE_SCENARIO_NAME}"

if ! command -v vrooli >/dev/null 2>&1; then
  testing::phase::add_error "vrooli CLI not available; cannot resolve runtime ports"
  testing::phase::end_with_summary "Integration tests incomplete"
fi

api_url=$(testing::connectivity::get_api_url "$SCENARIO_NAME" 2>/dev/null || true)
if [ -z "$api_url" ]; then
  testing::phase::add_error "Unable to resolve API URL for $SCENARIO_NAME"
  testing::phase::end_with_summary "Integration tests incomplete"
fi

API_PORT="${api_url##*:}"
API_PORT="${API_PORT#/}"
export API_PORT

testing::phase::check "API health endpoint" testing::connectivity::test_api "$SCENARIO_NAME"

testing::phase::check "Book upload workflow" bash test/test-book-upload.sh

testing::phase::end_with_summary "Integration validation completed"
