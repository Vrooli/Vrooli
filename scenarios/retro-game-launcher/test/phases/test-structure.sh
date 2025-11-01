#!/bin/bash
# Validates required files and directories for retro-game-launcher

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

required_dirs=(
  api
  cli
  ui
  initialization
  deployment
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
  api/main.go
  ui/package.json
  cli/retro-game-launcher
)
if testing::phase::check_files "${required_files[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

testing::phase::check "Legacy scenario-test.yaml removed" test ! -f scenario-test.yaml

testing::phase::end_with_summary "Structure validation completed"
