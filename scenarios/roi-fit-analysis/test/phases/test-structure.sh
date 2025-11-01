#!/bin/bash
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

cd "$TESTING_PHASE_SCENARIO_DIR"

required_dirs=(api ui cli test initialization)
if testing::phase::check_directories "${required_dirs[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

required_files=(
  .vrooli/service.json
  README.md
  Makefile
  test/phases/test-unit.sh
  test/phases/test-integration.sh
  test/phases/test-business.sh
  test/phases/test-performance.sh
)
if testing::phase::check_files "${required_files[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

if command -v jq >/dev/null 2>&1; then
  testing::phase::check "Validate .vrooli/service.json" jq empty .vrooli/service.json
else
  testing::phase::add_warning "jq not available; skipping service.json validation"
  testing::phase::add_test skipped
fi

if command -v go >/dev/null 2>&1; then
  testing::phase::check "Go sources compile" bash -c 'cd api && go build ./...'
  testing::phase::check "Go tests discovered" bash -c 'cd api && find . -name "*_test.go" | grep -q .'
  if command -v rg >/dev/null 2>&1; then
    testing::phase::check "Benchmark suite present" bash -c 'cd api && rg --quiet "func Benchmark"'
  else
    testing::phase::add_warning "rg not available; skipping benchmark scan"
    testing::phase::add_test skipped
  fi
else
  testing::phase::add_warning "Go toolchain not found; skipping build checks"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Structure validation completed"
