#!/bin/bash
# Dependency validation for Game Dialog Generator
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

run_go_check() {
  if [ ! -f "api/go.mod" ]; then
    testing::phase::add_warning "Go module not found; skipping Go dependency check"
    testing::phase::add_test skipped
    return 0
  fi

  if ! command -v go >/dev/null 2>&1; then
    testing::phase::add_warning "Go toolchain not available"
    testing::phase::add_test skipped
    return 0
  fi

  testing::phase::check "Go module graph resolves" bash -c 'cd api && go list ./... >/dev/null'
}

run_node_check() {
  if [ ! -f "ui/package.json" ]; then
    testing::phase::add_warning "ui/package.json missing; skipping Node check"
    testing::phase::add_test skipped
    return 0
  fi

  if ! command -v npm >/dev/null 2>&1; then
    testing::phase::add_warning "npm not available for dependency validation"
    testing::phase::add_test skipped
    return 0
  fi

  testing::phase::check "npm install --dry-run" bash -c 'cd ui && npm install --dry-run >/dev/null'
}

run_cli_check() {
  if [ ! -f "cli/game-dialog-generator" ]; then
    testing::phase::add_warning "CLI binary missing; skipping CLI dependency check"
    testing::phase::add_test skipped
    return 0
  fi

  testing::phase::check "CLI dependency check (install script present)" test -f cli/install.sh
}

run_go_check || true
run_node_check || true
run_cli_check || true

testing::phase::end_with_summary "Dependency validation completed"
