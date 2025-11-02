#!/bin/bash
# Exercises live API endpoints and core CLI interactions.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "240s" --require-runtime

CLI_PATH="${TESTING_PHASE_SCENARIO_DIR}/cli/scenario-dependency-analyzer"
API_BASE_URL=$(testing::connectivity::get_api_url "${TESTING_PHASE_SCENARIO_NAME}")

if ! command -v curl >/dev/null 2>&1; then
  testing::phase::add_error "curl is required for integration validation"
  testing::phase::add_test failed
  testing::phase::end_with_summary "Integration tests incomplete"
fi

testing::phase::check "API health endpoint" curl -sf "${API_BASE_URL}/health"

testing::phase::check "Dependency analysis endpoint" bash -c "curl -sf '${API_BASE_URL}/api/v1/scenarios/chart-generator/dependencies' >/dev/null"

testing::phase::check "Graph generation endpoint" bash -c "curl -sf '${API_BASE_URL}/api/v1/graph/combined' >/dev/null"

testing::phase::check "Proposed scenario analysis" bash -c "curl -sf -X POST '${API_BASE_URL}/api/v1/analyze/proposed' -H 'Content-Type: application/json' -d '{\"name\":\"test-scenario\",\"description\":\"Automated test\",\"requirements\":[\"postgres\",\"redis\"]}' >/dev/null"

if [ -x "$CLI_PATH" ]; then
  testing::phase::check "CLI help command" "$CLI_PATH" help
  testing::phase::check "CLI status command" "$CLI_PATH" status --json
else
  testing::phase::add_warning "CLI wrapper not executable; skipping CLI availability checks"
  testing::phase::add_test skipped
fi

if command -v jq >/dev/null 2>&1; then
  testing::phase::check "Dependencies response names scenario" bash -c "curl -sf '${API_BASE_URL}/api/v1/scenarios/chart-generator/dependencies' | jq -e '.scenario == \"chart-generator\"' >/dev/null"
else
  testing::phase::add_warning "jq not available; skipping response shape validation"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Integration validation completed"
