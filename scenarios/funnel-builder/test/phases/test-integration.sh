#!/bin/bash
# Runtime integration checks for funnel-builder services.
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "240s" --require-runtime

SCENARIO_NAME="$TESTING_PHASE_SCENARIO_NAME"

API_BASE=""
UI_BASE=""
if API_BASE=$(testing::connectivity::get_api_url "$SCENARIO_NAME"); then
  testing::phase::check "API health endpoint" bash -c "curl -sf \"${API_BASE}/health\" || curl -sf \"${API_BASE}/api/v1/health\""
else
  testing::phase::add_error "Unable to resolve API URL for $SCENARIO_NAME"
  testing::phase::add_test failed
fi

if UI_BASE=$(testing::connectivity::get_ui_url "$SCENARIO_NAME"); then
  testing::phase::check "UI responds" curl -sf "$UI_BASE"
else
  testing::phase::add_warning "Unable to resolve UI URL for $SCENARIO_NAME"
  testing::phase::add_test skipped
fi

CLI_PATH="$TESTING_PHASE_SCENARIO_DIR/cli/funnel-builder"
if [ -x "$CLI_PATH" ]; then
  testing::phase::check "CLI help command" "$CLI_PATH" --help
else
  testing::phase::add_warning "CLI executable missing or not executable"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Integration checks completed"
