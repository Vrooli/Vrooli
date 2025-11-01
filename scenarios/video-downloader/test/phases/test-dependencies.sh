#!/bin/bash
# Dependency verification phase

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

run_go_check() {
  if [ ! -f "api/go.mod" ]; then
    testing::phase::add_warning "Go module not detected; skipping Go dependency check"
    testing::phase::add_test skipped
    return
  fi

  if ! command -v go >/dev/null 2>&1; then
    testing::phase::add_warning "Go toolchain missing; cannot verify Go dependencies"
    testing::phase::add_test skipped
    return
  fi

  if testing::phase::check "Go module graph resolves" bash -c 'cd api && go list ./... >/dev/null'; then
    return
  fi
}

run_node_check() {
  if [ ! -f "ui/package.json" ]; then
    testing::phase::add_warning "UI package.json not found; skipping Node dependency check"
    testing::phase::add_test skipped
    return
  fi

  if ! command -v npm >/dev/null 2>&1; then
    testing::phase::add_warning "npm CLI missing; cannot verify Node dependencies"
    testing::phase::add_test skipped
    return
  fi

  if testing::phase::check "npm install --dry-run" bash -c 'cd ui && npm install --dry-run --silent >/dev/null'; then
    return
  fi
}

run_go_check || true
run_node_check || true

testing::phase::end_with_summary "Dependency validation completed"
