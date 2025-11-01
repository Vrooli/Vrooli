#!/bin/bash
# Validate required files and directories exist for the calendar scenario shell.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

required_dirs=(api ui docs initialization cli)
if testing::phase::check_directories "${required_dirs[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

required_files=(
  .vrooli/service.json
  README.md
  PRD.md
  Makefile
)
if testing::phase::check_files "${required_files[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

# Ensure legacy scenario-test.yaml has been removed
if [ -f "scenario-test.yaml" ]; then
  testing::phase::add_warning "Legacy scenario-test.yaml still present"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Structure validation completed"
