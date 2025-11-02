#!/bin/bash
# Validates required files and directories for the brand-manager scenario.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "45s"

required_dirs=(api cli initialization scripts ui)
if testing::phase::check_directories "${required_dirs[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

required_files=(
  .vrooli/service.json
  Makefile
  PRD.md
  TEST_IMPLEMENTATION_SUMMARY.md
  api/main.go
  initialization/storage/postgres/schema.sql
  initialization/configuration/comfyui-workflows/logo-generator.json
)
if testing::phase::check_files "${required_files[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

testing::phase::check "Legacy scenario-test.yaml removed" test ! -f "${TESTING_PHASE_SCENARIO_DIR}/scenario-test.yaml"

testing::phase::end_with_summary "Structure validation completed"
