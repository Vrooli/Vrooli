#!/bin/bash
# Integration tests for text-tools using dynamic runtime discovery
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Require the scenario runtime because we exercise live endpoints
testing::phase::init --target-time "180s" --require-runtime

if ! command -v jq >/dev/null 2>&1; then
  testing::phase::add_error "jq is required for integration validation"
  testing::phase::end_with_summary "Integration tests aborted"
fi

if ! API_BASE_URL=$(testing::connectivity::get_api_url "$TESTING_PHASE_SCENARIO_NAME"); then
  testing::phase::add_error "Unable to determine API URL for $TESTING_PHASE_SCENARIO_NAME"
  testing::phase::end_with_summary "Integration tests aborted"
fi

export API_BASE_URL

echo "Using API base URL: $API_BASE_URL"

run_check() {
  local description="$1"
  shift
  testing::phase::check "$description" "$@"
}

run_check "API health endpoint" curl -sf "$API_BASE_URL/health"
run_check "Diff endpoint" bash -c "curl -sf -X POST \"${API_BASE_URL}/api/v1/text/diff\" -H 'Content-Type: application/json' -d '{\"text1\":\"hello world\",\"text2\":\"hello Vrooli\"}' | jq -e '.changes'"
run_check "Search endpoint" bash -c "curl -sf -X POST \"${API_BASE_URL}/api/v1/text/search\" -H 'Content-Type: application/json' -d '{\"text\":\"The quick brown fox\",\"pattern\":\"quick\"}' | jq -e '.matches'"
run_check "Transform endpoint" bash -c "curl -sf -X POST \"${API_BASE_URL}/api/v1/text/transform\" -H 'Content-Type: application/json' -d '{\"text\":\"hello\",\"transformations\":[{\"type\":\"case\",\"parameters\":{\"type\":\"upper\"}}]}' | jq -e '.result == \"HELLO\"'"
run_check "Analyze endpoint" bash -c "curl -sf -X POST \"${API_BASE_URL}/api/v1/text/analyze\" -H 'Content-Type: application/json' -d '{\"text\":\"test@example.com\",\"analyses\":[\"entities\"]}' | jq -e '.entities'"

# End phase with summary
testing::phase::end_with_summary "Integration tests completed"
