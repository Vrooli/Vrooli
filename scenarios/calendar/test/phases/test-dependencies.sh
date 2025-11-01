#!/bin/bash
# Ensure language dependencies resolve without mutating the workspace.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"

run_go_checks() {
  if [ ! -f "api/go.mod" ]; then
    testing::phase::add_warning "Go module not detected; skipping Go dependency check"
    testing::phase::add_test skipped
    return 0
  fi

  if ! command -v go >/dev/null 2>&1; then
    testing::phase::add_warning "Go toolchain missing; cannot verify Go dependencies"
    testing::phase::add_test skipped
    return 0
  fi

  testing::phase::check "Go module graph resolves" bash -c 'cd api && go list ./... >/dev/null'
}

run_node_checks() {
  if [ ! -f "ui/package.json" ]; then
    testing::phase::add_warning "UI package.json not found; skipping Node dependency check"
    testing::phase::add_test skipped
    return 0
  fi

  local manager="npm"
  if grep -q '"packageManager"\s*:\s*"pnpm@' ui/package.json 2>/dev/null; then
    manager="pnpm"
  fi

  case "$manager" in
    pnpm)
      if command -v pnpm >/dev/null 2>&1; then
        testing::phase::check "pnpm install --dry-run" bash -c 'cd ui && pnpm install --dry-run >/dev/null'
      else
        testing::phase::add_warning "pnpm not installed; skipping Node dependency check"
        testing::phase::add_test skipped
      fi
      ;;
    *)
      if command -v npm >/dev/null 2>&1; then
        testing::phase::check "npm install --dry-run" bash -c 'cd ui && npm install --dry-run >/dev/null'
      else
        testing::phase::add_warning "npm CLI missing; cannot verify Node dependencies"
        testing::phase::add_test skipped
      fi
      ;;
  esac
}

run_go_checks || true
run_node_checks || true

testing::phase::end_with_summary "Dependency validation completed"
