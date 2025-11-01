#!/usr/bin/env bash
# Exercises live API/UI endpoints and end-to-end security workflows.
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/connectivity.sh"

testing::phase::init --target-time "300s" --require-runtime

scenario_name="$TESTING_PHASE_SCENARIO_NAME"

API_URL=$(testing::connectivity::get_api_url "$scenario_name" 2>/dev/null || true)
UI_URL=$(testing::connectivity::get_ui_url "$scenario_name" 2>/dev/null || true)

if [ -z "$API_URL" ]; then
  testing::phase::add_error "Unable to discover API URL via vrooli"
  testing::phase::end_with_summary "Integration tests incomplete"
fi

if [ -z "$UI_URL" ]; then
  testing::phase::add_error "Unable to discover UI URL via vrooli"
  testing::phase::end_with_summary "Integration tests incomplete"
fi

API_PORT="${API_URL##*:}"
UI_PORT="${UI_URL##*:}"

testing::phase::check "API health endpoint" bash -c 'curl -sf "$1" | jq -e ".status == \"healthy\"" >/dev/null' _ "${API_URL}/health"
testing::phase::check "UI root serves content" bash -c 'curl -sf "$1" | grep -qi "Prompt Injection Arena"' _ "${UI_URL}/"
testing::phase::check "UI health endpoint" bash -c 'curl -sf "$1" | jq -e ".api_connectivity.connected == true" >/dev/null' _ "${UI_URL}/health"

testing::phase::check "Injection library endpoint" bash -c 'curl -sf "$1" | jq -e ".techniques" >/dev/null' _ "${API_URL}/api/v1/injections/library"
testing::phase::check "Agent leaderboard endpoint" bash -c 'curl -sf "$1" | jq -e ".leaderboard" >/dev/null' _ "${API_URL}/api/v1/leaderboards/agents"
testing::phase::check "Export formats endpoint" bash -c 'curl -sf "$1" | jq -e ".formats" >/dev/null' _ "${API_URL}/api/v1/export/formats"

if vrooli resource status qdrant >/dev/null 2>&1; then
  testing::phase::check "Vector similarity search" bash -c 'curl -sf "$1" | jq -e "type==\"array\"" >/dev/null' _ "${API_URL}/api/v1/injections/similar?query=override"
else
  testing::phase::add_warning "Qdrant unavailable; skipping similarity search validation"
  testing::phase::add_test skipped
fi

testing::phase::check "Agent security integration script" bash -c 'API_PORT="$1" UI_PORT="$2" bash "$3"' _ "$API_PORT" "$UI_PORT" "$TESTING_PHASE_SCENARIO_DIR/test/test-agent-security.sh"

testing::phase::end_with_summary "Integration validation completed"
