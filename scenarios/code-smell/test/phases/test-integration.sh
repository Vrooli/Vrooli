#!/bin/bash
# Validate core service connectivity and lifecycle endpoints while the scenario is running.
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "240s" --require-runtime

SCENARIO_NAME="${TESTING_PHASE_SCENARIO_NAME}"

if ! command -v curl >/dev/null 2>&1; then
  testing::phase::add_error "curl CLI required for integration checks"
  testing::phase::end_with_summary "Integration checks could not run"
fi

API_PORT_OUTPUT=$(vrooli scenario port "$SCENARIO_NAME" API_PORT 2>/dev/null || true)
API_PORT=$(echo "$API_PORT_OUTPUT" | awk -F= '/=/{print $2}' | tr -d ' ')
if [ -z "$API_PORT" ]; then
  API_PORT=$(echo "$API_PORT_OUTPUT" | tr -d '[:space:]')
fi

if [ -z "$API_PORT" ]; then
  testing::phase::add_error "Unable to resolve API_PORT for $SCENARIO_NAME"
  testing::phase::end_with_summary "Integration tests incomplete"
fi

UI_PORT_OUTPUT=$(vrooli scenario port "$SCENARIO_NAME" UI_PORT 2>/dev/null || true)
UI_PORT=$(echo "$UI_PORT_OUTPUT" | awk -F= '/=/{print $2}' | tr -d ' ')
if [ -z "$UI_PORT" ]; then
  UI_PORT=$(echo "$UI_PORT_OUTPUT" | tr -d '[:space:]')
fi

API_URL="http://localhost:${API_PORT}/api/v1"
UI_URL="http://localhost:${UI_PORT}"

# API health endpoints
testing::phase::check "API health endpoint responds" curl -fsS "$API_URL/health"
testing::phase::check "API readiness endpoint responds" curl -fsS "$API_URL/health/ready"

# Primary rules endpoint returns JSON payload
if command -v jq >/dev/null 2>&1; then
  testing::phase::check "Rules endpoint returns rule set" bash -c 'curl -fsS "$0" | jq -e ".rules" >/dev/null' "$API_URL/code-smell/rules"
else
  testing::phase::add_warning "jq not available; skipping structured validation of /code-smell/rules"
  testing::phase::check "Rules endpoint responds" curl -fsS "$API_URL/code-smell/rules"
fi

# UI landing page basic availability
if [ -n "$UI_PORT" ]; then
  testing::phase::check "UI landing page reachable" curl -fsS "$UI_URL"
else
  testing::phase::add_warning "UI port not published; skipping UI availability check"
  testing::phase::add_test skipped
fi

# CLI sanity checks
if [ -x "cli/code-smell" ]; then
  testing::phase::check "CLI status command" bash -c 'cd "$0" && ./cli/code-smell status --json >/dev/null' "$TESTING_PHASE_SCENARIO_DIR"
else
  testing::phase::add_warning "cli/code-smell binary not executable; skipping CLI status"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Integration validation completed"
