#!/bin/bash
# Validates scenario structure and required assets are present.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

required_dirs=(
  api
  cli
  data
  initialization
  test
)
if testing::phase::check_directories "${required_dirs[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

required_files=(
  README.md
  PRD.md
  .vrooli/service.json
  Makefile
  initialization/storage/postgres/schema.sql
  initialization/automation/n8n/personalization-pipeline.json
)
if testing::phase::check_files "${required_files[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

if [ -f "scenario-test.yaml" ]; then
  testing::phase::add_warning "Legacy scenario-test.yaml detected; phased tests supersede it."
fi

testing::phase::end_with_summary "Structure validation completed"
