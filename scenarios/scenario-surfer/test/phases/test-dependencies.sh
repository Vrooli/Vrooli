#!/bin/bash
# Verifies language toolchains, binaries, and client prerequisites

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

testing::phase::check "Scenario Surfer API binary built" test -f "api/scenario-surfer-api"

testing::phase::check "Scenario Surfer CLI installed" test -f "cli/scenario-surfer"

if command -v go >/dev/null 2>&1; then
  testing::phase::check "Go module graph resolves" bash -c 'cd api && go list ./... >/dev/null'
else
  testing::phase::add_warning "Go toolchain not available; skipping Go dependency validation"
  testing::phase::add_test skipped
fi

if [ -f "ui/package.json" ]; then
  if [ -d "ui/node_modules" ]; then
    testing::phase::check "UI dependencies installed" test -d "ui/node_modules"
  else
    testing::phase::add_warning "UI dependencies missing; run npm install from the ui directory"
    testing::phase::add_test skipped
  fi
fi

testing::phase::end_with_summary "Dependency validation completed"
