#!/bin/bash
# Exercise API endpoints and higher level integrations.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/connectivity.sh"

set -euo pipefail

testing::phase::init --target-time "240s" --require-runtime

SCENARIO_NAME="${TESTING_PHASE_SCENARIO_NAME}"
API_PORT_OUTPUT=$(vrooli scenario port "$SCENARIO_NAME" API_PORT 2>/dev/null || true)
API_PORT=$(echo "$API_PORT_OUTPUT" | awk -F= '/=/{print $2}' | tr -d ' ')
if [ -z "$API_PORT" ]; then
  API_PORT=$(echo "$API_PORT_OUTPUT" | tr -d '[:space:]')
fi

if [ -z "$API_PORT" ]; then
  testing::phase::add_error "Unable to resolve API_PORT for $SCENARIO_NAME"
  testing::phase::end_with_summary "Integration checks incomplete"
fi

API_URL="http://localhost:${API_PORT}"

# Sanity check overall connectivity first.
testing::phase::check "API health endpoint" curl -sf "${API_URL}/health"
testing::phase::check "Jobs API endpoint" curl -sf "${API_URL}/api/jobs"
testing::phase::check "Candidates API endpoint" curl -sf "${API_URL}/api/candidates"
testing::phase::check "Search API basic query" curl -sf "${API_URL}/api/search?query=engineer&type=both"

# Deep search validation script (validates JSON structure & variant queries).
if ! testing::phase::check "Search endpoint regression suite" API_PORT="$API_PORT" bash test/test-search-endpoint.sh; then
  testing::phase::end_with_summary "Integration validation failed"
fi

testing::phase::end_with_summary "Integration validation completed"
