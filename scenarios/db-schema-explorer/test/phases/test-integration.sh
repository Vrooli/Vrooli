#!/bin/bash
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "240s" --require-runtime

if ! command -v vrooli >/dev/null 2>&1; then
  testing::phase::add_error "vrooli CLI not available; cannot resolve scenario ports"
  testing::phase::end_with_summary "Integration tests incomplete"
fi

SCENARIO_NAME="${TESTING_PHASE_SCENARIO_NAME}"

API_PORT_OUTPUT=$(vrooli scenario port "$SCENARIO_NAME" API_PORT 2>/dev/null || true)
API_PORT=$(echo "$API_PORT_OUTPUT" | awk -F= '/=/{print $2}' | tr -d ' ')
if [ -z "$API_PORT" ]; then
  API_PORT=$(echo "$API_PORT_OUTPUT" | tr -d '[:space:]')
fi

if [ -z "$API_PORT" ]; then
  testing::phase::add_error "Unable to resolve API_PORT"
  testing::phase::end_with_summary "Integration tests incomplete"
fi

export API_PORT

if ! command -v jq >/dev/null 2>&1; then
  testing::phase::add_error "jq is required for integration checks"
  testing::phase::end_with_summary "Integration tests incomplete"
fi

if ! command -v curl >/dev/null 2>&1; then
  testing::phase::add_error "curl is required for integration checks"
  testing::phase::end_with_summary "Integration tests incomplete"
fi

testing::phase::check "API health endpoint" curl -sf "http://localhost:${API_PORT}/health"

testing::phase::check "Schema listing endpoint" curl -sf "http://localhost:${API_PORT}/api/v1/schema/list"

testing::phase::check "Query generation smoke" bash tests/test-query-generation.sh

testing::phase::end_with_summary "Integration validation completed"
