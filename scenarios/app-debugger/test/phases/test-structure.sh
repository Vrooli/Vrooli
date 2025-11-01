#!/bin/bash
# Validates required files and lint targets for app-debugger

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

required_dirs=(api cli data initialization test ui)
if testing::phase::check_directories "${required_dirs[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

required_files=(
  PRD.md
  README.md
  Makefile
  .vrooli/service.json
)
if testing::phase::check_files "${required_files[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

if command -v make >/dev/null 2>&1; then
  testing::phase::check "Run scenario lint target" bash -c 'make lint >/dev/null'
else
  testing::phase::add_warning "make not available; skipping lint target"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Structure validation completed"
