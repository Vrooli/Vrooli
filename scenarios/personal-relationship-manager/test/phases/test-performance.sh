#!/bin/bash
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase
testing::phase::init --target-time "90s"

echo "🧪 Running performance tests..."

cd "$TESTING_PHASE_SCENARIO_DIR/api" || exit 1

# Run Go performance tests
if command -v go >/dev/null 2>&1; then
    log::info "Running Go performance tests..."

    # Run performance benchmarks
    if go test -v -tags=testing -run "TestHandlerPerformance" -timeout 60s 2>&1 | tee /tmp/performance-test-output.txt; then
        log::success "✅ Performance tests completed"

        # Extract performance metrics
        if grep -q "GetContacts:" /tmp/performance-test-output.txt; then
            perf_line=$(grep "GetContacts:" /tmp/performance-test-output.txt)
            log::info "📊 $perf_line"
        fi

        if grep -q "CreateContact:" /tmp/performance-test-output.txt; then
            perf_line=$(grep "CreateContact:" /tmp/performance-test-output.txt)
            log::info "📊 $perf_line"
        fi

        # Check for warnings
        if grep -q "Warning:" /tmp/performance-test-output.txt; then
            warning_count=$(grep -c "Warning:" /tmp/performance-test-output.txt || echo 0)
            testing::phase::add_warning "⚠️  Performance warnings found: $warning_count"
        fi
    else
        testing::phase::add_error "❌ Performance tests failed"
    fi

    # Run benchmarks if available
    log::info "Running Go benchmarks..."
    if go test -bench=. -benchmem -tags=testing -run=^$ -timeout 60s 2>&1 | tee /tmp/benchmark-output.txt; then
        if grep -q "Benchmark" /tmp/benchmark-output.txt; then
            log::success "✅ Benchmarks completed"
            grep "Benchmark" /tmp/benchmark-output.txt | head -10
        else
            log::info "ℹ️  No benchmarks defined"
        fi
    else
        log::info "ℹ️  Benchmark execution completed with warnings"
    fi
else
    testing::phase::add_error "❌ Go toolchain not available for performance tests"
fi

# End with summary
testing::phase::end_with_summary "Performance tests completed"
