#!/bin/bash
# Performance test phase for retro-game-launcher
# Tests load handling and performance benchmarks

set -euo pipefail

# Get app root (4 levels up from this script)
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize test phase
testing::phase::init --target-time "180s"

# Change to scenario directory
cd "$TESTING_PHASE_SCENARIO_DIR"

testing::phase::log_info "Running performance tests..."

# Run Go benchmark tests
if [[ -d "api" ]]; then
    cd api

    testing::phase::log_info "Running Go performance benchmarks..."

    # Run benchmark tests
    if go list -tags=testing ./... 2>/dev/null | grep -q .; then
        testing::phase::log_info "Executing benchmarks..."

        # Run benchmarks with memory profiling
        if go test -tags=testing -bench=. -benchmem -benchtime=1s -timeout 180s 2>&1 | tee benchmark_results.txt; then
            testing::phase::log_success "Performance benchmarks completed"

            # Display summary
            if [[ -f benchmark_results.txt ]]; then
                testing::phase::log_info "Benchmark Summary:"
                grep -E "Benchmark|PASS|FAIL" benchmark_results.txt | head -20 || true
            fi
        else
            testing::phase::log_warning "Some benchmarks failed or timed out"
        fi

        # Cleanup
        rm -f benchmark_results.txt
    else
        testing::phase::log_warning "No Go performance tests found"
    fi

    cd ..
fi

# Basic load test if API is running
if command -v ab &> /dev/null && command -v curl &> /dev/null; then
    API_PORT="${API_PORT:-8080}"

    if curl -sf "http://localhost:${API_PORT}/health" &> /dev/null; then
        testing::phase::log_info "Running Apache Bench load test..."

        # Simple load test: 100 requests, 10 concurrent
        if ab -n 100 -c 10 -q "http://localhost:${API_PORT}/health" 2>&1 | grep -E "Requests per second|Time per request" || true; then
            testing::phase::log_success "Load test completed"
        fi
    else
        testing::phase::log_info "API not running, skipping load tests"
    fi
else
    testing::phase::log_info "Apache Bench (ab) not available, skipping load tests"
fi

# End test phase with summary
testing::phase::end_with_summary "Performance tests completed"
