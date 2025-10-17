#!/bin/bash
# Performance tests for test-scenario
set -euo pipefail

# Initialize phase environment
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "Running performance tests..."

# Test 1: Build time performance
log::info "Testing build performance"
build_start=$(date +%s%3N)
cd api && go build -o test-scenario-perf . && cd ..
build_end=$(date +%s%3N)
build_time=$((build_end - build_start))

if [ "$build_time" -lt 5000 ]; then
    log::success "Build completed in ${build_time}ms (excellent)"
elif [ "$build_time" -lt 10000 ]; then
    log::success "Build completed in ${build_time}ms (good)"
else
    log::warning "Build took ${build_time}ms (consider optimization)"
    ((TESTING_PHASE_WARNING_COUNT++))
fi

# Clean up build artifact
rm -f api/test-scenario-perf

# Test 2: Test execution time
log::info "Testing unit test execution time"
test_start=$(date +%s%3N)
cd api && go test -run TestHelperFunctions -v &> /dev/null || true
test_end=$(date +%s%3N)
test_time=$((test_end - test_start))

if [ "$test_time" -lt 1000 ]; then
    log::success "Tests executed in ${test_time}ms (excellent)"
elif [ "$test_time" -lt 3000 ]; then
    log::success "Tests executed in ${test_time}ms (good)"
else
    log::warning "Tests took ${test_time}ms (consider optimization)"
    ((TESTING_PHASE_WARNING_COUNT++))
fi
cd ..

# Test 3: Memory footprint of test utilities
log::info "Checking test file sizes"
test_helpers_size=$(wc -c < api/test_helpers.go 2>/dev/null || echo "0")
test_patterns_size=$(wc -c < api/test_patterns.go 2>/dev/null || echo "0")
main_test_size=$(wc -c < api/main_test.go 2>/dev/null || echo "0")

total_test_size=$((test_helpers_size + test_patterns_size + main_test_size))
total_kb=$((total_test_size / 1024))

if [ "$total_kb" -lt 50 ]; then
    log::success "Test code is ${total_kb}KB (compact)"
elif [ "$total_kb" -lt 100 ]; then
    log::success "Test code is ${total_kb}KB (reasonable)"
else
    log::info "Test code is ${total_kb}KB"
fi

# Test 4: Line count analysis
log::info "Analyzing test coverage metrics"
if command -v go &> /dev/null; then
    cd api
    total_lines=$(cat *.go | wc -l 2>/dev/null || echo "0")
    test_lines=$(cat *_test.go test_*.go 2>/dev/null | wc -l || echo "0")
    cd ..

    if [ "$total_lines" -gt 0 ]; then
        test_ratio=$((test_lines * 100 / total_lines))
        log::info "Test code ratio: ${test_ratio}% (${test_lines}/${total_lines} lines)"
    fi
fi

# Test 5: Concurrency performance (from Go tests)
log::info "Running concurrency performance tests"
cd api
test_output=$(go test -run TestConcurrentRequests 2>&1) || true
if echo "$test_output" | grep -q "PASS"; then
    log::success "Concurrency tests passed"
else
    log::warning "Concurrency tests had issues"
    ((TESTING_PHASE_WARNING_COUNT++))
fi
cd ..

testing::phase::end_with_summary "Performance tests completed"
