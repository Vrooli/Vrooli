#!/bin/bash
# Dependency validation for visitor-intelligence

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "90s"

run_go_check() {
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

run_node_check() {
  if [ ! -f "ui/package.json" ]; then
    testing::phase::add_warning "UI package.json not found; skipping Node dependency check"
    testing::phase::add_test skipped
    return 0
  fi

  if ! command -v npm >/dev/null 2>&1; then
    testing::phase::add_warning "npm CLI missing; cannot verify Node dependencies"
    testing::phase::add_test skipped
    return 0
  fi

  testing::phase::check "npm install --dry-run" bash -c 'cd ui && npm install --dry-run >/dev/null'
}

run_redis_tool_check() {
  if command -v redis-cli >/dev/null 2>&1; then
    testing::phase::add_test passed
  else
    testing::phase::add_warning "redis-cli not found; Redis smoke checks will rely on runtime connectivity"
    testing::phase::add_test skipped
  fi
}

run_go_check || true
run_node_check || true
run_redis_tool_check || true

testing::phase::end_with_summary "Dependency validation completed"
