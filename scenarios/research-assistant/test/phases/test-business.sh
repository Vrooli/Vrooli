#!/bin/bash
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"

validate_json_files() {
  local label="$1"
  shift
  local files=("$@")

  if ! command -v jq >/dev/null 2>&1; then
    testing::phase::add_warning "jq not installed; skipping ${label} validation"
    testing::phase::add_test skipped
    return
  fi

  local all_good=true
  for file in "${files[@]}"; do
    if [ ! -f "$file" ]; then
      testing::phase::add_warning "Expected file missing: $file"
      all_good=false
      continue
    fi
    if ! jq empty "$file" >/dev/null 2>&1; then
      testing::phase::add_error "Invalid JSON in $file"
      all_good=false
    fi
  done

  if [ "$all_good" = true ]; then
    testing::phase::add_test passed
  else
    testing::phase::add_test failed
  fi
}

if [ -d "initialization/automation/n8n" ]; then
  mapfile -t workflow_files < <(find initialization/automation/n8n -maxdepth 1 -type f -name '*.json' ! -name '*template*')
  if [ ${#workflow_files[@]} -gt 0 ]; then
    validate_json_files "n8n workflows" "${workflow_files[@]}"
  fi
fi

validate_json_files "configuration files" \
  "initialization/configuration/research-config.json" \
  "initialization/configuration/report-templates.json" \
  "initialization/configuration/schedule-presets.json"

if grep -q "handleHealth" api/main.go; then
  testing::phase::add_test passed
else
  testing::phase::add_warning "API health handler reference not found"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Business validations completed"
