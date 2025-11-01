#!/bin/bash
# Verify language runtimes and external tooling required for Secrets Manager
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

ensure_command() {
  local name="$1"
  if ! command -v "$name" >/dev/null 2>&1; then
    testing::phase::add_warning "Required command '$name' not found"
    testing::phase::add_test skipped
    return 1
  fi
  return 0
}

run_go_dependency_check() {
  if [ ! -f "api/go.mod" ]; then
    testing::phase::add_warning "Go module not detected; skipping Go dependency check"
    testing::phase::add_test skipped
    return 0
  fi

  if ensure_command go; then
    testing::phase::check "Go module graph resolves" bash -c 'cd api && go list ./... >/dev/null'
  fi
}

run_node_dependency_check() {
  if [ ! -f "ui/package.json" ]; then
    testing::phase::add_warning "UI package.json not found; skipping Node dependency check"
    testing::phase::add_test skipped
    return 0
  fi

  local pkg_manager="npm"
  if command -v pnpm >/dev/null 2>&1 && grep -q '"packageManager"' ui/package.json; then
    pkg_manager="pnpm"
  elif command -v yarn >/dev/null 2>&1 && [ -f ui/yarn.lock ]; then
    pkg_manager="yarn"
  fi

  if ! command -v "$pkg_manager" >/dev/null 2>&1; then
    testing::phase::add_warning "$pkg_manager CLI missing; cannot verify UI dependencies"
    testing::phase::add_test skipped
    return 0
  fi

  case "$pkg_manager" in
    pnpm)
      testing::phase::check "pnpm install --validate" bash -c 'cd ui && pnpm install --lockfile-only >/dev/null'
      ;;
    yarn)
      testing::phase::check "yarn install --check-files" bash -c 'cd ui && yarn install --check-files --frozen-lockfile >/dev/null'
      ;;
    *)
      testing::phase::check "npm install --dry-run" bash -c 'cd ui && npm install --dry-run >/dev/null'
      ;;
  esac
}

run_cli_prerequisites_check() {
  ensure_command jq || true
  ensure_command curl || true
  if ! command -v psql >/dev/null 2>&1; then
    testing::phase::add_warning "psql not found; database validation tests may be skipped"
    testing::phase::add_test skipped
  fi
}

run_go_dependency_check || true
run_node_dependency_check || true
run_cli_prerequisites_check || true

testing::phase::end_with_summary "Dependency validation completed"
