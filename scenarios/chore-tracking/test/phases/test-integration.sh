#!/bin/bash
# Integration test phase ensuring core APIs remain healthy.
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s" --require-runtime

cd "$TESTING_PHASE_SCENARIO_DIR"

API_BASE_URL=$(testing::connectivity::get_api_url "$TESTING_PHASE_SCENARIO_NAME")
if [ -z "$API_BASE_URL" ]; then
  testing::phase::add_error "Unable to resolve API base URL"
  testing::phase::end_with_summary "Integration tests incomplete"
fi

testing::phase::check "Health endpoint responds" curl -sf "${API_BASE_URL}/health"

testing::phase::check "List chores" curl -sf "${API_BASE_URL}/api/chores"

testing::phase::check "List users" curl -sf "${API_BASE_URL}/api/users"

testing::phase::check "List achievements" curl -sf "${API_BASE_URL}/api/achievements"

testing::phase::check "List rewards" curl -sf "${API_BASE_URL}/api/rewards"

if command -v jq >/dev/null 2>&1; then
  testing::phase::check "Complete chore endpoint" bash -c "curl -sf -X POST '${API_BASE_URL}/api/chores/1/complete' -H 'Content-Type: application/json' -d '{\"user_id\":1}' | jq -e '.success == true' >/dev/null"
else
  testing::phase::add_warning "jq not available; skipping chore completion validation"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Integration tests completed"
