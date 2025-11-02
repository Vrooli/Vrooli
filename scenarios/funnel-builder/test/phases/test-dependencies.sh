#!/bin/bash
# Ensure language dependencies resolve without modifying local state.
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "90s"

run_go_dependency_check() {
  if [ ! -f "api/go.mod" ]; then
    testing::phase::add_warning "Go module not detected; skipping go list"
    testing::phase::add_test skipped
    return 0
  fi

  if ! command -v go >/dev/null 2>&1; then
    testing::phase::add_warning "Go toolchain not available; cannot verify dependencies"
    testing::phase::add_test skipped
    return 0
  fi

  testing::phase::check "go list ./..." bash -c 'cd api && go list ./... >/dev/null'
}

run_node_dependency_check() {
  if [ ! -f "ui/package.json" ]; then
    testing::phase::add_warning "UI package.json not found; skipping npm install --dry-run"
    testing::phase::add_test skipped
    return 0
  fi

  local package_manager="npm"
  if command -v jq >/dev/null 2>&1; then
    package_manager=$(jq -r '.packageManager // "npm"' ui/package.json 2>/dev/null | cut -d@ -f1)
    [ -z "$package_manager" ] && package_manager="npm"
  fi

  case "$package_manager" in
    pnpm)
      if command -v pnpm >/dev/null 2>&1; then
        testing::phase::check "pnpm install --lockfile-only" bash -c 'cd ui && pnpm install --lockfile-only >/dev/null'
      else
        testing::phase::add_warning "pnpm declared but not installed; skipping"
        testing::phase::add_test skipped
      fi
      ;;
    yarn)
      if command -v yarn >/dev/null 2>&1; then
        testing::phase::check "yarn install --mode=update-lock" bash -c 'cd ui && YARN_ENABLE_IMMUTABLE_INSTALLS=false yarn install --mode=update-lock --check-cache >/dev/null'
      else
        testing::phase::add_warning "yarn declared but not installed; skipping"
        testing::phase::add_test skipped
      fi
      ;;
    *)
      if command -v npm >/dev/null 2>&1; then
        testing::phase::check "npm install --dry-run" bash -c 'cd ui && npm install --dry-run >/dev/null'
      else
        testing::phase::add_warning "npm CLI not available; skipping UI dependency check"
        testing::phase::add_test skipped
      fi
      ;;
  esac
}

run_go_dependency_check || true
run_node_dependency_check || true

testing::phase::end_with_summary "Dependency validation completed"
