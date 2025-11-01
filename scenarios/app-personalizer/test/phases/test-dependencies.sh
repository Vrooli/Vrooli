#!/bin/bash
# Ensures language/runtime dependencies resolve without external installs.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

run_go_dependency_check() {
  if [ ! -f "api/go.mod" ]; then
    testing::phase::add_warning "No Go module detected; skipping go mod tidy"
    testing::phase::add_test skipped
    return 0
  fi

  if ! command -v go >/dev/null 2>&1; then
    testing::phase::add_warning "Go toolchain not available; dependency graph not validated"
    testing::phase::add_test skipped
    return 0
  fi

  testing::phase::check "Go modules resolve" bash -c 'cd api && go list ./... >/dev/null'
}

run_cli_dependency_check() {
  if [ ! -f "cli/app-personalizer.bats" ]; then
    testing::phase::add_warning "CLI BATS suite missing; skipping CLI dependency check"
    testing::phase::add_test skipped
    return 0
  fi

  if ! command -v bats >/dev/null 2>&1; then
    testing::phase::add_warning "BATS not installed; CLI tests may be skipped later"
    testing::phase::add_test skipped
    return 0
  fi

  testing::phase::add_test passed
}

run_go_dependency_check || true
run_cli_dependency_check || true

testing::phase::end_with_summary "Dependency validation completed"
