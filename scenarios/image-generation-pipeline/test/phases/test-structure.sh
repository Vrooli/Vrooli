#!/bin/bash
# Validates foundational files and directories for the scenario shell.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "45s"

# Required directories ensure API/UI/CLI and initialization assets are present.
required_directories=(
  api
  ui
  cli
  initialization
  data
  test/phases
  .vrooli
)
if testing::phase::check_directories "${required_directories[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

# Critical files that document and configure the scenario.
required_files=(
  PRD.md
  .vrooli/service.json
  Makefile
  test/run-tests.sh
)
if testing::phase::check_files "${required_files[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

if [ ! -f "README.md" ]; then
  testing::phase::add_warning "README.md missing; add scenario documentation to align with standards"
fi

testing::phase::end_with_summary "Structure validation completed"
