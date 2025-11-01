#!/bin/bash
# Business workflow validation for Secrets Manager
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "240s" --require-runtime

SCENARIO_NAME="${TESTING_PHASE_SCENARIO_NAME}"
API_URL=$(testing::connectivity::get_api_url "$SCENARIO_NAME" || true)
UI_URL=$(testing::connectivity::get_ui_url "$SCENARIO_NAME" || true)

if [ -z "$API_URL" ] || [ -z "$UI_URL" ]; then
  testing::phase::add_error "Unable to determine scenario endpoints"
  testing::phase::end_with_summary "Business workflow checks failed"
fi

API_PORT="${API_URL##*:}"
UI_PORT="${UI_URL##*:}"
SCRIPT_DIR="$TESTING_PHASE_SCENARIO_DIR"
export API_PORT UI_PORT SCRIPT_DIR

CLI_BIN="secrets-manager"
if ! command -v "$CLI_BIN" >/dev/null 2>&1; then
  CLI_BIN="${TESTING_PHASE_SCENARIO_DIR}/cli/secrets-manager"
fi

if [ -x "$CLI_BIN" ]; then
  testing::phase::check "CLI status command" "$CLI_BIN" status --json
  testing::phase::check "CLI scan command" "$CLI_BIN" scan --output json
  testing::phase::check "CLI validate command" "$CLI_BIN" validate --json
else
  testing::phase::add_warning "CLI binary not found; skipping CLI workflow checks"
  testing::phase::add_test skipped
fi

CUSTOM_TESTS="${TESTING_PHASE_SCENARIO_DIR}/custom-tests.sh"
if [ -f "$CUSTOM_TESTS" ]; then
  source "$CUSTOM_TESTS"

  run_custom_test() {
    local label="$1"
    local fn="$2"
    if "$fn"; then
      log::success "✅ $label"
      testing::phase::add_test passed
    else
      log::error "❌ $label failed"
      testing::phase::add_error "$label"
      testing::phase::add_test failed
    fi
  }

  run_custom_test "Dark chrome UI elements" test_dark_chrome_ui_accessibility
  run_custom_test "Resource secret discovery pipeline" test_resource_secret_discovery
  run_custom_test "Vault integration checks" test_vault_secret_storage
  run_custom_test "Dashboard realtime widgets" test_dashboard_ui_loads
  run_custom_test "CLI advanced features" test_cli_advanced_features
else
  testing::phase::add_warning "custom-tests.sh not found; skipping business-specific assertions"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Business workflow validation completed"
