#!/bin/bash
# Basic integration smoke checks for accessibility-compliance-hub.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s"

SCENARIO_NAME="${TESTING_PHASE_SCENARIO_NAME}"

if ! command -v vrooli >/dev/null 2>&1; then
  testing::phase::add_warning "vrooli CLI unavailable; skipping integration checks"
  testing::phase::add_test skipped
  testing::phase::end_with_summary "Integration checks skipped"
fi

API_PORT_OUTPUT=$(vrooli scenario port "$SCENARIO_NAME" API_PORT 2>/dev/null || true)
API_PORT=$(echo "$API_PORT_OUTPUT" | awk -F= '/=/{print $2}' | tr -d ' ')
if [ -z "$API_PORT" ]; then
  API_PORT=$(echo "$API_PORT_OUTPUT" | tr -d '[:space:]')
fi

if [ -z "$API_PORT" ]; then
  testing::phase::add_warning "Unable to resolve API_PORT; skipping API smoke tests"
  testing::phase::add_test skipped
  testing::phase::end_with_summary "Integration checks skipped"
fi

API_URL="http://localhost:${API_PORT}"

if command -v curl >/dev/null 2>&1; then
  testing::phase::check "API health endpoint responds" curl -sf "${API_URL}/health"
else
  testing::phase::add_warning "curl missing; cannot exercise API"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Integration checks completed"
