#!/bin/bash
# Verifies language/runtime dependencies resolve without mutating the workspace.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "90s"

if [ -f "api/go.mod" ]; then
  testing::phase::check "Go modules resolve" bash -c 'cd api && GO111MODULE=on go list ./... >/dev/null'
  testing::phase::check "Go modules verified" bash -c 'cd api && GO111MODULE=on go mod verify >/dev/null'
else
  testing::phase::add_warning "Go module not detected; skipping Go dependency checks"
  testing::phase::add_test skipped
fi

if [ -f "cli/scenario-dependency-analyzer" ]; then
  testing::phase::check "CLI runtime dependencies present" bash -c 'grep -q "SCENARIO_NAME" cli/scenario-dependency-analyzer'
fi

testing::phase::end_with_summary "Dependency validation completed"
