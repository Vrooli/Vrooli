#!/bin/bash
# Validates scenario layout and required artifacts.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

required_dirs=(api ui cli test initialization)
if testing::phase::check_directories "${required_dirs[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

required_files=(
  PRD.md
  README.md
  .vrooli/service.json
  cli/qr-generator
  test/run-tests.sh
)
if testing::phase::check_files "${required_files[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

testing::phase::check "CLI script parses" bash -c 'bash -n cli/qr-generator'

testing::phase::end_with_summary "Structure validation completed"
