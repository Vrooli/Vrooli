#!/bin/bash
# Track performance baselines via Go benchmarks and stress tests
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"

ARTIFACT_DIR="${TESTING_PHASE_SCENARIO_DIR}/artifacts"
mkdir -p "$ARTIFACT_DIR"

if [ -f "api/performance_comprehensive_test.go" ]; then
  testing::phase::check "Go benchmark sweep" \
    bash -c 'cd api && go test -timeout 120s -bench=. -benchmem ./... > ../artifacts/performance-bench.txt'
else
  testing::phase::add_warning "No dedicated performance test file; skipping benchmarks"
  testing::phase::add_test skipped
fi

if [ -f "api/main_test.go" ]; then
  testing::phase::check "Performance smoke tests" \
    bash -c 'cd api && go test -timeout 60s -run "Test(Performance|Concurrency)"'
else
  testing::phase::add_warning "main_test.go missing; skipping performance smoke tests"
  testing::phase::add_test skipped
fi

if [ -f "artifacts/performance-bench.txt" ]; then
  echo "Top benchmark samples:"
  grep '^Benchmark' artifacts/performance-bench.txt | head -10 || true
fi

testing::phase::end_with_summary "Performance phase completed"
