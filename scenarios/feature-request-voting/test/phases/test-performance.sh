#!/bin/bash
#
# Performance Test Phase for feature-request-voting
# Tests system performance under load
#

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s"

cd "$TESTING_PHASE_SCENARIO_DIR"

testing::phase::announce "Running performance tests"

if [ -d "api" ]; then
    testing::phase::step "Running Go performance tests"
    cd api

    # Run performance tests
    if go test -tags=testing -v -run TestPerformance -timeout 3m 2>&1 | tee /tmp/performance-test.log; then
        testing::phase::success "Performance tests passed"
    else
        testing::phase::error "Performance tests failed"
    fi

    # Run benchmarks
    testing::phase::step "Running benchmarks"
    if go test -tags=testing -bench=. -benchmem -run=^$ 2>&1 | tee /tmp/benchmark.log; then
        testing::phase::success "Benchmarks completed"
    else
        testing::phase::warning "Benchmarks failed (non-critical)"
    fi

    cd ..
fi

testing::phase::end_with_summary "Performance tests completed"
