#!/bin/bash
# Ensures language toolchains and package manifests resolve without modifications

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

# Go modules
if [ -f "api/go.mod" ]; then
  if command -v go >/dev/null 2>&1; then
    testing::phase::check "Go module graph resolves" bash -c 'cd api && go list ./... >/dev/null'
  else
    testing::phase::add_warning "Go toolchain unavailable; skipping Go dependency check"
    testing::phase::add_test skipped
  fi
else
  testing::phase::add_warning "api/go.mod not found; skipping Go dependency check"
  testing::phase::add_test skipped
fi

# Node package metadata
if [ -f "ui/package.json" ]; then
  if command -v node >/dev/null 2>&1; then
    testing::phase::check "UI package.json parses" bash -c 'cd ui && node -e "require(\"./package.json\")"'
  else
    testing::phase::add_warning "Node runtime unavailable; skipping UI manifest check"
    testing::phase::add_test skipped
  fi
else
  testing::phase::add_warning "ui/package.json not found; skipping Node dependency check"
  testing::phase::add_test skipped
fi

# CLI script should be executable
if [ -f "cli/retro-game-launcher" ]; then
  testing::phase::check "CLI script is executable" test -x cli/retro-game-launcher
else
  testing::phase::add_error "CLI script missing"
fi

testing::phase::end_with_summary "Dependency validation completed"
