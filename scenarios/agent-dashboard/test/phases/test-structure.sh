#!/bin/bash
# Validate required files and directories for the Agent Dashboard scenario
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

required_dirs=(
  api
  cli
  ui
  .vrooli
)
if testing::phase::check_directories "${required_dirs[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

required_files=(
  README.md
  PRD.md
  Makefile
  .vrooli/service.json
)
if testing::phase::check_files "${required_files[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

# Ensure legacy scenario-test.yaml has been removed
if [ -f "scenario-test.yaml" ]; then
  testing::phase::add_warning "Legacy scenario-test.yaml detected; phased architecture should be used"
  testing::phase::add_error "Legacy scenario-test.yaml still present"
else
  testing::phase::check "Legacy test file removed" test ! -f "scenario-test.yaml"
fi

testing::phase::end_with_summary "Structure validation completed"
