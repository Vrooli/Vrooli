#!/bin/bash
# Dependency validation for audio-intelligence-platform
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

go_check() {
  if [ ! -f "api/go.mod" ]; then
    testing::phase::add_warning "Go module not found; skipping Go dependency check"
    testing::phase::add_test skipped
    return 0
  fi

  if ! command -v go >/dev/null 2>&1; then
    testing::phase::add_warning "Go toolchain missing; cannot verify Go dependencies"
    testing::phase::add_test skipped
    return 0
  fi

  if testing::phase::check "go list ./..." bash -c 'cd api && go list ./... >/dev/null'; then
    return 0
  fi

  return 1
}

node_check() {
  if [ ! -f "ui/package.json" ]; then
    testing::phase::add_warning "UI package.json missing; skipping Node dependency check"
    testing::phase::add_test skipped
    return 0
  fi

  local pkg_manager="npm"
  if [ -f "ui/pnpm-lock.yaml" ]; then
    pkg_manager="pnpm"
  elif [ -f "ui/yarn.lock" ]; then
    pkg_manager="yarn"
  fi

  case "$pkg_manager" in
    pnpm)
      if command -v pnpm >/dev/null 2>&1; then
        if testing::phase::check "pnpm install --lockfile-only" bash -c 'cd ui && pnpm install --lockfile-only >/dev/null'; then
          return 0
        fi
      else
        testing::phase::add_warning "pnpm not available; skipping Node dependency verification"
        testing::phase::add_test skipped
        return 0
      fi
      ;;
    yarn)
      if command -v yarn >/dev/null 2>&1; then
        if testing::phase::check "yarn install --mode=skip-build" bash -c 'cd ui && yarn install --mode=skip-build >/dev/null'; then
          return 0
        fi
      else
        testing::phase::add_warning "yarn not available; skipping Node dependency verification"
        testing::phase::add_test skipped
        return 0
      fi
      ;;
    npm|*)
      if command -v npm >/dev/null 2>&1; then
        if testing::phase::check "npm install --dry-run" bash -c 'cd ui && npm install --dry-run >/dev/null'; then
          return 0
        fi
      else
        testing::phase::add_warning "npm not available; skipping Node dependency verification"
        testing::phase::add_test skipped
        return 0
      fi
      ;;
  esac

  return 1
}

# Execute checks; non-zero exits already recorded via add_test
go_check || true
node_check || true

testing::phase::end_with_summary "Dependency validation completed"
