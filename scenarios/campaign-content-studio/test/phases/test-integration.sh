#!/bin/bash
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "90s"

cd "$TESTING_PHASE_SCENARIO_DIR"

API_FILE="api/main.go"
UI_FILE="ui/server.js"

if [ -f "$API_FILE" ]; then
  declare -a endpoints=(
    "/health"
    "/campaigns"
    "/campaigns/:id/documents"
    "/content/generate"
    "/documents/search"
  )

  for endpoint in "${endpoints[@]}"; do
    testing::phase::check "API route registered: ${endpoint}" grep -q "$endpoint" "$API_FILE"
  done
else
  testing::phase::add_error "API main.go missing; cannot verify routes"
fi

if [ -f "$UI_FILE" ]; then
  testing::phase::check "UI exposes health endpoint" grep -q "/health" "$UI_FILE"
  testing::phase::check "UI exposes campaign dashboard" grep -q "/api/campaigns" "$UI_FILE"
else
  testing::phase::add_warning "UI server file missing"
  testing::phase::add_test skipped
fi

if [ -x "cli/campaign-content-studio" ]; then
  testing::phase::check "CLI exposes help" bash -c 'cli/campaign-content-studio --help >/dev/null'
else
  testing::phase::add_warning "CLI binary not built; skipping CLI smoke"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Integration validations completed"
