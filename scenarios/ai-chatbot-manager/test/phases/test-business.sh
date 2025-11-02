#!/bin/bash
# Validate higher-level business workflows (CLI, widget, premium feature flows)

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "240s" --require-runtime

SCENARIO_NAME="$TESTING_PHASE_SCENARIO_NAME"
API_URL=""
API_PORT_VALUE=""

if API_URL=$(testing::connectivity::get_api_url "$SCENARIO_NAME"); then
  API_PORT_VALUE="${API_URL##*:}"
else
  testing::phase::add_warning "Unable to resolve API URL for ${SCENARIO_NAME}; API-dependent business checks may be skipped"
fi

run_script_with_env() {
  local description=$1
  local script_path=$2
  local env_args=()
  shift 2

  if [ ! -f "$script_path" ]; then
    testing::phase::add_warning "Business helper missing: ${script_path}"
    testing::phase::add_test skipped
    return 0
  fi

  env_args=("env")
  while [ $# -gt 0 ]; do
    env_args+=("$1")
    shift
  done
  env_args+=("bash" "$script_path")

  testing::phase::check "$description" "${env_args[@]}"
}

# CLI tests (BATS) if present
if [ -f "$TESTING_PHASE_SCENARIO_DIR/cli/ai-chatbot-manager.bats" ]; then
  testing::phase::check "CLI BATS suite" bash -c 'cd cli && bats ai-chatbot-manager.bats'
else
  testing::phase::add_warning "CLI BATS suite not found"
  testing::phase::add_test skipped
fi

if [ -n "$API_URL" ]; then
  run_script_with_env "P1 feature smoke" "$TESTING_PHASE_SCENARIO_DIR/test/test-p1-features.sh" "API_PORT=$API_PORT_VALUE"
  run_script_with_env "Widget generation workflow" "$TESTING_PHASE_SCENARIO_DIR/test/test-widget-generation.sh" "API_BASE_URL=$API_URL"
else
  testing::phase::add_warning "Skipping API-driven business checks without resolved API URL"
  testing::phase::add_test skipped
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Business validation completed"
