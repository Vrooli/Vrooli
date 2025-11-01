#!/bin/bash
# Validate required files and directories for the time-tools scenario shell.

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
  Makefile
  cli/time-tools
  ui/index.html
  initialization/storage/postgres/schema.sql
  test/run-tests.sh
  test/phases/test-unit.sh
)
# Prefer the modern service definition but fall back if migration incomplete.
if [ -f ".vrooli/service.json" ]; then
  required_files+=(".vrooli/service.json")
else
  required_files+=("service.json")
fi

if testing::phase::check_files "${required_files[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

testing::phase::end_with_summary "Structure validation completed"
