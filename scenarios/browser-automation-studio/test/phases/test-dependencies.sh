#!/bin/bash
# Ensures language runtimes and package manifests resolve without downloading.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

run_go_check() {
  if [ ! -f "api/go.mod" ]; then
    testing::phase::add_warning "Go module not detected; skipping Go dependency check"
    testing::phase::add_test skipped
    return 0
  fi

  if command -v go >/dev/null 2>&1; then
    if testing::phase::check "Go module graph resolves" bash -c 'cd api && go list ./... >/dev/null'; then
      return 0
    fi
  else
    testing::phase::add_warning "Go toolchain missing; cannot verify Go dependencies"
    testing::phase::add_test skipped
    return 0
  fi

  return 1
}

run_node_check() {
  if [ ! -f "ui/package.json" ]; then
    testing::phase::add_warning "UI package.json not found; skipping Node dependency check"
    testing::phase::add_test skipped
    return 0
  fi

  # Check for pnpm first (workspace protocol support)
  if command -v pnpm >/dev/null 2>&1; then
    # Just verify node_modules exists and package.json is readable
    if testing::phase::check "pnpm workspace dependencies" bash -c 'cd ui && [ -d node_modules ] && pnpm list >/dev/null 2>&1'; then
      return 0
    fi
  elif command -v npm >/dev/null 2>&1; then
    # Fallback to npm for non-workspace projects
    if testing::phase::check "npm install --dry-run" bash -c 'cd ui && npm install --dry-run >/dev/null 2>&1'; then
      return 0
    fi
  else
    testing::phase::add_warning "Neither pnpm nor npm CLI found; cannot verify Node dependencies"
    testing::phase::add_test skipped
    return 0
  fi

  return 1
}

run_go_check || true
run_node_check || true

testing::phase::end_with_summary "Dependency validation completed"
