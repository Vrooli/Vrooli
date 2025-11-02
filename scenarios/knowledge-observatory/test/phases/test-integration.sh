#!/bin/bash
# Integration validation for knowledge-observatory
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
  testing::phase::add_error "Unable to discover API URL"
  testing::phase::end_with_summary "Missing API runtime"
fi

testing::phase::check "API health endpoint" curl -sf "$API_URL/health"

testing::phase::check "Search endpoint" curl -sf -X POST "$API_URL/api/v1/knowledge/search" \
  -H 'Content-Type: application/json' \
  -d '{"query":"integration smoke","limit":3}'

testing::phase::check "Graph endpoint" curl -sf "$API_URL/api/v1/knowledge/graph?max_nodes=10"

# CLI integration (prefer installed binary, fallback to local)
CLI_CMD="knowledge-observatory"
if ! command -v "$CLI_CMD" >/dev/null 2>&1; then
  CLI_CMD="./cli/knowledge-observatory"
fi
if [ -x "$CLI_CMD" ]; then
  testing::phase::check "CLI status command" bash -c "API_URL='$API_URL' $CLI_CMD status --json >/dev/null"
  testing::phase::check "CLI help command" bash -c "$CLI_CMD --help >/dev/null"
else
  testing::phase::add_warning "knowledge-observatory CLI not executable; skipping CLI checks"
fi

# UI reachability
if [ -n "$UI_URL" ]; then
  testing::phase::check "UI root accessible" curl -sf "$UI_URL/"
else
  testing::phase::add_warning "Unable to discover UI URL"
fi

# Resource sanity (best-effort)
if command -v vrooli >/dev/null 2>&1; then
  testing::phase::check "Qdrant health" bash -c "vrooli resource status qdrant --json >/dev/null"
  testing::phase::check "Postgres status" bash -c "vrooli resource status postgres --json >/dev/null"
else
  testing::phase::add_warning "vrooli CLI not available; resource status skipped"
fi


testing::phase::end_with_summary "Integration tests completed"
