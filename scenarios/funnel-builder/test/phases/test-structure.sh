#!/bin/bash
# Validate required files and directories for funnel-builder.
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

required_dirs=(api ui cli initialization .vrooli)
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
)
if testing::phase::check_files "${required_files[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

# Ensure the CLI entrypoint is executable to keep automations healthy.
if [ -f "cli/funnel-builder" ]; then
  testing::phase::check "CLI entrypoint executable" test -x "cli/funnel-builder"
else
  testing::phase::add_warning "cli/funnel-builder missing; CLI packaging incomplete"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Structure validation completed"
