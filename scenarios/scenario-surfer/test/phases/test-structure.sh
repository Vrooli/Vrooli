#!/bin/bash
# Validates required files and directories for Scenario Surfer

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

required_files=(
  api/main.go
  cli/install.sh
  cli/scenario-surfer
  ui/index.html
  ui/server.js
  .vrooli/service.json
  Makefile
  PRD.md
  README.md
)
if testing::phase::check_files "${required_files[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

required_dirs=(api ui cli initialization test)
if testing::phase::check_directories "${required_dirs[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

testing::phase::end_with_summary "Structure validation completed"
