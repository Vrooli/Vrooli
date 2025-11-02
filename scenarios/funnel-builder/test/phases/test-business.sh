#!/bin/bash
# Business workflow validation (currently focused on CLI contract smoke tests).
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s" --require-runtime

CLI_BATS="$TESTING_PHASE_SCENARIO_DIR/cli/funnel-builder.bats"
if [ -f "$CLI_BATS" ]; then
  if command -v bats >/dev/null 2>&1; then
    testing::phase::check "CLI BATS suite" bats "$CLI_BATS"
  else
    testing::phase::add_warning "bats not installed; skipping CLI workflow validation"
    testing::phase::add_test skipped
  fi
else
  testing::phase::add_warning "cli/funnel-builder.bats missing; business workflow tests not yet defined"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Business validation completed"
