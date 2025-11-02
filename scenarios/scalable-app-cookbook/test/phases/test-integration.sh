#!/bin/bash
# Validates core API endpoints and data integrity expectations.
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "240s" --require-runtime

scenario_name="$TESTING_PHASE_SCENARIO_NAME"
api_base=""

if ! api_base=$(testing::connectivity::get_api_url "$scenario_name"); then
  testing::phase::add_error "Unable to discover API port for ${scenario_name}"
  testing::phase::end_with_summary "Integration tests incomplete"
fi

has_jq=false
if command -v jq >/dev/null 2>&1; then
  has_jq=true
else
  testing::phase::add_warning "jq not available; JSON assertions downgraded"
fi

if $has_jq; then
  testing::phase::check "API health reports healthy" bash -c \
    "curl -sf --retry 3 --retry-delay 2 '${api_base}/health' | jq -e '.status == \"healthy\" and .database == true' >/dev/null"
else
  testing::phase::check "API health endpoint reachable" bash -c \
    "curl -sf --retry 3 --retry-delay 2 '${api_base}/health' >/dev/null"
fi

if $has_jq; then
  testing::phase::check "Pattern search returns results" bash -c \
    "curl -sf --retry 3 --retry-delay 2 '${api_base}/api/v1/patterns/search?chapter=Part%20A' | jq -e '.total >= 5' >/dev/null"
else
  testing::phase::check "Pattern search endpoint reachable" bash -c \
    "curl -sf --retry 3 --retry-delay 2 '${api_base}/api/v1/patterns/search?chapter=Part%20A' >/dev/null"
fi

if $has_jq; then
  testing::phase::check "JWT query provides pattern data" bash -c \
    "curl -sf --retry 3 --retry-delay 2 '${api_base}/api/v1/patterns/search?query=jwt' | jq -e '.patterns | length > 0' >/dev/null"
else
  testing::phase::check "JWT query endpoint reachable" bash -c \
    "curl -sf --retry 3 --retry-delay 2 '${api_base}/api/v1/patterns/search?query=jwt' >/dev/null"
fi

testing::phase::check "Chapters endpoint responds" bash -c \
  "curl -sf --retry 3 --retry-delay 2 '${api_base}/api/v1/patterns/chapters' >/dev/null"

if $has_jq; then
  testing::phase::check "Stats endpoint exposes totals" bash -c \
    "curl -sf --retry 3 --retry-delay 2 '${api_base}/api/v1/patterns/stats' | jq -e '.statistics.total_patterns >= 8' >/dev/null"
else
  testing::phase::check "Stats endpoint reachable" bash -c \
    "curl -sf --retry 3 --retry-delay 2 '${api_base}/api/v1/patterns/stats' >/dev/null"
fi

testing::phase::end_with_summary "Integration validation completed"
