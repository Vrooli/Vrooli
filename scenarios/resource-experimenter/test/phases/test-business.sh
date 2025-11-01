#!/bin/bash
# Exercises business workflows such as creating an experiment via the API
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s" --require-runtime

SCENARIO_NAME="$TESTING_PHASE_SCENARIO_NAME"
API_BASE=""

if API_BASE=$(testing::connectivity::get_api_url "$SCENARIO_NAME"); then
  testing::phase::add_test passed
else
  testing::phase::add_error "Failed to resolve API URL for $SCENARIO_NAME"
  testing::phase::end_with_summary "Business tests incomplete"
fi

if ! command -v curl >/dev/null 2>&1; then
  testing::phase::add_error "curl is required for business workflow checks"
  testing::phase::end_with_summary "Business tests incomplete"
fi

if ! command -v jq >/dev/null 2>&1; then
  testing::phase::add_warning "jq not installed; JSON assertions will be skipped"
fi

EXPERIMENT_NAME="cli-validation-${RANDOM}"
PAYLOAD='{
  "name": "%s",
  "description": "Automated validation",
  "prompt": "Add redis to analytics-dashboard",
  "target_scenario": "analytics-dashboard",
  "new_resource": "redis"
}'

create_experiment() {
  local payload
  payload=$(printf "$PAYLOAD" "$EXPERIMENT_NAME")
  RESPONSE=$(curl -sSf --max-time 20 \
    -H 'Content-Type: application/json' \
    -d "$payload" \
    "${API_BASE}/api/experiments")
  echo "$RESPONSE" > "$TESTING_PHASE_SCENARIO_DIR/coverage/phase-results/business-last-response.json"
  if command -v jq >/dev/null 2>&1; then
    echo "$RESPONSE" | jq -e --arg name "$EXPERIMENT_NAME" '(.name == $name) and (.status // "") != ""' >/dev/null
  fi
}

verify_list_contains_experiment() {
  local response
  response=$(curl -sSf --max-time 20 "${API_BASE}/api/experiments")
  if command -v jq >/dev/null 2>&1; then
    echo "$response" | jq -e --arg name "$EXPERIMENT_NAME" 'map(select(.name == $name)) | length >= 1' >/dev/null
  fi
}

testing::phase::check "create experiment workflow" create_experiment

testing::phase::check "experiments list reflects new entry" verify_list_contains_experiment

if command -v resource-claude-code >/dev/null 2>&1; then
  testing::phase::check "resource-claude-code status" resource-claude-code status
else
  testing::phase::add_warning "resource-claude-code CLI not found; skipping delegated status check"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Business workflow validation completed"
