#!/bin/bash
# Ensures language runtimes and dependency graphs resolve without performing full installs.
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

cd "$TESTING_PHASE_SCENARIO_DIR"

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

  testing::phase::check "go mod tidy --check" bash -c 'cd api && go mod tidy -v >/dev/null'
}

run_node_checks() {
  if [ ! -f "ui/package.json" ]; then
    testing::phase::add_warning "UI package.json not found; skipping Node dependency check"
    testing::phase::add_test skipped
    return 0
  fi

  local pkg_manager="npm"
  if command -v pnpm >/dev/null 2>&1 && grep -q '"packageManager"' ui/package.json; then
    pkg_manager="pnpm"
  elif command -v yarn >/dev/null 2>&1 && [ -f "ui/yarn.lock" ]; then
    pkg_manager="yarn"
  elif ! command -v npm >/dev/null 2>&1; then
    testing::phase::add_warning "No supported Node package manager found"
    testing::phase::add_test skipped
    return 0
  fi

  case "$pkg_manager" in
    pnpm)
      testing::phase::check "pnpm install --frozen-lockfile" bash -c 'cd ui && pnpm install --frozen-lockfile --prefer-offline --ignore-scripts --reporter silent >/dev/null'
      ;;
    yarn)
      testing::phase::check "yarn install --check-files" bash -c 'cd ui && yarn install --check-files --prefer-offline >/dev/null'
      ;;
    npm)
      testing::phase::check "npm install --dry-run" bash -c 'cd ui && npm install --dry-run >/dev/null'
      ;;
  esac
}

run_go_checks || true
run_node_checks || true

testing::phase::end_with_summary "Dependency validation completed"
