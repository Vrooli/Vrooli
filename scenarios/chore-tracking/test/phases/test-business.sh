#!/bin/bash
# Validates end-to-end business workflows like schedule generation and achievement processing.
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "150s" --require-runtime

cd "$TESTING_PHASE_SCENARIO_DIR"

if ! command -v jq >/dev/null 2>&1; then
  testing::phase::add_warning "jq not available; skipping business workflow validation"
  testing::phase::add_test skipped
  testing::phase::end_with_summary "Business workflows skipped"
fi

API_BASE_URL=$(testing::connectivity::get_api_url "$TESTING_PHASE_SCENARIO_NAME")

# Baseline catalogue coverage
if testing::phase::check "Rewards catalogue available" bash -c "curl -sf ${API_BASE_URL}/api/rewards | jq -e 'length >= 1' >/dev/null"; then
  :
fi

if testing::phase::check "Achievements catalogue available" bash -c "curl -sf ${API_BASE_URL}/api/achievements | jq -e 'length >= 1' >/dev/null"; then
  :
fi

schedule_payload=$(jq -n \
  --arg ws "$(date -I)" \
  '{user_id: 1, week_start: $ws, preferences: {max_daily_chores: 3, preferred_time: "evening", skip_days: [6]}}')

schedule_response=$(curl -sf -X POST "${API_BASE_URL}/api/schedule/generate" \
  -H 'Content-Type: application/json' \
  -d "$schedule_payload" || true)
if echo "$schedule_response" | jq -e '.success == true and (.assignments | type == "array")' >/dev/null 2>&1; then
  testing::phase::add_test passed
else
  testing::phase::add_error "Weekly schedule generation failed"
  testing::phase::add_test failed
fi

points_payload='{"user_id":1,"chore_id":1}'
points_response=$(curl -sf -X POST "${API_BASE_URL}/api/points/calculate" \
  -H 'Content-Type: application/json' \
  -d "$points_payload" || true)
if echo "$points_response" | jq -e '.success == true and (.points_awarded | type == "number")' >/dev/null 2>&1; then
  testing::phase::add_test passed
else
  testing::phase::add_error "Points calculation failed"
  testing::phase::add_test failed
fi

achievements_payload='{"user_id":1}'
achievements_response=$(curl -sf -X POST "${API_BASE_URL}/api/achievements/process" \
  -H 'Content-Type: application/json' \
  -d "$achievements_payload" || true)
if echo "$achievements_response" | jq -e '.success == true and (.new_achievements | type == "array")' >/dev/null 2>&1; then
  testing::phase::add_test passed
else
  testing::phase::add_error "Achievement processing failed"
  testing::phase::add_test failed
fi

testing::phase::end_with_summary "Business workflow validation completed"
