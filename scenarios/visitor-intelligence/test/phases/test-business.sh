#!/bin/bash
# Business workflow validation for visitor-intelligence

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "240s" --require-runtime

if ! command -v vrooli >/dev/null 2>&1; then
  testing::phase::add_error "vrooli CLI unavailable; cannot resolve scenario ports"
  testing::phase::add_test failed
  testing::phase::end_with_summary "Business workflow checks incomplete"
fi

SCENARIO_NAME="${TESTING_PHASE_SCENARIO_NAME}"
API_PORT_OUTPUT=$(vrooli scenario port "$SCENARIO_NAME" API_PORT 2>/dev/null || true)
API_PORT=$(echo "$API_PORT_OUTPUT" | awk -F= '/=/{print $2}' | tr -d ' ')
if [ -z "$API_PORT" ]; then
  API_PORT=$(echo "$API_PORT_OUTPUT" | tr -d '[:space:]')
fi

if [ -z "$API_PORT" ]; then
  testing::phase::add_error "Unable to determine API_PORT for $SCENARIO_NAME"
  testing::phase::add_test failed
  testing::phase::end_with_summary "Business workflow checks incomplete"
fi

API_BASE="http://localhost:${API_PORT}/api/v1"
TEST_SCENARIO="phase-business"
FINGERPRINT="biz-$RANDOM-$(date +%s)"

TRACK_PAYLOAD=$(cat <<JSON
{
  "fingerprint": "${FINGERPRINT}",
  "event_type": "pageview",
  "scenario": "${TEST_SCENARIO}",
  "page_url": "https://example.com/business-phase",
  "properties": {
    "phase": "business",
    "source": "phased-testing"
  }
}
JSON
)

tracking_cmd=(curl -sSf -X POST -H "Content-Type: application/json" -d "$TRACK_PAYLOAD" "${API_BASE}/visitor/track")
if testing::phase::check "Track visitor event" "${tracking_cmd[@]}"; then
  true
fi

sleep 1

analytics_cmd=(curl -sSf "${API_BASE}/analytics/scenario/${TEST_SCENARIO}?timeframe=7d")
if testing::phase::check "Fetch scenario analytics" "${analytics_cmd[@]}"; then
  true
fi

CLI_PATH="${TESTING_PHASE_SCENARIO_DIR}/cli/visitor-intelligence"
if [ -x "$CLI_PATH" ]; then
  status_cmd=($CLI_PATH status --json)
  if testing::phase::check "CLI status reports health" "${status_cmd[@]}"; then
    true
  fi

  analytics_cli_cmd=($CLI_PATH analytics "${TEST_SCENARIO}" --json)
  if ! testing::phase::check "CLI analytics command" "${analytics_cli_cmd[@]}"; then
    testing::phase::add_warning "CLI analytics command failed; ensure data processing workers are available"
  fi
else
  testing::phase::add_warning "CLI binary missing or non-executable"
  testing::phase::add_test skipped
fi

teardown_payload=$(cat <<JSON
{
  "fingerprint": "${FINGERPRINT}",
  "event_type": "cleanup",
  "scenario": "${TEST_SCENARIO}",
  "page_url": "https://example.com/business-phase"
}
JSON
)

curl -sSf -X POST -H "Content-Type: application/json" -d "$teardown_payload" "${API_BASE}/visitor/track" >/dev/null 2>&1 || true

testing::phase::end_with_summary "Business workflow validation completed"
