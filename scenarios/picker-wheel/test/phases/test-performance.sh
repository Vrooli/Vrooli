#!/bin/bash
# Lightweight performance guardrails for picker-wheel endpoints
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s" --require-runtime

SCENARIO_NAME="$TESTING_PHASE_SCENARIO_NAME"
API_URL="$(testing::connectivity::get_api_url "$SCENARIO_NAME" || true)"

if [ -z "$API_URL" ]; then
  testing::phase::add_error "Unable to discover API URL"
  testing::phase::end_with_summary "Performance tests incomplete"
fi

if ! command -v curl >/dev/null 2>&1; then
  testing::phase::add_error "curl is required for performance checks"
  testing::phase::end_with_summary "Performance tests incomplete"
fi

testing::phase::check "Health endpoint responds within 1s" \
  curl -s --max-time 1 -o /dev/null "${API_URL}/health"

testing::phase::check "Spin endpoint responds within 2s" \
  curl -s --max-time 2 -o /dev/null -X POST "${API_URL}/api/spin" \
    -H "Content-Type: application/json" \
    -d '{"wheel_id":"yes-or-no"}'

PERF_SCRIPT=$(cat <<'SH'
set -euo pipefail
for i in {1..10}; do
  curl -s --max-time 3 -o /dev/null -X POST "${API_URL}/api/spin" \
    -H "Content-Type: application/json" \
    -d '{"wheel_id":"dinner-decider"}' &
done
wait
SH
)

testing::phase::check "Handle 10 concurrent spins under 3s" \
  bash -c "$PERF_SCRIPT"

testing::phase::end_with_summary "Performance validation completed"
