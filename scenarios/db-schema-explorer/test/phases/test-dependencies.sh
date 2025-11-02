#!/bin/bash
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

run_go_dependencies() {
  if [ ! -f "api/go.mod" ]; then
    testing::phase::add_warning "Go module missing; skipping Go dependency check"
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

run_ui_dependencies() {
  if [ ! -f "ui/package.json" ]; then
    testing::phase::add_warning "UI package.json missing; skipping Node dependency check"
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

run_cli_dependencies() {
  if [ ! -d "cli" ]; then
    testing::phase::add_warning "CLI directory missing; skipping CLI dependency check"
    testing::phase::add_test skipped
    return 0
  fi

  testing::phase::check "CLI install script present" test -f cli/install.sh

  if ! command -v bats >/dev/null 2>&1; then
    testing::phase::add_warning "BATS not installed; CLI phase will skip"
  fi
}

run_go_dependencies || true
run_ui_dependencies || true
run_cli_dependencies || true

testing::phase::end_with_summary "Dependency validation completed"
