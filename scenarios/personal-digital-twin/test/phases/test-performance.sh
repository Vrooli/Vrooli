#!/bin/bash
# Benchmarks API responsiveness and throughput targets.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/connectivity.sh"

testing::phase::init --target-time "240s" --require-runtime

missing_tools=()
for tool in curl jq awk; do
  if ! command -v "$tool" >/dev/null 2>&1; then
    missing_tools+=("$tool")
  fi
done

if [ ${#missing_tools[@]} -gt 0 ]; then
  testing::phase::add_error "Required tooling missing: ${missing_tools[*]}"
  testing::phase::end_with_summary "Performance checks blocked"
fi

SCENARIO_NAME="$TESTING_PHASE_SCENARIO_NAME"
API_BASE_URL=$(testing::connectivity::get_api_url "$SCENARIO_NAME" || true)
if [ -z "$API_BASE_URL" ]; then
  testing::phase::add_error "Unable to resolve API URL for $SCENARIO_NAME"
  testing::phase::end_with_summary "Performance checks blocked"
fi

CHAT_PORT_OUTPUT=$(vrooli scenario port "$SCENARIO_NAME" CHAT_PORT 2>/dev/null || true)
CHAT_PORT=$(echo "$CHAT_PORT_OUTPUT" | awk -F= '/=/{print $2}' | tr -d ' ')
if [ -z "$CHAT_PORT" ]; then
  CHAT_BASE_URL="$API_BASE_URL"
else
  CHAT_BASE_URL="http://localhost:${CHAT_PORT}"
fi

PERSONA_ID=""

create_test_persona() {
  local response
  response=$(curl -s -X POST "$API_BASE_URL/api/persona/create" \
    -H "Content-Type: application/json" \
    -d '{"name": "Performance Persona", "description": "Performance test persona"}') || return 1
  PERSONA_ID=$(echo "$response" | jq -r '.id // empty')
  [ -n "$PERSONA_ID" ]
}

compare_threshold() {
  local value="$1"
  local threshold="$2"
  awk "BEGIN {exit !($value < $threshold)}"
}

latency_check() {
  local label="$1"
  local url="$2"
  local threshold="$3"
  local method="${4:-GET}"
  local payload="${5:-}"
  local response_time

  if [ "$method" = "POST" ]; then
    response_time=$(curl -s -w "%{time_total}" -o /dev/null -X POST "$url" -H "Content-Type: application/json" -d "$payload") || return 1
  else
    response_time=$(curl -s -w "%{time_total}" -o /dev/null "$url") || return 1
  fi

  log::info "${label} response time: ${response_time}s (limit ${threshold}s)"
  compare_threshold "$response_time" "$threshold"
}

chat_latency_check() {
  local payload
  payload=$(jq -cn --arg persona "$PERSONA_ID" '{persona_id: $persona, message: "Performance test message"}')
  latency_check "Chat message" "$CHAT_BASE_URL/api/chat" 2.5 "POST" "$payload"
}

concurrency_check() {
  local iterations=10
  local start_time end_time total
  start_time=$(date +%s.%N)
  seq 1 "$iterations" | xargs -I{} -P "$iterations" curl -s "$API_BASE_URL/api/persona/${PERSONA_ID}" >/dev/null || return 1
  end_time=$(date +%s.%N)
  total=$(awk -v start="$start_time" -v finish="$end_time" 'BEGIN {printf "%.4f", (finish-start)}')
  log::info "Concurrent persona fetch (x${iterations}) completed in ${total}s (limit 5s)"
  compare_threshold "$total" 5
}

record_performance() {
  local description="$1"
  shift
  if "$@"; then
    testing::phase::add_test passed
    log::success "✅ $description"
  else
    testing::phase::add_test failed
    log::error "❌ $description"
  fi
}

record_performance "Create performance persona" create_test_persona

record_performance "Health endpoint latency <0.1s" latency_check "Health endpoint" "$API_BASE_URL/health" 0.1
record_performance "Persona retrieval latency <0.5s" latency_check "Persona retrieval" "$API_BASE_URL/api/persona/${PERSONA_ID}" 0.5
record_performance "Persona list latency <1.0s" latency_check "Persona list" "$API_BASE_URL/api/personas" 1.0
search_payload=$(jq -cn --arg persona "$PERSONA_ID" --arg query "performance" '{persona_id: $persona, query: $query, limit: 10}')
token_payload=$(jq -cn --arg persona "$PERSONA_ID" --arg name "perf-token" '{persona_id: $persona, name: $name}')

record_performance "Search latency <1.0s" latency_check "Search" "$API_BASE_URL/api/search" 1.0 "POST" "$search_payload"
record_performance "Token creation latency <0.5s" latency_check "Token creation" "$API_BASE_URL/api/tokens/create" 0.5 "POST" "$token_payload"
record_performance "Chat latency <2.5s" chat_latency_check
record_performance "Concurrent persona fetch <5s" concurrency_check

testing::phase::end_with_summary "Performance benchmarks completed"
