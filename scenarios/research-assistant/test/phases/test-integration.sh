#!/bin/bash
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "240s" --require-runtime

cd "$TESTING_PHASE_SCENARIO_DIR"

run_bats_suite() {
  local title="$1"
  local file="$2"

  if ! command -v bats >/dev/null 2>&1; then
    testing::phase::add_warning "BATS not installed; skipping ${title}"
    testing::phase::add_test skipped
    return 0
  fi

  if bats "$file"; then
    testing::phase::add_test passed
  else
    testing::phase::add_error "${title} failed"
    testing::phase::add_test failed
  fi
}

if [ -f "cli/vrooli.bats" ]; then
  run_bats_suite "CLI integration tests" "cli/vrooli.bats"
fi

if [ -f "test.bats" ]; then
  run_bats_suite "Scenario BATS tests" "test.bats"
fi

if [ -x "custom-tests.sh" ]; then
  if ./custom-tests.sh >/dev/null; then
    testing::phase::add_test passed
  else
    testing::phase::add_error "Custom workflow validation failed"
    testing::phase::add_test failed
  fi
else
  testing::phase::add_warning "custom-tests.sh not executable; skipping"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Integration tests completed"
