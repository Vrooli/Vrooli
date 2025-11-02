#!/bin/bash
# Validates core brand-generation assets and configuration artifacts.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s"

domain_artifacts=(
  initialization/automation/n8n/brand-pipeline.json
  initialization/configuration/comfyui-workflows/icon-creator.json
  initialization/storage/postgres/seed.sql
)
if testing::phase::check_files "${domain_artifacts[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

if command -v jq >/dev/null 2>&1; then
  testing::phase::check "n8n brand pipeline JSON is valid" bash -c 'jq empty initialization/automation/n8n/brand-pipeline.json'
  testing::phase::check "ComfyUI logo workflow JSON is valid" bash -c 'jq empty initialization/configuration/comfyui-workflows/logo-generator.json'
else
  testing::phase::add_warning "jq not available; skipping JSON structure validation"
  testing::phase::add_test skipped
fi

testing::phase::check "Postgres seed prepares brand tables" grep -q "INSERT INTO" initialization/storage/postgres/seed.sql

testing::phase::end_with_summary "Business validation completed"
