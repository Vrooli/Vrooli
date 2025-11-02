#!/bin/bash
# Ensures language runtimes and package manifests resolve without downloading heavy assets.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "90s"

run_go_check() {
  if [ ! -f "api/go.mod" ]; then
    testing::phase::add_warning "Go module not detected; skipping Go dependency check"
    testing::phase::add_test skipped
    return 0
  fi

  if command -v go >/dev/null 2>&1; then
    testing::phase::check "Go module graph resolves" bash -c 'cd api && go list ./... >/dev/null'
    return $?
  fi

  testing::phase::add_warning "Go toolchain missing; cannot verify Go dependencies"
  testing::phase::add_test skipped
  return 0
}

run_ui_node_check() {
  if [ ! -f "ui/package.json" ]; then
    testing::phase::add_warning "UI package.json not found; skipping UI dependency check"
    testing::phase::add_test skipped
    return 0
  fi

  if command -v npm >/dev/null 2>&1; then
    testing::phase::check "UI npm install --dry-run" bash -c 'cd ui && npm install --dry-run >/dev/null'
    return $?
  fi

  testing::phase::add_warning "npm CLI missing; cannot verify UI dependencies"
  testing::phase::add_test skipped
  return 0
}

run_cli_check() {
  if [ ! -f "cli/package.json" ] && [ ! -f "cli/install.sh" ]; then
    testing::phase::add_warning "CLI tooling not detected; skipping CLI dependency check"
    testing::phase::add_test skipped
    return 0
  fi

  if [ -f "cli/package.json" ] && command -v npm >/dev/null 2>&1; then
    testing::phase::check "CLI npm install --dry-run" bash -c 'cd cli && npm install --dry-run >/dev/null'
    return $?
  fi

  testing::phase::add_warning "CLI dependency execution skipped (npm not available or package.json missing)"
  testing::phase::add_test skipped
  return 0
}

run_go_check || true
run_ui_node_check || true
run_cli_check || true

testing::phase::end_with_summary "Dependency validation completed"
