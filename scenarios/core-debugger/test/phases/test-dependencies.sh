#!/bin/bash
# Dependency validation for core-debugger scenario
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "45s"

if command -v go >/dev/null 2>&1; then
  testing::phase::check "Go modules resolve" bash -c 'cd api && go list ./... >/dev/null'
else
  testing::phase::add_warning "Go toolchain not available; skipping module resolution"
  testing::phase::add_test skipped
fi

if [ -x cli/core-debugger ]; then
  testing::phase::check "CLI dependencies satisfied" ./cli/core-debugger --help
else
  testing::phase::add_warning "CLI binary missing; skipping CLI dependency check"
  testing::phase::add_test skipped
fi

if [ -f data/components.json ] && command -v jq >/dev/null 2>&1; then
  testing::phase::check "Component registry parses" jq '.components | length >= 1' data/components.json
else
  testing::phase::add_warning "Component registry not available or jq missing"
  testing::phase::add_test skipped
fi

if [ -f scripts/health-monitor.sh ]; then
  testing::phase::check "Health monitor script lint" bash -c 'bash -n scripts/health-monitor.sh'
else
  testing::phase::add_warning "Health monitor script missing"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Dependency validation completed"
