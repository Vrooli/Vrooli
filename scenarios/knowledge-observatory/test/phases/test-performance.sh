#!/bin/bash
# Lightweight performance smoke checks for knowledge-observatory
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/connectivity.sh"

testing::phase::init --target-time "180s" --require-runtime

SCENARIO_NAME="$TESTING_PHASE_SCENARIO_NAME"
API_URL=$(testing::connectivity::get_api_url "$SCENARIO_NAME" || true)
UI_URL=$(testing::connectivity::get_ui_url "$SCENARIO_NAME" || true)

if [ -z "$API_URL" ]; then
  testing::phase::add_warning "API unavailable; skipping performance sampling"
  testing::phase::end_with_summary "Performance checks skipped"
fi

if ! command -v bc >/dev/null 2>&1; then
  testing::phase::add_warning "bc not available; skipping latency calculations"
  testing::phase::end_with_summary "Performance checks skipped"
fi

measure_latency() {
  local endpoint="$1"
  local iterations="$2"
  local total=0
  local success=0
  for _ in $(seq 1 "$iterations"); do
    local sample
    sample=$(curl -s -w "%{time_total}" -o /dev/null "$API_URL$endpoint" 2>/dev/null || echo "0")
    if [[ "$sample" != "0" ]]; then
      total=$(printf '%.3f' "$(echo "$total + $sample" | bc -l)")
      success=$((success + 1))
    fi
  done
  if [ "$success" -eq 0 ]; then
    echo "0"
  else
    printf '%.3f' "$(echo "$total / $success" | bc -l)"
  fi
}

health_avg=$(measure_latency "/health" 6)
if (( $(echo "$health_avg > 0" | bc -l) )); then
  testing::phase::add_test passed
  log::info "Health average latency: ${health_avg}s"
  if (( $(echo "$health_avg > 2.0" | bc -l) )); then
    testing::phase::add_warning "Health endpoint slower than 2s target"
  fi
else
  testing::phase::add_test failed
  testing::phase::add_error "Unable to capture health latency"
fi

search_avg=$(measure_latency "/api/v1/knowledge/search" 4)
if (( $(echo "$search_avg > 0" | bc -l) )); then
  testing::phase::add_test passed
  log::info "Search average latency: ${search_avg}s"
  if (( $(echo "$search_avg > 1.0" | bc -l) )); then
    testing::phase::add_warning "Search endpoint slower than 1s target"
  fi
fi

if [ -n "$UI_URL" ]; then
  ui_latency=$(curl -s -w "%{time_total}" -o /dev/null "$UI_URL/" 2>/dev/null || echo "0")
  if (( $(echo "$ui_latency > 0" | bc -l) )); then
    testing::phase::add_test passed
    log::info "UI load latency: ${ui_latency}s"
    if (( $(echo "$ui_latency > 3.0" | bc -l) )); then
      testing::phase::add_warning "UI load slower than 3s"
    fi
  fi
fi

# Memory usage sample (best-effort)
if pgrep -f "knowledge-observatory-api" >/dev/null && command -v ps >/dev/null 2>&1; then
  pid=$(pgrep -f "knowledge-observatory-api" | head -1)
  rss_mb=$(ps -o rss= -p "$pid" 2>/dev/null | awk '{printf "%.1f", $1/1024}')
  if [ -n "$rss_mb" ]; then
    testing::phase::add_test passed
    log::info "API RSS usage: ${rss_mb}MB"
    if (( $(echo "$rss_mb > 512" | bc -l) )); then
      testing::phase::add_warning "API memory usage exceeds 512MB"
    fi
  fi
else
  testing::phase::add_warning "Unable to sample process memory"
fi


testing::phase::end_with_summary "Performance sampling completed"
