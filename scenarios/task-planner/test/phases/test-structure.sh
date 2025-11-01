#!/bin/bash
# Validate required files and directories for task-planner scenario

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

required_dirs=(api ui cli initialization data test)
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
  cli/install.sh
)
if testing::phase::check_files "${required_files[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

# warn if legacy scenario-test.yaml still present
if [ -f "scenario-test.yaml" ]; then
  testing::phase::add_warning "Legacy scenario-test.yaml detected; remove once migration completes"
fi

testing::phase::end_with_summary "Structure validation completed"
