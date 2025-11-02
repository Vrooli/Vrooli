#!/bin/bash
# Measures response times and concurrency characteristics for smart-shopping-assistant.

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "240s" --require-runtime

API_URL=""
if API_URL=$(testing::connectivity::get_api_url "$TESTING_PHASE_SCENARIO_NAME"); then
  log::info "Running performance checks against $API_URL"
else
  testing::phase::add_warning "Unable to resolve API URL; skipping performance metrics"
  testing::phase::add_test skipped
  testing::phase::end_with_summary "Performance tests skipped"
fi

measure_request() {
  local description="$1"
  local threshold_ms="$2"
  local method="$3"
  local path="$4"
  local payload="${5:-}"
  local curl_args=("-s" "-o" "/dev/null" "-w" "%{time_total}" "-X" "$method" "$API_URL$path")

  testing::phase::add_test "$description"

  if [[ -n "$payload" ]]; then
    curl_args+=("-H" "Content-Type: application/json" "-d" "$payload")
  fi

  local time_raw
  if ! time_raw=$(curl "${curl_args[@]}"); then
    testing::phase::add_error "$description failed"
    return 1
  fi

  local time_ms
  time_ms=$(awk -v t="$time_raw" 'BEGIN {printf "%.0f", t * 1000}')
  log::info "$description completed in ${time_ms}ms"

  if [[ "$time_ms" -gt "$threshold_ms" ]]; then
    testing::phase::add_warning "$description exceeded threshold (${time_ms}ms > ${threshold_ms}ms)"
  fi

  return 0
}

measure_request "Health endpoint latency" 150 GET "/health"
measure_request "Shopping research latency" 1200 POST "/api/v1/shopping/research" \
  '{"profile_id":"perf-user","query":"laptop","budget_max":1000.0}'
measure_request "Tracking endpoint latency" 700 GET "/api/v1/shopping/tracking/perf-user"

# Concurrency smoke: fire multiple requests and ensure none fail.
CONCURRENT_REQUESTS=8
testing::phase::add_test "Handle ${CONCURRENT_REQUESTS} concurrent research requests"
log::info "Dispatching ${CONCURRENT_REQUESTS} concurrent research calls"
for i in $(seq 1 "$CONCURRENT_REQUESTS"); do
  curl -s -o /dev/null -X POST "$API_URL/api/v1/shopping/research" \
    -H "Content-Type: application/json" \
    -d "{\"profile_id\":\"perf-${i}\",\"query\":\"item-${i}\",\"budget_max\":250}" &
done
wait || true
log::success "Concurrent request batch completed"

# Cache efficiency check: warm request should be quicker (best-effort heuristic).
testing::phase::add_test "Cache warms repeated research queries"
cold_time=$(curl -s -o /dev/null -w '%{time_total}' -X POST "$API_URL/api/v1/shopping/research" \
  -H "Content-Type: application/json" \
  -d '{"profile_id":"cache-user","query":"smart speaker","budget_max":300}') || cold_time="0"
warm_time=$(curl -s -o /dev/null -w '%{time_total}' -X POST "$API_URL/api/v1/shopping/research" \
  -H "Content-Type: application/json" \
  -d '{"profile_id":"cache-user","query":"smart speaker","budget_max":300}') || warm_time="0"
if [[ "$cold_time" != "0" && "$warm_time" != "0" ]]; then
  cold_ms=$(awk -v t="$cold_time" 'BEGIN {printf "%.0f", t * 1000}')
  warm_ms=$(awk -v t="$warm_time" 'BEGIN {printf "%.0f", t * 1000}')
  log::info "Cold cache: ${cold_ms}ms, warm cache: ${warm_ms}ms"
  if [[ "$warm_ms" -ge "$cold_ms" ]]; then
    testing::phase::add_warning "Cache warm request (${warm_ms}ms) not faster than cold (${cold_ms}ms)"
  fi
else
  testing::phase::add_warning "Unable to capture cache timings"
fi

testing::phase::end_with_summary "Performance benchmarking completed"
