#!/bin/bash
# Integration smoke checks for core-debugger scenario
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

if [ -x cli/core-debugger ]; then
  testing::phase::check "CLI help command" ./cli/core-debugger help
  testing::phase::check "CLI version command" ./cli/core-debugger version
else
  testing::phase::add_warning "CLI binary missing; skipping CLI integration checks"
  testing::phase::add_test skipped
fi

if command -v jq >/dev/null 2>&1 && [ -f data/workarounds/common.json ]; then
  testing::phase::check "Workaround database loads" jq '.workarounds | length >= 0' data/workarounds/common.json
else
  testing::phase::add_warning "Workaround database or jq unavailable"
  testing::phase::add_test skipped
fi

if [ -f api/main.go ] && command -v go >/dev/null 2>&1; then
  testing::phase::check "API package tests" bash -c 'cd api && go test ./...'
else
  testing::phase::add_warning "Skipping Go tests; toolchain or sources missing"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Integration checks completed"
