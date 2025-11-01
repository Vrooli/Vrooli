#!/bin/bash
# Ensure language runtimes and essential tooling are available without mutating state.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

check_command() {
  local name="$1"
  if command -v "$name" >/dev/null 2>&1; then
    testing::phase::check "Command available: $name" command -v "$name"
  else
    testing::phase::add_warning "$name not found"
    testing::phase::add_test skipped
  fi
}

run_go_dependency_check() {
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

run_cli_dependency_check() {
  if [ ! -x "cli/time-tools" ]; then
    testing::phase::add_warning "CLI binary missing; install via cli/install.sh"
    testing::phase::add_test skipped
    return 0
  fi
  testing::phase::check "CLI core commands available" bash -c 'cli/time-tools --help >/dev/null'
}

check_command curl
check_command jq
check_command bats

run_go_dependency_check || true
run_cli_dependency_check || true

testing::phase::end_with_summary "Dependency validation completed"
