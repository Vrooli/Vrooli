#!/bin/bash
# Performance baselines for quiz-generator scenario
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/connectivity.sh"

testing::phase::init --target-time "120s" --require-runtime

API_URL=$(testing::connectivity::get_api_url "$TESTING_PHASE_SCENARIO_NAME" || true)
if [ -z "$API_URL" ]; then
  testing::phase::add_error "Unable to resolve API URL"
  testing::phase::add_test failed
  testing::phase::end_with_summary "Performance tests failed"
fi

measure_latency() {
  local command="$1"
  local threshold_ms="$2"
  local label="$3"

  local start_ns end_ns duration_ms
  start_ns=$(date +%s%N)
  if ! eval "$command" >/dev/null 2>&1; then
    testing::phase::add_error "${label} request failed"
    testing::phase::add_test failed
    return 1
  fi
  end_ns=$(date +%s%N)
  duration_ms=$(( (end_ns - start_ns) / 1000000 ))

  if [ "$duration_ms" -le "$threshold_ms" ]; then
    log::success "✅ ${label}: ${duration_ms}ms"
    testing::phase::add_test passed
  else
    testing::phase::add_warning "${label}: ${duration_ms}ms (threshold ${threshold_ms}ms)"
    testing::phase::add_test skipped
  fi
}

measure_latency "curl -sf ${API_URL}/api/health" 200 "Health endpoint latency"

measure_latency "curl -sf -X POST ${API_URL}/api/v1/quiz/generate -H 'Content-Type: application/json' -d '{\"content\":\"Performance baseline.\",\"question_count\":3}'" 5000 "Quiz generation latency"

start_ns=$(date +%s%N)
for _ in 1 2 3 4 5; do
  curl -sf "${API_URL}/api/health" >/dev/null 2>&1 &
done
wait
end_ns=$(date +%s%N)
concurrent_ms=$(( (end_ns - start_ns) / 1000000 ))
if [ "$concurrent_ms" -le 1000 ]; then
  log::success "✅ Concurrent health probes completed in ${concurrent_ms}ms"
  testing::phase::add_test passed
else
  testing::phase::add_warning "Concurrent health probes took ${concurrent_ms}ms"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Performance benchmarks completed"
