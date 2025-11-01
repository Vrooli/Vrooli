#!/bin/bash
# Performance testing phase for symbol-search scenario

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/connectivity.sh"

testing::phase::init --target-time "120s" --require-runtime

if ! command -v bc >/dev/null 2>&1; then
  testing::phase::add_warning "bc not available; skipping performance benchmarking"
  testing::phase::add_test skipped
  testing::phase::end_with_summary "Performance checks skipped"
fi

SCENARIO_NAME="${TESTING_PHASE_SCENARIO_NAME}"
API_URL=$(testing::connectivity::get_api_url "$SCENARIO_NAME" || true)

if [ -z "$API_URL" ]; then
  testing::phase::add_error "Unable to discover API URL for $SCENARIO_NAME"
  testing::phase::end_with_summary "Performance checks incomplete"
fi

measure_response_time() {
  local endpoint="$1"
  local method="${2:-GET}"
  local data="${3:-}"
  local iterations=8
  local total_ms=0

  for _ in $(seq 1 $iterations); do
    local elapsed
    if [ "$method" = "GET" ]; then
      elapsed=$(curl -s -w "%{time_total}" -o /dev/null "${API_URL}${endpoint}")
    else
      elapsed=$(curl -s -X POST -w "%{time_total}" -o /dev/null \
        -H 'Content-Type: application/json' \
        -d "$data" \
        "${API_URL}${endpoint}")
    fi
    total_ms=$(echo "$total_ms + ($elapsed * 1000)" | bc -l)
  done

  echo "scale=2; $total_ms / $iterations" | bc -l
}

check_threshold() {
  local description="$1"
  local endpoint="$2"
  local target_ms="$3"
  local method="${4:-GET}"
  local payload="${5:-}"

  local avg_ms
  avg_ms=$(measure_response_time "$endpoint" "$method" "$payload")

  printf "📊 %s average latency: %sms\n" "$description" "$avg_ms"

  if (( $(echo "$avg_ms <= $target_ms" | bc -l) )); then
    testing::phase::add_test passed
    return 0
  fi

  testing::phase::add_error "$description latency ${avg_ms}ms exceeds ${target_ms}ms target"
  testing::phase::add_test failed
  return 1
}

testing::phase::check "API reachable for performance run" curl -sf "${API_URL}/health"

check_threshold "Search endpoint" "/api/search?q=LATIN&limit=100" 50
check_threshold "Character detail endpoint" "/api/character/U+0041" 25
check_threshold "Categories endpoint" "/api/categories" 40
check_threshold "Blocks endpoint" "/api/blocks" 40
check_threshold "Bulk range endpoint" "/api/bulk/range" 200 POST '{"ranges":[{"start":"U+0041","end":"U+005A"}]}'

testing::phase::end_with_summary "Performance tests completed"
