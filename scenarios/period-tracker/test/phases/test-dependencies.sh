#!/bin/bash
# Ensures language runtimes and package manifests resolve without installs
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

go_mod_dir="api"
if [ -f "$go_mod_dir/go.mod" ]; then
  if command -v go >/dev/null 2>&1; then
    if testing::phase::check "Go modules resolve" bash -c "cd '$go_mod_dir' && go list ./... >/dev/null"; then
      :
    fi
  else
    testing::phase::add_warning "Go toolchain not available; skipping Go dependency check"
    testing::phase::add_test skipped
  fi
else
  testing::phase::add_warning "Go module not detected; skipping Go dependency check"
  testing::phase::add_test skipped
fi

ui_dir="ui"
if [ -f "$ui_dir/package.json" ]; then
  if command -v npm >/dev/null 2>&1; then
    if testing::phase::check "npm dependency dry-run" bash -c "cd '$ui_dir' && npm install --package-lock-only --ignore-scripts --dry-run >/dev/null"; then
      :
    fi
  else
    testing::phase::add_warning "npm CLI not available; skipping Node dependency check"
    testing::phase::add_test skipped
  fi
else
  testing::phase::add_warning "UI package.json not found; skipping Node dependency check"
  testing::phase::add_test skipped
fi

if command -v redis-cli >/dev/null 2>&1; then
  testing::phase::check "Redis CLI available" redis-cli --version
else
  testing::phase::add_warning "redis-cli not available; optional cache checks skipped"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Dependency validation completed"
