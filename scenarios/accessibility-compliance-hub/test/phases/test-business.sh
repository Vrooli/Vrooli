#!/bin/bash
# Business logic checks for accessibility-compliance-hub.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"

CLI_BINARY="cli/accessibility-compliance-hub"

if [ -x "$CLI_BINARY" ]; then
  testing::phase::check "CLI help command" "$CLI_BINARY" help
  testing::phase::check "CLI version command" "$CLI_BINARY" version
else
  testing::phase::add_warning "CLI binary unavailable; run cli/install.sh to generate it"
  testing::phase::add_test skipped
fi

validate_json() {
  local label="$1"
  local path="$2"

  if [ ! -f "$path" ]; then
    testing::phase::add_warning "$label missing at $path"
    testing::phase::add_test skipped
    return
  fi

  if command -v jq >/dev/null 2>&1; then
    testing::phase::check "$label is valid JSON" jq empty "$path"
  else
    testing::phase::add_warning "jq missing; skipped validation for $label"
    testing::phase::add_test skipped
  fi
}

validate_json "Audit workflow definition" "initialization/n8n/scheduled-reports.json"
validate_json "Threshold monitor definition" "initialization/n8n/threshold-monitor.json"
validate_json "Monitoring configuration" "initialization/configuration/monitoring-config.json"

testing::phase::end_with_summary "Business checks completed"
