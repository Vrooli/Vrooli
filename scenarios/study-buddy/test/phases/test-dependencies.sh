#!/bin/bash
# Ensures the language and tooling dependencies resolve without performing destructive installs.

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

run_go_checks() {
  if [ ! -f "api/go.mod" ]; then
    testing::phase::add_warning "Go module not detected; skipping Go dependency check"
    testing::phase::add_test skipped
    return 0
  fi

  if ! command -v go >/dev/null 2>&1; then
    testing::phase::add_warning "Go toolchain missing; skipping Go dependency check"
    testing::phase::add_test skipped
    return 0
  fi

  if testing::phase::check "Go module graph resolves" bash -c 'cd api && go list ./... >/dev/null'; then
    return 0
  fi

  return 1
}

run_node_checks() {
  if [ ! -f "ui/package.json" ]; then
    testing::phase::add_warning "UI package.json not found; skipping Node dependency check"
    testing::phase::add_test skipped
    return 0
  fi

  if ! command -v npm >/dev/null 2>&1; then
    testing::phase::add_warning "npm CLI unavailable; skipping Node dependency check"
    testing::phase::add_test skipped
    return 0
  fi

  local install_cmd='cd ui && npm install --ignore-scripts --no-audit --dry-run >/dev/null'
  if testing::phase::check "npm install --dry-run" bash -c "$install_cmd"; then
    return 0
  fi

  return 1
}

run_cli_checks() {
  if command -v study-buddy >/dev/null 2>&1; then
    testing::phase::check "study-buddy CLI available" bash -c 'command -v study-buddy >/dev/null'
    return 0
  fi

  testing::phase::add_warning "CLI binary not on PATH; install step may be required"
  testing::phase::add_test skipped
  return 0
}

run_tooling_checks() {
  if command -v jq >/dev/null 2>&1; then
    testing::phase::check "jq available" bash -c 'command -v jq >/dev/null'
    return 0
  fi

  testing::phase::add_warning "jq not installed; JSON assertions will be downgraded"
  testing::phase::add_test skipped
  return 0
}

run_go_checks || true
run_node_checks || true
run_cli_checks || true
run_tooling_checks || true

testing::phase::end_with_summary "Dependency validation completed"
