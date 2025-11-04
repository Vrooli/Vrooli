#!/bin/bash
# Exercises end-to-end workflows against a running scenario.
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "240s" --require-runtime
cd "$TESTING_PHASE_SCENARIO_DIR"

SCENARIO_NAME="$TESTING_PHASE_SCENARIO_NAME"
WORKFLOW_DEFINITION="automation/workflows/smoke.yaml"

# Execute workflow automations defined in the requirements registry. Replace
# SAMPLE-FUNC-001 with the requirement ID that this workflow validates.
if [ -f "$WORKFLOW_DEFINITION" ]; then
  if testing::phase::run_workflow_yaml \
    --file "$WORKFLOW_DEFINITION" \
    --label "Scenario smoke workflow" \
    --requirement SAMPLE-FUNC-001; then
    testing::phase::add_test passed
  else
    testing::phase::add_test failed
  fi
else
  testing::phase::add_warning "No $WORKFLOW_DEFINITION found; skip or update to match your automation naming"
  testing::phase::add_test skipped
fi

if command -v vrooli >/dev/null 2>&1; then
  API_PORT_OUTPUT=$(vrooli scenario port "$SCENARIO_NAME" API_PORT 2>/dev/null || true)
  API_PORT=$(echo "$API_PORT_OUTPUT" | awk -F= '/=/{print $2}' | tr -d ' ')
  if [ -z "$API_PORT" ]; then
    API_PORT=$(echo "$API_PORT_OUTPUT" | tr -d '[:space:]')
  fi

  if [ -n "$API_PORT" ]; then
    API_BASE_URL="http://localhost:${API_PORT}"
    testing::phase::check "API health endpoint" curl -sf "$API_BASE_URL/health"
  else
    testing::phase::add_warning "API_PORT not defined; skipping API connectivity checks"
    testing::phase::add_test skipped
  fi
else
  testing::phase::add_warning "vrooli CLI not available; skipping port discovery"
  testing::phase::add_test skipped
fi

# Add additional contract checks here (CLI workflows, replay assertions, etc.).

testing::phase::end_with_summary "Integration checks completed"
