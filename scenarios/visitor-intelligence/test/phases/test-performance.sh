#!/bin/bash
# Performance smoke tests for visitor-intelligence

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s" --require-runtime

SCENARIO_NAME="${TESTING_PHASE_SCENARIO_NAME}"
if ! command -v vrooli >/dev/null 2>&1; then
  testing::phase::add_warning "vrooli CLI unavailable; skipping runtime-dependent checks"
  testing::phase::add_test skipped
  testing::phase::end_with_summary "Performance phase skipped"
fi

API_PORT_OUTPUT=$(vrooli scenario port "$SCENARIO_NAME" API_PORT 2>/dev/null || true)
API_PORT=$(echo "$API_PORT_OUTPUT" | awk -F= '/=/{print $2}' | tr -d ' ')
if [ -z "$API_PORT" ]; then
  API_PORT=$(echo "$API_PORT_OUTPUT" | tr -d '[:space:]')
fi

API_BASE="http://localhost:${API_PORT}/api/v1"

latency_payload='{"fingerprint":"perf-phase","event_type":"pageview","scenario":"performance-phase","page_url":"https://example.com/perf"}'
duration=$(curl -s -o /tmp/vi-perf-response-$$.json -w '%{time_total}' -X POST \
  -H 'Content-Type: application/json' \
  -d "$latency_payload" \
  "${API_BASE}/visitor/track" || echo "0")

if [ -s "/tmp/vi-perf-response-$$.json" ]; then
  rm -f "/tmp/vi-perf-response-$$.json" 2>/dev/null || true
fi

if awk "BEGIN {exit !($duration > 0)}"; then
  log::info "Tracking endpoint latency: ${duration}s"
  if ! awk "BEGIN {exit !($duration <= 0.20)}"; then
    testing::phase::add_warning "Tracking latency above 200ms threshold"
  fi
  testing::phase::add_test passed
else
  testing::phase::add_error "Tracking endpoint latency probe failed"
  testing::phase::add_test failed
fi

if command -v go >/dev/null 2>&1 && [ -f "api/go.mod" ]; then
  if testing::phase::check "Run Go benchmarks" bash -c 'cd api && go test -bench=. -benchmem -run=^$ -timeout=2m'; then
    true
  fi
else
  testing::phase::add_warning "Go toolchain unavailable; skipping API benchmark"
  testing::phase::add_test skipped
fi

teardown_payload='{"fingerprint":"perf-phase","event_type":"cleanup","scenario":"performance-phase","page_url":"https://example.com/perf"}'
curl -sSf -X POST -H "Content-Type: application/json" -d "$teardown_payload" "${API_BASE}/visitor/track" >/dev/null 2>&1 || true

testing::phase::end_with_summary "Performance checks completed"
