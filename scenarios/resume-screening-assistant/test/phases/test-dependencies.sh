#!/bin/bash
# Verify language toolchains, package manifests, and CLI tests.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

set -euo pipefail

testing::phase::init --target-time "120s"

check_command() {
  local name="$1"
  if command -v "$name" >/dev/null 2>&1; then
    return 0
  fi
  testing::phase::add_warning "$name not available; related checks skipped"
  testing::phase::add_test skipped
  return 1
}

run_go_dependency_check() {
  if [ ! -f "api/go.mod" ]; then
    testing::phase::add_warning "Go module not found; skipping go list"
    testing::phase::add_test skipped
    return 0
  fi

  if check_command go; then
    if testing::phase::check "Go module graph resolves" bash -c 'cd api && go list ./... >/dev/null'; then
      return 0
    fi
    return 1
  fi
  return 0
}

run_ui_dependency_check() {
  if [ ! -f "ui/package.json" ]; then
    testing::phase::add_warning "UI package.json missing; skipping Node dependency check"
    testing::phase::add_test skipped
    return 0
  fi

  if check_command npm; then
    if testing::phase::check "npm install --dry-run" bash -c 'cd ui && npm install --dry-run >/dev/null'; then
      return 0
    fi
    return 1
  fi
  return 0
}

check_required_tools() {
  local tools=(curl jq)
  for tool in "${tools[@]}"; do
    if check_command "$tool"; then
      testing::phase::add_test passed
    fi
  done
}

run_cli_bats() {
  if ! [ -f "cli/resume-screening-assistant.bats" ]; then
    testing::phase::add_warning "CLI BATS suite missing; skipping"
    testing::phase::add_test skipped
    return 0
  fi

  if check_command bats; then
    testing::phase::check "CLI BATS regression suite" bats cli/resume-screening-assistant.bats
  fi
}

run_go_dependency_check || true
run_ui_dependency_check || true
check_required_tools
run_cli_bats || true

testing::phase::end_with_summary "Dependency validation completed"
