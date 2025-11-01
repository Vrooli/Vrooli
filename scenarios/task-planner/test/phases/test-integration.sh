#!/bin/bash
# Basic integration smoke checks for task-planner API once runtime is available

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Require runtime but allow caller to decide lifecycle management via runner
if ! testing::phase::init --target-time "180s" --require-runtime; then
  exit $?
fi

SCENARIO_NAME="$TESTING_PHASE_SCENARIO_NAME"

if ! command -v vrooli >/dev/null 2>&1; then
  testing::phase::add_warning "vrooli CLI not available; skipping integration smoke checks"
  testing::phase::add_test skipped
  testing::phase::end_with_summary "Integration checks skipped"
fi

API_PORT_OUTPUT=$(vrooli scenario port "$SCENARIO_NAME" API_PORT 2>/dev/null || true)
API_PORT=$(echo "$API_PORT_OUTPUT" | awk -F= '/=/{print $2}' | tr -d ' ')
if [ -z "$API_PORT" ]; then
  API_PORT=$(echo "$API_PORT_OUTPUT" | tr -d '[:space:]')
fi

if [ -z "$API_PORT" ]; then
  testing::phase::add_warning "Unable to resolve API_PORT; skipping integration smoke"
  testing::phase::add_test skipped
  testing::phase::end_with_summary "Integration checks skipped"
fi

BASE_URL="http://localhost:${API_PORT}"

integration_checks=0
integration_failures=0

run_check() {
  local description="$1"
  shift

  if "$@"; then
    log::success "✅ ${description}"
    testing::phase::add_test passed
  else
    log::error "❌ ${description}"
    testing::phase::add_error "${description} failed"
    testing::phase::add_test failed
    integration_failures=$((integration_failures + 1))
  fi
  integration_checks=$((integration_checks + 1))
}

run_check "API health endpoint responds" curl -sf "${BASE_URL}/health"
run_check "Apps endpoint returns data" curl -sf "${BASE_URL}/api/apps"
run_check "Tasks endpoint returns data" curl -sf "${BASE_URL}/api/tasks"

if command -v task-planner >/dev/null 2>&1; then
  run_check "CLI status command succeeds" task-planner status >/dev/null
else
  testing::phase::add_warning "task-planner CLI not installed; skipping CLI smoke"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Integration smoke checks completed"
