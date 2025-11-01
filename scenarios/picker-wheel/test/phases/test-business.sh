#!/bin/bash
# Validates core business flows: CLI spins, API CRUD, presets
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s" --require-runtime

SCENARIO_NAME="$TESTING_PHASE_SCENARIO_NAME"
API_URL="$(testing::connectivity::get_api_url "$SCENARIO_NAME" || true)"

if [ -z "$API_URL" ]; then
  testing::phase::add_error "Unable to discover API URL"
  testing::phase::end_with_summary "Business tests incomplete"
fi

if ! command -v curl >/dev/null 2>&1; then
  testing::phase::add_error "curl is required for business checks"
  testing::phase::end_with_summary "Business tests incomplete"
fi

CLI_BIN="${TESTING_PHASE_SCENARIO_DIR}/cli/picker-wheel"

if [ -x "$CLI_BIN" ]; then
  testing::phase::check "CLI dry-run spin (yes-or-no)" \
    bash -c 'output=$("$0" spin yes-or-no --dry-run); echo "$output" | grep -qi "result"' "$CLI_BIN"

  testing::phase::check "CLI weighted dry-run" \
    bash -c '$0 spin --options "High:10,Low:1" --dry-run >/dev/null' "$CLI_BIN"
else
  testing::phase::add_warning "Scenario CLI binary missing or not executable"
  testing::phase::add_test skipped
fi

if command -v jq >/dev/null 2>&1; then
  testing::phase::check "Create custom wheel via API" \
    bash -c 'curl -sf -X POST "${API_URL}/api/wheels" -H "Content-Type: application/json" -d "{\"name\":\"Business Test Wheel\",\"description\":\"Created during business tests\",\"options\":[{\"label\":\"A\",\"color\":\"#FF6B6B\",\"weight\":1},{\"label\":\"B\",\"color\":\"#4ECDC4\",\"weight\":1}]}" | jq -e .id >/dev/null'
else
  testing::phase::check "Create custom wheel via API" \
    curl -sf -X POST "${API_URL}/api/wheels" -H "Content-Type: application/json" -d '{"name":"Business Test Wheel","description":"Created during business tests","options":[{"label":"A","color":"#FF6B6B","weight":1},{"label":"B","color":"#4ECDC4","weight":1}]}'
  testing::phase::add_warning "jq unavailable; custom wheel response not parsed"
fi

PRESETS=(yes-or-no dinner-decider d20)
for preset in "${PRESETS[@]}"; do
  testing::phase::check "Preset wheel available: $preset" \
    curl -sf "${API_URL}/api/wheels/${preset}"
done

SPIN_HISTORY_PAYLOAD='{"wheel_id":"dinner-decider"}'
testing::phase::check "Record spin history" \
  curl -sf -X POST "${API_URL}/api/spin" -H "Content-Type: application/json" -d "${SPIN_HISTORY_PAYLOAD}"

testing::phase::check "History endpoint returns entries" \
  curl -sf "${API_URL}/api/history"

testing::phase::end_with_summary "Business logic validation completed"
