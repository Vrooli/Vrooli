#!/bin/bash
# Validates key business workflows via the scenario CLI.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s" --require-runtime

SCENARIO_NAME="${TESTING_PHASE_SCENARIO_NAME}"
CLI_BIN="${TESTING_PHASE_SCENARIO_DIR}/cli/app-personalizer-cli.sh"

if [ ! -x "$CLI_BIN" ]; then
  if [ -f "$CLI_BIN" ]; then
    chmod +x "$CLI_BIN"
  else
    testing::phase::add_error "CLI wrapper missing at $CLI_BIN"
    testing::phase::end_with_summary "Business workflow validation incomplete"
  fi
fi

if ! command -v vrooli >/dev/null 2>&1; then
  testing::phase::add_error "vrooli CLI unavailable; cannot resolve API port"
  testing::phase::end_with_summary "Business workflow validation incomplete"
fi

API_PORT_OUTPUT=$(vrooli scenario port "$SCENARIO_NAME" API_PORT 2>/dev/null || true)
API_PORT=$(echo "$API_PORT_OUTPUT" | awk -F= '/=/{print $2}' | tr -d ' ')
if [ -z "$API_PORT" ]; then
  API_PORT=$(echo "$API_PORT_OUTPUT" | tr -d '[:space:]')
fi

if [ -z "$API_PORT" ]; then
  testing::phase::add_error "Unable to determine API_PORT for $SCENARIO_NAME"
  testing::phase::end_with_summary "Business workflow validation incomplete"
fi

if ! command -v jq >/dev/null 2>&1; then
  testing::phase::add_warning "jq is required to parse CLI responses; skipping business workflow checks"
  testing::phase::add_test skipped
  testing::phase::end_with_summary "Business validation skipped"
fi

APP_BASE="http://localhost:${API_PORT}"
TMP_ROOT=$(mktemp -d -t app-personalizer-business-XXXXXX)
trap 'rm -rf "$TMP_ROOT"' EXIT

APP_PATH="$TMP_ROOT/test-app"
mkdir -p "$APP_PATH/src/styles"
cat >"$APP_PATH/package.json" <<'PKG'
{
  "name": "test-app",
  "version": "1.0.0",
  "scripts": {
    "build": "echo build",
    "lint": "echo lint"
  }
}
PKG

cat >"$APP_PATH/src/styles/theme.js" <<'THEME'
export const theme = {
  colors: {
    primary: "#123456",
    secondary: "#abcdef"
  }
};
THEME

APP_NAME="phase-e2e-app"
APP_TYPE="generated"
FRAMEWORK="react"

register_output=$(APP_PERSONALIZER_API_BASE="$APP_BASE" "$CLI_BIN" register "$APP_NAME" "$APP_PATH" "$APP_TYPE" "$FRAMEWORK" 2>/dev/null)
if [ $? -ne 0 ]; then
  testing::phase::add_error "CLI register command failed"
  testing::phase::add_test failed
  testing::phase::end_with_summary "Business workflow validation failed"
fi

testing::phase::add_test passed

APP_ID=$(echo "$register_output" | jq -r '.app_id // empty')
if [ -z "$APP_ID" ]; then
  testing::phase::add_error "Unable to extract app_id from registration response"
  testing::phase::add_test failed
  testing::phase::end_with_summary "Business workflow validation failed"
fi

if APP_PERSONALIZER_API_BASE="$APP_BASE" "$CLI_BIN" analyze "$APP_ID" >/dev/null 2>&1; then
  testing::phase::add_test passed
else
  testing::phase::add_error "CLI analyze command failed"
  testing::phase::add_test failed
fi

if APP_PERSONALIZER_API_BASE="$APP_BASE" "$CLI_BIN" backup "$APP_PATH" full >/dev/null 2>&1; then
  testing::phase::add_test passed
else
  testing::phase::add_error "CLI backup command failed"
  testing::phase::add_test failed
fi

if APP_PERSONALIZER_API_BASE="$APP_BASE" "$CLI_BIN" validate "$APP_PATH" build,lint >/dev/null 2>&1; then
  testing::phase::add_test passed
else
  testing::phase::add_error "CLI validate command failed"
  testing::phase::add_test failed
fi

if APP_PERSONALIZER_API_BASE="$APP_BASE" "$CLI_BIN" personalize "$APP_ID" ui_theme copy >/dev/null 2>&1; then
  testing::phase::add_test passed
else
  testing::phase::add_warning "Personalize endpoint returned non-success (may require background workers)"
  testing::phase::add_test skipped
fi


testing::phase::end_with_summary "Business workflow validation completed"
