#!/bin/bash
# Lightweight performance sanity checks for chore-tracking APIs.
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s" --require-runtime

cd "$TESTING_PHASE_SCENARIO_DIR"

API_BASE_URL=$(testing::connectivity::get_api_url "$TESTING_PHASE_SCENARIO_NAME")
if [ -z "$API_BASE_URL" ]; then
  testing::phase::add_error "Unable to resolve API base URL"
  testing::phase::end_with_summary "Performance checks incomplete"
fi

measure_endpoint() {
  local label="$1"
  local endpoint="$2"
  local threshold="$3"

  local elapsed
  elapsed=$(curl -w '%{time_total}' -o /dev/null -s "${API_BASE_URL}${endpoint}" || echo "10")

  if awk "BEGIN {exit !(${elapsed} < ${threshold})}"; then
    log::info "${label}: ${elapsed}s"
    testing::phase::add_test passed
  else
    testing::phase::add_error "${label} exceeded threshold (${elapsed}s >= ${threshold}s)"
    testing::phase::add_test failed
  fi
}

measure_endpoint "Health check latency" "/health" 1.0
measure_endpoint "Chore listing latency" "/api/chores" 2.0
measure_endpoint "User listing latency" "/api/users" 2.0

# Basic concurrency probe using health endpoint
concurrent_requests=8
start_time=$(date +%s.%N)
for _ in $(seq 1 $concurrent_requests); do
  curl -s "${API_BASE_URL}/health" >/dev/null &
done
wait
end_time=$(date +%s.%N)
total_time=$(awk -v start="$start_time" -v end="$end_time" 'BEGIN {printf "%.4f", end - start}')

if awk "BEGIN {exit !(${total_time} < 5.0)}"; then
  log::info "Concurrent health checks completed in ${total_time}s"
  testing::phase::add_test passed
else
  testing::phase::add_error "Concurrent health checks too slow (${total_time}s)"
  testing::phase::add_test failed
fi

testing::phase::end_with_summary "Performance checks completed"
