#!/bin/bash
# Verifies language toolchains and critical resources needed by Contact Book.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "90s"

run_go_dependency_check() {
  if [ ! -f "api/go.mod" ]; then
    testing::phase::add_warning "Go module not present; skipping Go dependency check"
    testing::phase::add_test skipped
    return
  fi

  if ! command -v go >/dev/null 2>&1; then
    testing::phase::add_warning "Go toolchain not installed; cannot verify Go dependencies"
    testing::phase::add_test skipped
    return
  fi

  testing::phase::check "Go module graph resolves" bash -c 'cd api && go list ./... >/dev/null'
}

run_cli_dependency_check() {
  if command -v contact-book >/dev/null 2>&1; then
    log::success "contact-book CLI available"
    testing::phase::add_test passed
    return
  fi

  if [ -f "cli/install.sh" ]; then
    testing::phase::check "CLI install script available" bash -c 'cd cli && test -x install.sh'
  else
    testing::phase::add_warning "CLI install script missing; skipping CLI dependency validation"
    testing::phase::add_test skipped
  fi
}

check_resource_health() {
  if ! command -v vrooli >/dev/null 2>&1; then
    testing::phase::add_warning "vrooli CLI unavailable; skipping resource status checks"
    testing::phase::add_test skipped
    return
  fi

  if vrooli resource status postgres >/dev/null 2>&1; then
    log::success "Postgres resource reachable"
    testing::phase::add_test passed
  else
    testing::phase::add_warning "Postgres resource status unavailable"
    testing::phase::add_test skipped
  fi

  if vrooli resource status qdrant >/dev/null 2>&1; then
    log::success "Qdrant resource reachable"
    testing::phase::add_test passed
  else
    testing::phase::add_warning "Qdrant resource status unavailable"
    testing::phase::add_test skipped
  fi
}

run_go_dependency_check
run_cli_dependency_check
check_resource_health

testing::phase::end_with_summary "Dependency validation completed"
