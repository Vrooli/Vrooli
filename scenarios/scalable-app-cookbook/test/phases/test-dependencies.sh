#!/bin/bash
# Ensures runtime dependencies and tooling are available without modifying state.
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

check_go_dependencies() {
  if [ ! -f "api/go.mod" ]; then
    testing::phase::add_warning "Go module not found; skipping Go dependency checks"
    testing::phase::add_test skipped
    return 0
  fi

  if ! command -v go >/dev/null 2>&1; then
    testing::phase::add_warning "Go toolchain not available"
    testing::phase::add_test skipped
    return 0
  fi

  if ! testing::phase::check "Go module resolves" bash -c 'cd api && go list ./... >/dev/null'; then
    return 1
  fi
}

check_node_dependencies() {
  if [ ! -f "ui/package.json" ]; then
    testing::phase::add_warning "UI package.json not found; skipping Node checks"
    testing::phase::add_test skipped
    return 0
  fi

  if ! command -v npm >/dev/null 2>&1; then
    testing::phase::add_warning "npm CLI not available"
    testing::phase::add_test skipped
    return 0
  fi

  if ! testing::phase::check "npm install --dry-run" bash -c 'cd ui && npm install --dry-run >/dev/null'; then
    return 1
  fi
}

check_tool() {
  local name="$1"
  local command="$2"

  if ! testing::phase::check "${name} available" bash -c "$command >/dev/null 2>&1"; then
    return 1
  fi
}

check_go_dependencies || true
check_node_dependencies || true

check_tool "curl" "command -v curl"
check_tool "jq" "command -v jq"

testing::phase::end_with_summary "Dependency validation completed"
