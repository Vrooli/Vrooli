#!/bin/bash
# Lightweight performance smoke checks for Contact Book.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s" --require-runtime

scenario_name="$TESTING_PHASE_SCENARIO_NAME"
API_URL=""
if command -v vrooli >/dev/null 2>&1; then
  API_URL=$(testing::connectivity::get_api_url "$scenario_name" || true)
fi

if [ -z "$API_URL" ]; then
  testing::phase::add_error "Unable to determine API URL (vrooli CLI required)"
  testing::phase::end_with_summary "Performance checks incomplete"
fi

measure_endpoint() {
  local description="$1"
  local threshold_ms="$2"
  shift 2

  local timing
  timing=$(curl -sf -o /dev/null -w '%{time_total}' "$@" || echo "-1")
  if [ "$timing" = "-1" ] || [ -z "$timing" ]; then
    testing::phase::add_error "$description request failed"
    testing::phase::add_test failed
    return
  fi
  local elapsed_ms
  elapsed_ms=$(awk -v t="$timing" 'BEGIN{printf "%.0f", t*1000}')
  if [ "$elapsed_ms" -le "$threshold_ms" ]; then
    log::success "${description}: ${elapsed_ms}ms"
    testing::phase::add_test passed
  else
    testing::phase::add_warning "${description} slower than target (${elapsed_ms}ms > ${threshold_ms}ms)"
    testing::phase::add_test skipped
  fi
}

measure_endpoint "Health endpoint" 200 "${API_URL}/health"
measure_endpoint "Contacts endpoint" 200 "${API_URL}/api/v1/contacts?limit=50"
measure_endpoint "Search endpoint" 500 -X POST "${API_URL}/api/v1/search" -H 'Content-Type: application/json' -d '{"query":"performance","limit":10}'

if command -v contact-book >/dev/null 2>&1; then
  start=$(date +%s%N)
  if contact-book list --limit 20 >/dev/null 2>&1; then
    end=$(date +%s%N)
    cli_ms=$(( (end - start) / 1000000 ))
    if [ "$cli_ms" -le 2000 ]; then
      log::success "contact-book list latency ${cli_ms}ms"
      testing::phase::add_test passed
    else
      testing::phase::add_warning "contact-book list slower than target (${cli_ms}ms > 2000ms)"
      testing::phase::add_test skipped
    fi
  else
    testing::phase::add_error "contact-book list command failed"
    testing::phase::add_test failed
  fi
else
  testing::phase::add_warning "contact-book CLI not in PATH; skipping CLI performance"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Performance sampling completed"
