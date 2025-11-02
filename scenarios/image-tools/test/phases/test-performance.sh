#!/bin/bash
# Performance phase – ensures key operations stay within latency expectations.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "90s" --require-runtime

if ! command -v curl >/dev/null 2>&1; then
  testing::phase::add_error "curl is required for performance checks"
  testing::phase::end_with_summary "Performance benchmarks skipped"
fi

API_BASE=$(testing::connectivity::get_api_url "$TESTING_PHASE_SCENARIO_NAME" || true)
if [ -z "$API_BASE" ]; then
  testing::phase::add_error "Unable to resolve API endpoint for $TESTING_PHASE_SCENARIO_NAME"
  testing::phase::end_with_summary "Performance benchmarks skipped"
fi

TMP_IMAGE=$(mktemp -t image-tools-perf-XXXXXX.jpg)
trap 'rm -f "$TMP_IMAGE" 2>/dev/null || true' EXIT

echo "/9j/4AAQSkZJRgABAQEAYABgAAD/2wBDAAgGBgcGBQgHBwcJCQgKDBQNDAsLDBkSEw8UHRofHh0aHBwgJC4nICIsIxwcKDcpLDAxNDQ0Hyc5PTgyPC4zNDL/2wBDAQkJCQwLDBgNDRgyIRwhMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjL/wAARCAABAAEDASIAAhEBAxEB/8QAFQABAQAAAAAAAAAAAAAAAAAAAAr/xAAUEAEAAAAAAAAAAAAAAAAAAAAA/8QAFQEBAQAAAAAAAAAAAAAAAAAAAAX/xAAUEQEAAAAAAAAAAAAAAAAAAAAA/9oADAMBAAIRAxEAPwCwAA8A/9k=" | base64 -d > "$TMP_IMAGE"

benchmark_endpoint() {
  local description="$1"
  local endpoint="$2"
  shift 2
  local result
  result=$(curl -s -o /dev/null -w "%{http_code}:%{time_total}" "$API_BASE$endpoint" "$@")
  local status_code="${result%%:*}"
  local seconds="${result##*:}"
  # Convert to milliseconds (rounded)
  local ms
  ms=$(awk -v val="${seconds}" 'BEGIN { printf("%d", val * 1000 + 0.5) }')
  echo "   ${description}: ${ms}ms (status ${status_code})"
  if [ "$status_code" = "200" ] && [ "$ms" -le 2000 ]; then
    return 0
  fi
  return 1
}

if benchmark_endpoint "Health endpoint latency" "/health"; then
  testing::phase::add_test passed
else
  testing::phase::add_error "Health endpoint exceeded latency budget"
  testing::phase::add_test failed
fi

if benchmark_endpoint "Compression latency" "/api/v1/image/compress" -F "image=@$TMP_IMAGE" -F "quality=80"; then
  testing::phase::add_test passed
else
  testing::phase::add_error "Compression request too slow"
  testing::phase::add_test failed
fi

if benchmark_endpoint "Resize latency" "/api/v1/image/resize" -F "image=@$TMP_IMAGE" -F "width=80" -F "height=80" -F "maintain_aspect=true"; then
  testing::phase::add_test passed
else
  testing::phase::add_error "Resize request too slow"
  testing::phase::add_test failed
fi

MEASURE_PID=$(pgrep -f "image-tools-api" | head -1)
if [ -n "$MEASURE_PID" ]; then
  RSS_KB=$(ps -o rss= -p "$MEASURE_PID" 2>/dev/null || echo 0)
  RSS_MB=$((RSS_KB / 1024))
  echo "   API memory footprint: ${RSS_MB}MB"
  if [ "$RSS_MB" -le 2048 ]; then
    testing::phase::add_test passed
  else
    testing::phase::add_warning "API process using ${RSS_MB}MB (>2GB)"
    testing::phase::add_test failed
  fi
else
  testing::phase::add_warning "API process not detected; skipping memory observation"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Performance checks completed"
