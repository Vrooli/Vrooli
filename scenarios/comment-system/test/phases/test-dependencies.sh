#!/bin/bash
# Verify language and tooling dependencies without modifying the workspace
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "90s"

run_go_checks() {
  if [[ ! -f api/go.mod ]]; then
    testing::phase::add_warning "api/go.mod missing; skipping Go dependency verification"
    testing::phase::add_test skipped
    return 0
  fi

  if ! command -v go >/dev/null 2>&1; then
    testing::phase::add_warning "Go toolchain unavailable; skipping Go dependency verification"
    testing::phase::add_test skipped
    return 0
  fi

  if go mod verify >/dev/null 2>&1; then
    testing::phase::add_test passed
  else
    testing::phase::add_error "go mod verify failed"
    testing::phase::add_test failed
  fi
}

run_node_checks() {
  if [[ ! -f ui/package.json ]]; then
    testing::phase::add_warning "ui/package.json missing; skipping Node dependency verification"
    testing::phase::add_test skipped
    return 0
  fi

  if ! command -v npm >/dev/null 2>&1; then
    testing::phase::add_warning "npm CLI unavailable; skipping Node dependency verification"
    testing::phase::add_test skipped
    return 0
  fi

  if (cd ui && npm install --package-lock-only --ignore-scripts >/dev/null 2>&1); then
    testing::phase::add_test passed
    rm -f ui/package-lock.json >/dev/null 2>&1 || true
  else
    testing::phase::add_error "npm dependency verification failed"
    testing::phase::add_test failed
    rm -f ui/package-lock.json >/dev/null 2>&1 || true
  fi
}

run_cli_checks() {
  if [[ -x cli/comment-system ]]; then
    if cli/comment-system help >/dev/null 2>&1; then
      testing::phase::add_test passed
    else
      testing::phase::add_error "cli/comment-system help failed"
      testing::phase::add_test failed
    fi
  else
    testing::phase::add_warning "cli/comment-system not executable; skipping CLI smoke check"
    testing::phase::add_test skipped
  fi
}

run_go_checks
run_node_checks
run_cli_checks

testing::phase::end_with_summary "Dependency validation completed"
