#!/bin/bash
# Validates CLI and API integration paths while the scenario runtime is active.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s" --require-runtime

CLI_BIN="${TESTING_PHASE_SCENARIO_DIR}/cli/qr-generator"
TMP_FILE=""

cleanup_tmp_artifacts() {
  if [ -n "$TMP_FILE" ] && [ -f "$TMP_FILE" ]; then
    rm -f "$TMP_FILE"
  fi
}

testing::phase::register_cleanup cleanup_tmp_artifacts

if [ -f "$CLI_BIN" ]; then
  TMP_FILE="$(mktemp -t qr-generator-int-XXXXXX).png"
  if bash "$CLI_BIN" generate "Integration test" --output "$TMP_FILE" --size 256; then
    if [ -f "$TMP_FILE" ]; then
      testing::phase::add_test passed
    else
      testing::phase::add_error "CLI generation succeeded but output file missing"
      testing::phase::add_test failed
    fi
  else
    testing::phase::add_error "CLI generation command failed"
    testing::phase::add_test failed
  fi
else
  testing::phase::add_warning "CLI script not found; skipping CLI integration"
  testing::phase::add_test skipped
fi

API_URL="$(testing::connectivity::get_api_url "$TESTING_PHASE_SCENARIO_NAME" || true)"
if [ -n "$API_URL" ]; then
  if response=$(curl -sf "${API_URL}/health" 2>/dev/null); then
    if echo "$response" | grep -qi "healthy"; then
      testing::phase::add_test passed
      log::success "✅ API health endpoint responded with healthy status"
    else
      testing::phase::add_error "API health endpoint response missing healthy indicator"
      testing::phase::add_test failed
    fi
  else
    testing::phase::add_error "API health endpoint unreachable"
    testing::phase::add_test failed
  fi
else
  testing::phase::add_error "Unable to resolve API URL; ensure vrooli CLI is available"
  testing::phase::add_test failed
fi

testing::phase::end_with_summary "Integration validation completed"
