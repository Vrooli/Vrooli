#!/bin/bash
# Ensures language/runtime dependencies resolve without mutating the workspace.

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"

cd "$TESTING_PHASE_SCENARIO_DIR"

if [ -f "api/go.mod" ]; then
  if command -v go >/dev/null 2>&1; then
    testing::phase::check "Go module graph resolves" bash -c 'cd api && go list ./... >/dev/null'
  else
    testing::phase::add_warning "Go toolchain missing; skipping Go dependency check"
    testing::phase::add_test skipped
  fi
else
  testing::phase::add_warning "Go module not detected; skipping Go dependency check"
  testing::phase::add_test skipped
fi

if [ -f "ui/package.json" ]; then
  if command -v npm >/dev/null 2>&1; then
    testing::phase::check "npm install --dry-run" bash -c 'cd ui && npm install --dry-run >/dev/null'
  else
    testing::phase::add_warning "npm CLI missing; cannot verify Node dependencies"
    testing::phase::add_test skipped
  fi
else
  testing::phase::add_warning "UI package.json not found; skipping Node dependency check"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Dependency validation completed"
