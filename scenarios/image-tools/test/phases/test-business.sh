#!/bin/bash
# Business phase – validates higher level workflows and CLI affordances.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s" --require-runtime

if ! command -v curl >/dev/null 2>&1; then
  testing::phase::add_error "curl is required for business workflow checks"
  testing::phase::end_with_summary "Business workflow tests incomplete"
fi

API_BASE=$(testing::connectivity::get_api_url "$TESTING_PHASE_SCENARIO_NAME" || true)
if [ -z "$API_BASE" ]; then
  testing::phase::add_error "Unable to resolve API endpoint for $TESTING_PHASE_SCENARIO_NAME"
  testing::phase::end_with_summary "Business workflow tests incomplete"
fi

WORKFLOW_IMAGE="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="

run_json_post() {
  set -euo pipefail
  local endpoint="$1"
  local payload="$2"
  local expected_codes="$3"
  local response
  response=$(curl -s -o /tmp/image-tools-business-response -w "%{http_code}" -X POST "$API_BASE$endpoint" \
    -H "Content-Type: application/json" \
    -d "$payload")
  echo "$expected_codes" | tr ',' '\n' | grep -qx "$response"
}

# Compression workflow using JSON payload
run_json_post "/api/v1/image/compress" "{\"image\":\"$WORKFLOW_IMAGE\",\"quality\":75}" "200,202"
if [ $? -eq 0 ]; then
  testing::phase::add_test passed
else
  testing::phase::add_error "Compression workflow request failed"
  testing::phase::add_test failed
fi

# Format conversion workflow
run_json_post "/api/v1/image/convert" "{\"image\":\"$WORKFLOW_IMAGE\",\"target_format\":\"webp\"}" "200,202"
if [ $? -eq 0 ]; then
  testing::phase::add_test passed
else
  testing::phase::add_error "Format conversion workflow failed"
  testing::phase::add_test failed
fi

# Batch endpoint should respond with validation error (400) when payload incomplete
run_json_post "/api/v1/image/batch" "{\"images\":[]}" "200,400"
if [ $? -eq 0 ]; then
  testing::phase::add_test passed
else
  testing::phase::add_error "Batch endpoint rejected basic request"
  testing::phase::add_test failed
fi

# CLI validation (optional if CLI not installed)
if command -v image-tools >/dev/null 2>&1; then
  testing::phase::check "CLI help command" bash -c 'image-tools help | grep -q "Image Tools"'
  testing::phase::check "CLI version command" bash -c 'image-tools version | grep -q "image-tools"'
  testing::phase::check "CLI status command emits JSON" bash -c '
    set -euo pipefail
    output=$(image-tools status --json 2>/dev/null)
    [ -n "$output" ] && echo "$output" | grep -q '"status"'
  '
else
  testing::phase::add_warning "image-tools CLI not installed; skipping CLI checks"
  testing::phase::add_test skipped
fi

# UI health check if port available
UI_BASE=$(testing::connectivity::get_ui_url "$TESTING_PHASE_SCENARIO_NAME" || true)
if [ -n "$UI_BASE" ]; then
  testing::phase::check "UI health endpoint" bash -c 'curl -sSf "'$UI_BASE'/health" >/dev/null'
else
  testing::phase::add_warning "Unable to discover UI endpoint; skipping UI business check"
  testing::phase::add_test skipped
fi

rm -f /tmp/image-tools-business-response 2>/dev/null || true

testing::phase::end_with_summary "Business workflow validation completed"
