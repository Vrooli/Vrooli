#!/bin/bash
# Verify language/runtime dependencies resolve without mutating workspace state.

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "90s"

run_go_dependency_check() {
  if [ ! -f "api/go.mod" ]; then
    testing::phase::add_warning "Go module not detected; skipping Go dependency check"
    testing::phase::add_test skipped
    return 0
  fi

  if ! command -v go >/dev/null 2>&1; then
    testing::phase::add_warning "Go toolchain not available; skipping Go dependency validation"
    testing::phase::add_test skipped
    return 0
  fi

  if testing::phase::check "Go module graph resolves" bash -c 'cd api && go list ./... >/dev/null'; then
    return 0
  fi
  return 1
}

run_node_dependency_check() {
  if [ ! -f "ui/package.json" ]; then
    testing::phase::add_warning "UI package.json not found; skipping Node dependency check"
    testing::phase::add_test skipped
    return 0
  fi

  # Detect package manager preference.
  local manager="npm"
  if [ -f "ui/pnpm-lock.yaml" ] || grep -q '"packageManager"\s*:\s*"pnpm@' ui/package.json 2>/dev/null; then
    manager="pnpm"
  elif [ -f "ui/yarn.lock" ]; then
    manager="yarn"
  fi

  case "$manager" in
    pnpm)
      if ! command -v pnpm >/dev/null 2>&1; then
        testing::phase::add_warning "pnpm not available; skipping UI dependency check"
        testing::phase::add_test skipped
        return 0
      fi
      testing::phase::check "pnpm install --dry-run" bash -c 'cd ui && pnpm install --lockfile-only --ignore-scripts --dry-run --reporter=silent'
      ;;
    yarn)
      if ! command -v yarn >/dev/null 2>&1; then
        testing::phase::add_warning "yarn not available; skipping UI dependency check"
        testing::phase::add_test skipped
        return 0
      fi
      testing::phase::check "yarn install --dry-run" bash -c 'cd ui && yarn install --check-files --dry-run >/dev/null'
      ;;
    *)
      if ! command -v npm >/dev/null 2>&1; then
        testing::phase::add_warning "npm not available; skipping UI dependency check"
        testing::phase::add_test skipped
        return 0
      fi
      testing::phase::check "npm install --dry-run" bash -c 'cd ui && npm install --ignore-scripts --no-audit --no-fund --dry-run >/dev/null'
      ;;
  esac
}

run_go_dependency_check || true
run_node_dependency_check || true

testing::phase::end_with_summary "Dependency validation completed"
