#!/bin/bash
# Performance tests for scenario-to-extension

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"

cd "$TESTING_PHASE_SCENARIO_DIR/api" || { echo "Failed to cd to api directory"; exit 1; }

testing::phase::info "Starting performance tests for scenario-to-extension"

# Run performance tests
testing::phase::step "Running performance test suite"
if ! go test -v -run "TestPerformance" -timeout 120s 2>&1 | tee /tmp/performance-test-output.log; then
    testing::phase::error "Performance tests failed"
    testing::phase::end_with_summary "Performance tests failed"
    exit 1
fi
testing::phase::success "Performance tests passed"

# Run benchmarks
testing::phase::step "Running benchmark tests"
if ! go test -bench=. -benchtime=1s -timeout 120s &>/tmp/benchmark-output.log; then
    testing::phase::warn "Benchmark tests failed (non-critical)"
else
    testing::phase::success "Benchmark tests completed"
    testing::phase::info "Benchmark results:"
    grep "Benchmark" /tmp/benchmark-output.log | while read -r line; do
        testing::phase::info "  $line"
    done
fi

testing::phase::end_with_summary "Performance tests completed successfully"
