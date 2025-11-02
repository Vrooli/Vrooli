#!/bin/bash
# Validates critical files and directories for the scenario shell.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "45s"

required_dirs=(
  api
  cli
  ui
  test
  initialization
  data
  .vrooli
)
if testing::phase::check_directories "${required_dirs[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

required_files=(
  README.md
  Makefile
  .vrooli/service.json
  test/run-tests.sh
  TEST_IMPLEMENTATION_SUMMARY.md
)
if testing::phase::check_files "${required_files[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

if [ -f "scenario-test.yaml" ]; then
  testing::phase::add_warning "Legacy scenario-test.yaml found; phased testing should not keep this file"
fi

testing::phase::end_with_summary "Structure validation completed"
