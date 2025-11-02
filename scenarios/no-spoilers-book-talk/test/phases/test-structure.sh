#!/bin/bash
# Validates required project structure for no-spoilers-book-talk
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

required_dirs=(api cli data initialization test)
if testing::phase::check_directories "${required_dirs[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

required_files=(
  api/main.go
  cli/no-spoilers-book-talk
  cli/no-spoilers-book-talk.bats
  Makefile
  .vrooli/service.json
  README.md
  PRD.md
)
if testing::phase::check_files "${required_files[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

# Warn if legacy scenario-test.yaml still exists (should be removed post-migration)
if [ -f "scenario-test.yaml" ]; then
  testing::phase::add_warning "Legacy scenario-test.yaml detected; ensure phased tests are authoritative"
fi

testing::phase::end_with_summary "Structure validation completed"
