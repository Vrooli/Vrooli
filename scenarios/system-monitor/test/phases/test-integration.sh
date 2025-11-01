#!/bin/bash
# Integration phase executes scenario-specific service tests.

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "240s"

cd "$TESTING_PHASE_SCENARIO_DIR"

if [ -d "api" ]; then
  if command -v go >/dev/null 2>&1; then
    testing::phase::check "Go integration test suite" bash -c 'cd api && go test -v -tags=integration ./... -timeout 240s'
  else
    testing::phase::add_warning "Go toolchain missing; skipping integration tests"
    testing::phase::add_test skipped
  fi
else
  testing::phase::add_warning "API directory missing; skipping integration tests"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Integration tests completed"
