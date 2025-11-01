#!/bin/bash
# Validates critical files, directories, and formatting for the System Monitor scenario.

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "45s"

cd "$TESTING_PHASE_SCENARIO_DIR"

required_dirs=(api cli ui initialization docs)
if testing::phase::check_directories "${required_dirs[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

required_files=(
  .vrooli/service.json
  PRD.md
  README.md
  test/run-tests.sh
  api/go.mod
  api/main.go
  cli/system-monitor
)
if testing::phase::check_files "${required_files[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

if command -v gofmt >/dev/null 2>&1 && [ -d "api" ]; then
  testing::phase::check "Go formatting clean" bash -c 'cd api && [ -z "$(gofmt -l .)" ]'
else
  testing::phase::add_warning "gofmt not available; skipping Go formatting check"
  testing::phase::add_test skipped
fi

if [ -f "ui/package.json" ]; then
  if command -v npm >/dev/null 2>&1; then
    testing::phase::check "UI lint passes" bash -c 'cd ui && npm run lint --silent --if-present'
  else
    testing::phase::add_warning "npm not available; skipping UI lint"
    testing::phase::add_test skipped
  fi
else
  testing::phase::add_warning "UI package.json missing; skipping lint"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Structure validation completed"
