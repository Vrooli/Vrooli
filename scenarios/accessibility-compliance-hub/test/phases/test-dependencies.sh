#!/bin/bash
# Verify language dependencies resolve for accessibility-compliance-hub.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

run_go_check() {
  if [ ! -f "api/go.mod" ]; then
    testing::phase::add_warning "No Go module detected; skipping Go dependency check"
    testing::phase::add_test skipped
    return 0
  fi

  if ! command -v go >/dev/null 2>&1; then
    testing::phase::add_warning "Go toolchain not available; skipping Go dependency check"
    testing::phase::add_test skipped
    return 0
  fi

  testing::phase::check "go list ./... succeeds" bash -c 'cd api && go list ./... >/dev/null'
}

run_node_check() {
  if [ ! -f "ui/package.json" ]; then
    testing::phase::add_warning "UI package.json not found; skipping Node dependency check"
    testing::phase::add_test skipped
    return 0
  fi

  local package_manager="npm"
  if command -v pnpm >/dev/null 2>&1 && grep -q '"packageManager"' ui/package.json; then
    package_manager="pnpm"
  fi

  if ! command -v "$package_manager" >/dev/null 2>&1; then
    testing::phase::add_warning "$package_manager not available; skipping Node dependency check"
    testing::phase::add_test skipped
    return 0
  fi

  if [ "$package_manager" = "pnpm" ]; then
    testing::phase::check "pnpm install --lockfile-only" bash -c 'cd ui && pnpm install --lockfile-only >/dev/null'
  else
    testing::phase::check "npm install --dry-run" bash -c 'cd ui && npm install --dry-run >/dev/null'
  fi
}

run_cli_dependencies() {
  if [ ! -f "cli/install.sh" ]; then
    testing::phase::add_warning "cli/install.sh missing; skipping CLI dependency check"
    testing::phase::add_test skipped
    return 0
  fi

  testing::phase::check "install script is executable" test -x "cli/install.sh"
}

run_go_check || true
run_node_check || true
run_cli_dependencies || true

testing::phase::end_with_summary "Dependency validation completed"
