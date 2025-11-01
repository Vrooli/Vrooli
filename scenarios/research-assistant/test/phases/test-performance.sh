#!/bin/bash
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"

if command -v go >/dev/null 2>&1 && [ -f "api/main.go" ]; then
  if testing::phase::check "Go API builds" bash -c 'cd api && go build -o /tmp/research-assistant-perf-test .'; then
    rm -f /tmp/research-assistant-perf-test
  else
    rm -f /tmp/research-assistant-perf-test
  fi
else
  testing::phase::add_warning "Skipped Go build performance check (missing Go toolchain)"
  testing::phase::add_test skipped
fi

if [ -f "ui/package.json" ]; then
  if command -v npm >/dev/null 2>&1; then
    testing::phase::check "UI build command" bash -c 'cd ui && npm run build --silent >/dev/null'
  else
    testing::phase::add_warning "npm CLI missing; UI build performance skipped"
    testing::phase::add_test skipped
  fi
fi

testing::phase::end_with_summary "Performance checks completed"
