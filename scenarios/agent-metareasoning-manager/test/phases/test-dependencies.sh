#!/bin/bash
# Ensures language runtimes and key manifests resolve without modification.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

run_go_module_check() {
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

  if testing::phase::check "Go module graph resolves" bash -c 'cd api && go list ./... >/dev/null'; then
    return 0
  fi

  return 1
}

run_makefile_check() {
  if [ ! -f "Makefile" ]; then
    testing::phase::add_error "Makefile missing"
    testing::phase::add_test failed
    return 1
  fi

  if testing::phase::check "Makefile exposes run target" bash -c 'grep -q "^run:" Makefile'; then
    return 0
  fi

  return 1
}

run_go_module_check || true
run_makefile_check || true

testing::phase::end_with_summary "Dependency validation completed"
