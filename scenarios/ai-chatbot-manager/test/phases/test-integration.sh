#!/bin/bash
# Run API and WebSocket integration checks against a live scenario instance

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "300s" --require-runtime

SCENARIO_NAME="$TESTING_PHASE_SCENARIO_NAME"
API_URL=""

if API_URL=$(testing::connectivity::get_api_url "$SCENARIO_NAME"); then
  :
else
  testing::phase::add_warning "Unable to resolve API URL for ${SCENARIO_NAME}; HTTP integration checks will be skipped"
fi

testing::phase::check "API health endpoint responds" testing::connectivity::test_api "$SCENARIO_NAME"

run_script_if_present() {
  local description=$1
  local script_path=$2
  local env_var_name=${3:-}
  local env_var_value=${4:-}

  if [ ! -f "$script_path" ]; then
    testing::phase::add_warning "Integration helper missing: ${script_path}"
    testing::phase::add_test skipped
    return 0
  fi

  if [ -n "$env_var_name" ]; then
    testing::phase::check "$description" env "$env_var_name=$env_var_value" bash "$script_path"
  else
    testing::phase::check "$description" bash "$script_path"
  fi
}

if [ -n "$API_URL" ]; then
  run_script_if_present "REST API smoke suite" "$TESTING_PHASE_SCENARIO_DIR/test/test-api-endpoints-improved.sh" API_BASE_URL "$API_URL"
else
  testing::phase::add_warning "Skipping REST API smoke suite without resolved API URL"
  testing::phase::add_test skipped
fi

if [ -n "$API_URL" ]; then
  run_script_if_present "WebSocket integration suite" "$TESTING_PHASE_SCENARIO_DIR/test/test-websocket-integration.sh" API_BASE_URL "$API_URL"
else
  testing::phase::add_warning "Skipping WebSocket integration without resolved API URL"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Integration validation completed"
