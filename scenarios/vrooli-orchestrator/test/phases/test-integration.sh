#!/bin/bash
# Runs API/UI/CLI integration checks against a managed scenario runtime.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "240s" --require-runtime

SCENARIO_NAME="${TESTING_PHASE_SCENARIO_NAME}"
TEST_PROFILE_ID="automation-smoke-${RANDOM}"

cleanup_profile() {
  local api_url
  api_url=$(testing::connectivity::get_api_url "$SCENARIO_NAME" 2>/dev/null || echo "")
  if [ -n "$api_url" ]; then
    curl -sS -X DELETE "${api_url}/api/v1/profiles/${TEST_PROFILE_ID}" >/dev/null 2>&1 || true
  fi
}

testing::phase::register_cleanup cleanup_profile

API_URL=$(testing::connectivity::get_api_url "$SCENARIO_NAME" 2>/dev/null || true)
UI_URL=$(testing::connectivity::get_ui_url "$SCENARIO_NAME" 2>/dev/null || true)

if [ -z "$API_URL" ]; then
  testing::phase::add_error "Unable to resolve API URL for $SCENARIO_NAME"
  testing::phase::end_with_summary "Integration tests incomplete"
fi

if [ -z "$UI_URL" ]; then
  testing::phase::add_warning "UI port not discovered; UI connectivity checks will be skipped"
fi

run_api_health() {
  curl -sSf "${API_URL}/health" >/dev/null
}

run_profiles_list() {
  curl -sSf "${API_URL}/api/v1/profiles" >/dev/null
}

run_profile_creation() {
  curl -sSf \
    -X POST "${API_URL}/api/v1/profiles" \
    -H 'Content-Type: application/json' \
    -d "{\"name\":\"${TEST_PROFILE_ID}\",\"display_name\":\"Automation Smoke Profile\",\"description\":\"Created during integration testing\",\"resources\":[\"postgres\"],\"scenarios\":[]}" \
    >/dev/null
}

run_profile_retrieval() {
  curl -sSf "${API_URL}/api/v1/profiles/${TEST_PROFILE_ID}" >/dev/null
}

run_ui_health() {
  [ -n "$UI_URL" ] && curl -sSf "${UI_URL}/health" >/dev/null
}

run_cli_list() {
  bash -c 'cd cli && ./vrooli-orchestrator list-profiles --json' >/dev/null
}

run_cli_help() {
  bash -c 'cd cli && ./vrooli-orchestrator help' >/dev/null
}

testing::phase::check "API health endpoint responds" run_api_health

testing::phase::check "Profiles endpoint lists profiles" run_profiles_list

testing::phase::check "Profile creation via API succeeds" run_profile_creation

testing::phase::check "Created profile retrievable" run_profile_retrieval

if [ -n "$UI_URL" ]; then
  testing::phase::check "UI health endpoint responds" run_ui_health
else
  testing::phase::add_test skipped
fi

if [ -x "cli/vrooli-orchestrator" ]; then
  testing::phase::check "CLI list-profiles command succeeds" run_cli_list
  testing::phase::check "CLI help command succeeds" run_cli_help
else
  testing::phase::add_warning "CLI executable missing; skipping CLI smoke checks"
  testing::phase::add_test skipped
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Integration checks completed"
