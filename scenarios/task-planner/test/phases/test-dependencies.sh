#!/bin/bash
# Ensure primary language dependencies resolve without performing full installs

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

check_go_deps() {
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

  if testing::phase::check "Go module graph resolves" bash -c 'cd api && go list ./... >/dev/null'; then
    return 0
  fi

  return 1
}

check_node_deps() {
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
  fi

  local cmd
  case "$pkg_manager" in
    pnpm)
      cmd="pnpm install --lockfile-only"
      ;;
    yarn)
      cmd="yarn install --mode=skip-build --ignore-scripts"
      ;;
    *)
      cmd="npm install --dry-run"
      ;;
  esac

  if command -v ${pkg_manager} >/dev/null 2>&1; then
    if testing::phase::check "${pkg_manager} dependency resolution" bash -c "cd ui && ${cmd} >/dev/null"; then
      return 0
    fi
  else
    testing::phase::add_warning "${pkg_manager} CLI missing; skipping Node dependency check"
    testing::phase::add_test skipped
    return 0
  fi

  return 1
}

check_go_deps || true
check_node_deps || true

testing::phase::end_with_summary "Dependency validation completed"
