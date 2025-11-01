#!/bin/bash
# Verify language toolchains and core dependencies resolve without side effects

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

run_go_dependency_check() {
  if [ ! -f "api/go.mod" ]; then
    testing::phase::add_warning "api/go.mod not found; skipping Go dependency check"
    testing::phase::add_test skipped
    return 0
  fi

  if ! command -v go >/dev/null 2>&1; then
    testing::phase::add_warning "Go toolchain not available"
    testing::phase::add_test skipped
    return 0
  fi

  testing::phase::check "go list ./..." bash -c 'cd api && go list ./... >/dev/null'
}

run_ui_dependency_check() {
  if [ ! -f "ui/package.json" ]; then
    testing::phase::add_warning "ui/package.json not found; skipping Node dependency check"
    testing::phase::add_test skipped
    return 0
  fi

  if ! command -v npm >/dev/null 2>&1; then
    testing::phase::add_warning "npm not available; cannot verify UI dependencies"
    testing::phase::add_test skipped
    return 0
  fi

  testing::phase::check "npm install --dry-run" bash -c 'cd ui && npm install --dry-run >/dev/null'
}

run_cli_dependency_check() {
  if [ ! -f "cli/code-tidiness-manager.bats" ]; then
    testing::phase::add_warning "CLI BATS tests missing; skipping CLI dependency check"
    testing::phase::add_test skipped
    return 0
  fi

  if ! command -v bats >/dev/null 2>&1; then
    testing::phase::add_warning "bats not available; CLI tests cannot run"
    testing::phase::add_test skipped
    return 0
  fi

  testing::phase::add_test passed
  return 0
}

run_go_dependency_check || true
run_ui_dependency_check || true
run_cli_dependency_check || true

if command -v redis-cli >/dev/null 2>&1; then
  testing::phase::check "redis-cli ping" redis-cli ping >/dev/null
else
  testing::phase::add_warning "redis-cli not found; cache validation skipped"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Dependency validation completed"
