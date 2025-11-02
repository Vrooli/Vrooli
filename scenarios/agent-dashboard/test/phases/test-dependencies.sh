#!/bin/bash
# Verify language/runtime dependencies resolve without mutating state
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "90s"

run_go_checks() {
  if [ ! -f "api/go.mod" ]; then
    testing::phase::add_warning "Go module not detected; skipping Go dependency validation"
    testing::phase::add_test skipped
    return 0
  fi

  if ! command -v go >/dev/null 2>&1; then
    testing::phase::add_warning "Go toolchain not available; skipping Go dependency validation"
    testing::phase::add_test skipped
    return 0
  fi

  testing::phase::check "Go module graph resolves" bash -c 'cd api && go list ./... >/dev/null'
}

run_node_checks() {
  if [ ! -f "ui/package.json" ]; then
    testing::phase::add_warning "ui/package.json not found; skipping Node dependency validation"
    testing::phase::add_test skipped
    return 0
  fi

  if ! command -v npm >/dev/null 2>&1; then
    testing::phase::add_warning "npm CLI not available; skipping Node dependency validation"
    testing::phase::add_test skipped
    return 0
  fi

  testing::phase::check "npm install --dry-run" bash -c 'cd ui && npm install --dry-run >/dev/null'
}

run_go_checks || true
run_node_checks || true

testing::phase::end_with_summary "Dependency validation completed"
