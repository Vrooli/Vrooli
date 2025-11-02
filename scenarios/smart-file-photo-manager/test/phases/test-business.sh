#!/bin/bash
# Business workflow validation for smart-file-photo-manager

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s" --require-runtime

SCENARIO_NAME="$TESTING_PHASE_SCENARIO_NAME"
API_BASE_URL=""

if API_BASE_URL=$(testing::connectivity::get_api_url "$SCENARIO_NAME"); then
  SEARCH_PAYLOAD='{"query":"vacation beach","type":"semantic","limit":5}'
  SEARCH_RESPONSE=$(curl -s -X POST "$API_BASE_URL/api/search" -H "Content-Type: application/json" -d "$SEARCH_PAYLOAD" || true)
  if echo "$SEARCH_RESPONSE" | grep -q '"files"'; then
    testing::phase::add_test passed
  else
    testing::phase::add_error "Semantic search response did not include files array: $SEARCH_RESPONSE"
    testing::phase::add_test failed
  fi

  STATS_CHECK=$(curl -s "$API_BASE_URL/api/stats" || true)
  if echo "$STATS_CHECK" | grep -q '"total_files"'; then
    testing::phase::add_test passed
  else
    testing::phase::add_error "Stats endpoint response unexpected: $STATS_CHECK"
    testing::phase::add_test failed
  fi
else
  testing::phase::add_error "Unable to determine API URL for $SCENARIO_NAME"
  testing::phase::add_test failed
fi

if command -v smart-file-photo-manager >/dev/null 2>&1; then
  if smart-file-photo-manager health >/dev/null 2>&1; then
    testing::phase::add_test passed
  else
    testing::phase::add_error "CLI health command failed"
    testing::phase::add_test failed
  fi
else
  testing::phase::add_warning "smart-file-photo-manager CLI not installed; skipping CLI validation"
  testing::phase::add_test skipped
fi

# Retain existing lightweight upload smoke check for business workflows
UPLOAD_SCRIPT="${TESTING_PHASE_SCENARIO_DIR}/test/test-upload-endpoint.sh"
if [ -x "$UPLOAD_SCRIPT" ]; then
  if SMART_FILE_MANAGER_API_URL="$API_BASE_URL" "$UPLOAD_SCRIPT" >/dev/null 2>&1; then
    testing::phase::add_test passed
  else
    testing::phase::add_error "Standalone upload smoke script failed"
    testing::phase::add_test failed
  fi
fi

testing::phase::end_with_summary "Business workflow validation completed"
