#!/usr/bin/env bash
# Performance smoke tests for SmartNotes API
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s" --require-runtime

scenario_name="$TESTING_PHASE_SCENARIO_NAME"
API_URL=$(testing::connectivity::get_api_url "$scenario_name" || echo "")

if [ -z "$API_URL" ]; then
  testing::phase::add_error "Unable to resolve API URL"
  testing::phase::end_with_summary "Performance tests incomplete"
fi

NOTES_CREATED=()

cleanup_perf_notes() {
  for note in "${NOTES_CREATED[@]}"; do
    curl -sf -X DELETE "${API_URL}/api/notes/${note}" >/dev/null 2>&1 || true
  done
}

testing::phase::register_cleanup cleanup_perf_notes

measure() {
  local method=$1
  local endpoint=$2
  local label=$3
  local threshold=$4
  local payload=${5:-}

  local start end duration response
  start=$(date +%s%N)
  if [ -n "$payload" ]; then
    if ! response=$(curl -sf -X "$method" "${API_URL}${endpoint}" -H "Content-Type: application/json" -d "$payload"); then
      return 2
    fi
  else
    if ! response=$(curl -sf -X "$method" "${API_URL}${endpoint}"); then
      return 2
    fi
  fi
  end=$(date +%s%N)
  duration=$(( (end - start) / 1000000 ))

  if [[ "$method" == "POST" && "$endpoint" == "/api/notes" ]]; then
    local note_id
    note_id=$(echo "$response" | jq -r '.id // empty')
    if [ -n "$note_id" ]; then
      NOTES_CREATED+=("$note_id")
    fi
  fi

  printf '%s' "$duration"
  if [ "$duration" -le "$threshold" ]; then
    return 0
  elif [ "$duration" -le $((threshold * 2)) ]; then
    return 1
  else
    return 3
  fi
}

run_measurement() {
  local method=$1
  local endpoint=$2
  local label=$3
  local threshold=$4
  local payload=${5:-}

  local duration
  if duration=$(measure "$method" "$endpoint" "$label" "$threshold" "$payload"); then
    log::success "✅ ${label}: ${duration}ms (target ${threshold}ms)"
    testing::phase::add_test passed
  else
    local status=$?
    case $status in
      1)
        log::warning "⚠️  ${label}: ${duration}ms exceeds target ${threshold}ms"
        testing::phase::add_warning "${label} slower than target"
        testing::phase::add_test passed
        ;;
      2)
        log::error "❌ ${label}: request failed"
        testing::phase::add_test failed
        ;;
      *)
        log::error "❌ ${label}: ${duration:-unknown}ms (target ${threshold}ms)"
        testing::phase::add_test failed
        ;;
    esac
  fi
}

run_measurement GET "/health" "Health endpoint latency" 500
run_measurement GET "/api/notes" "List notes latency" 700
run_measurement GET "/api/tags" "List tags latency" 700
run_measurement POST "/api/notes" "Create note latency" 900 '{"title":"Perf Note","content":"Performance test payload","content_type":"markdown"}'
run_measurement POST "/api/search" "Search latency" 900 '{"query":"performance","limit":5}'

# Lightweight concurrency probe
CONCURRENT=8
start=$(date +%s)
for _ in $(seq 1 "$CONCURRENT"); do
  curl -sf "${API_URL}/health" >/dev/null 2>&1 &
done
wait
elapsed=$(( $(date +%s) - start ))
if [ "$elapsed" -le 3 ]; then
  log::success "✅ ${CONCURRENT} concurrent health requests in ${elapsed}s"
  testing::phase::add_test passed
else
  log::warning "⚠️  Concurrent request handling took ${elapsed}s"
  testing::phase::add_warning "Concurrent health checks slower than expected"
  testing::phase::add_test passed
fi

testing::phase::end_with_summary "Performance checks completed"
