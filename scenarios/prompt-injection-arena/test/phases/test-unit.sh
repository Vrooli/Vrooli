#!/usr/bin/env bash
# Runs language-specific unit suites and CLI smoke coverage.
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s"
source "${APP_ROOT}/scripts/scenarios/testing/unit/run-all.sh"

cd "$TESTING_PHASE_SCENARIO_DIR"

testing::unit::run_all_tests \
  --go-dir "api" \
  --node-dir "ui" \
  --skip-python \
  --coverage-warn 60 \
  --coverage-error 40 \
  --verbose

if [ -f "cli/prompt-injection-arena.bats" ] && command -v bats >/dev/null 2>&1; then
  testing::phase::check "CLI BATS suite" bash -c 'cd cli && bats prompt-injection-arena.bats --tap'
else
  testing::phase::add_warning "CLI BATS suite unavailable; skipping CLI unit checks"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Unit tests completed"
