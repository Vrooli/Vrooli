#!/bin/bash
# Validate higher-level business workflows and error handling semantics
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s" --require-runtime

SCENARIO_NAME="$TESTING_PHASE_SCENARIO_NAME"
API_BASE_URL=$(testing::connectivity::get_api_url "$SCENARIO_NAME" || true)
JQ_AVAILABLE=false

if [ -z "$API_BASE_URL" ]; then
  testing::phase::add_error "Unable to discover API URL for $SCENARIO_NAME"
  testing::phase::end_with_summary "Business validation incomplete"
fi

if command -v jq >/dev/null 2>&1; then
  JQ_AVAILABLE=true
else
  testing::phase::add_warning "jq not available; response payload assertions will be limited"
fi

# POST scan endpoint should succeed and return a helpful message
testing::phase::check "Agent scan endpoint acknowledges request" env API_BASE_URL="$API_BASE_URL" bash -c 'curl -fsS -X POST "$API_BASE_URL/api/v1/agents/scan" | grep -q "Codex agent manager uses live run tracking"'

# Status endpoint should report counters (even zero) with success flag if jq present
if [ "$JQ_AVAILABLE" = true ]; then
  testing::phase::check "Status endpoint returns counters" env API_BASE_URL="$API_BASE_URL" bash -c 'curl -fsS "$API_BASE_URL/api/v1/agents/status" | jq -e ".success == true and (.data.total >= 0)"'
else
  testing::phase::add_test skipped
fi

# Capability search should reject missing capability payloads
testing::phase::check "Search endpoint validates missing capability" env API_BASE_URL="$API_BASE_URL" bash -c '
  tmp=$(mktemp)
  status=$(curl -s -o "$tmp" -w "%{http_code}" "$API_BASE_URL/api/v1/agents/search")
  rm -f "$tmp"
  [ "$status" = "400" ]
'

# Individual agent lookup for unknown id should yield 404
testing::phase::check "Unknown agent lookup returns 404" env API_BASE_URL="$API_BASE_URL" bash -c '
  tmp=$(mktemp)
  status=$(curl -s -o "$tmp" -w "%{http_code}" "$API_BASE_URL/api/v1/agents/unknown-id")
  rm -f "$tmp"
  [ "$status" = "404" ]
'

# CLI smoke test when available
if command -v agent-dashboard >/dev/null 2>&1; then
  testing::phase::check "agent-dashboard CLI help executes" agent-dashboard --help
else
  testing::phase::add_warning "agent-dashboard CLI not on PATH; skipping CLI smoke test"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Business validation completed"
