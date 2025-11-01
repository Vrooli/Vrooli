#!/bin/bash
# Validates the core file and directory layout for the study-buddy scenario.

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

required_dirs=(
  api
  ui
  cli
  initialization
  test
  data
)
if testing::phase::check_directories "${required_dirs[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

required_files=(
  .vrooli/service.json
  Makefile
  PRD.md
  README.md
  cli/study-buddy
  test/run-tests.sh
)
if testing::phase::check_files "${required_files[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

# Flag legacy configuration if it still exists to prevent regressions during the migration.
if [ -f "scenario-test.yaml" ]; then
  testing::phase::add_warning "Legacy scenario-test.yaml found; phased testing should own the lifecycle"
fi

testing::phase::end_with_summary "Structure validation completed"
