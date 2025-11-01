#!/bin/bash
# Validate critical files and directories exist for app-monitor

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

# Ensure repository layout is intact.
if testing::phase::check_directories api ui cli initialization docs test; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

required_files=(
  README.md
  PRD.md
  Makefile
  comprehensive-test.sh
  .vrooli/service.json
  test/run-tests.sh
  api/main.go
  ui/package.json
)
if testing::phase::check_files "${required_files[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

testing::phase::end_with_summary "Structure validation completed"
