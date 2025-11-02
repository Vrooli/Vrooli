#!/bin/bash
# Ensures language runtimes and package manifests resolve without installing heavy assets.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"

run_go_dependencies() {
  if [ ! -f "api/go.mod" ]; then
    testing::phase::add_warning "Go module not detected; skipping Go dependency check"
    testing::phase::add_test skipped
    return 0
  fi

  if ! command -v go >/dev/null 2>&1; then
    testing::phase::add_warning "Go toolchain not available; cannot verify Go dependencies"
    testing::phase::add_test skipped
    return 0
  fi

  testing::phase::check "Go module graph resolves" bash -c 'cd api && go list ./... >/dev/null'
}

run_node_dependencies() {
  if [ ! -f "ui/package.json" ]; then
    testing::phase::add_warning "UI package.json not found; skipping Node dependency check"
    testing::phase::add_test skipped
    return 0
  fi

  local package_manager="npm"
  if command -v pnpm >/dev/null 2>&1 && grep -q 'packageManager' ui/package.json; then
    package_manager="pnpm"
  fi

  case "$package_manager" in
    pnpm)
      testing::phase::check "pnpm install --dry-run" bash -c 'cd ui && pnpm install --frozen-lockfile --dry-run >/dev/null'
      ;;
    *)
      if ! command -v npm >/dev/null 2>&1; then
        testing::phase::add_warning "npm CLI missing; cannot verify Node dependencies"
        testing::phase::add_test skipped
        return 0
      fi
      testing::phase::check "npm install --dry-run" bash -c 'cd ui && npm install --ignore-scripts --dry-run >/dev/null'
      ;;
  esac
}

run_go_dependencies || true
run_node_dependencies || true

testing::phase::end_with_summary "Dependency validation completed"
