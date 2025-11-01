#!/bin/bash
# Placeholder performance gate to ensure benchmarks are acknowledged.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

if [ -f "api/benchmarks_test.go" ] || grep -q "//go:build benchmark" -R api 2>/dev/null; then
  if command -v go >/dev/null 2>&1; then
    testing::phase::check "Go benchmarks smoke" bash -c "cd '$TESTING_PHASE_SCENARIO_DIR/api' && go test -run=^$ -bench=. -benchtime=100x >/dev/null"
  else
    testing::phase::add_warning "Go toolchain missing; skipping benchmarks"
    testing::phase::add_test skipped
  fi
else
  testing::phase::add_warning "No dedicated benchmarks defined; implement go test -bench cases for core operations"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Performance phase completed"
