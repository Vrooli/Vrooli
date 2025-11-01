#!/bin/bash
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "90s"

cd "$TESTING_PHASE_SCENARIO_DIR"

if command -v go >/dev/null 2>&1; then
  testing::phase::check "Business logic tests" bash -c 'cd api && go test -run "^TestBusinessLogic" -timeout=90s ./...'
else
  testing::phase::add_warning "Go toolchain not found; skipping business logic suite"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Business validation completed"
