#!/usr/bin/env bash
# Performance Regression Testing
# Measures and compares test execution performance

set -euo pipefail

# Script directory
RUNNER_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_ROOT="$(dirname "$RUNNER_DIR")"

# Source var.sh for directory variables
# shellcheck disable=SC1091
source "$RUNNER_DIR/../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Colors for output
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly RED='\033[0;31m'
readonly NC='\033[0m'

echo -e "${YELLOW}🔬 Performance Regression Testing${NC}"
echo "========================================="

# Performance metrics storage
declare -A PERF_METRICS

#######################################
# Run performance test with timing
#######################################
run_timed_test() {
    local test_name="$1"
    local test_command="$2"
    
    echo -e "\n${YELLOW}Testing: $test_name${NC}"
    
    # Run test and capture time
    local start_time end_time duration
    start_time=$(date +%s%N)
    
    if eval "$test_command" >&/dev/null; then
        end_time=$(date +%s%N)
        duration=$((($end_time - $start_time) / 1000000))  # Convert to milliseconds
        PERF_METRICS["$test_name"]=$duration
        echo -e "  Duration: ${GREEN}${duration}ms${NC}"
        return 0
    else
        echo -e "  Status: ${RED}FAILED${NC}"
        return 1
    fi
}

#######################################
# Test setup/teardown overhead
#######################################
test_setup_overhead() {
    echo -e "\n${YELLOW}=== Setup/Teardown Overhead ===${NC}"
    
    # Test minimal setup
    run_timed_test "Minimal BATS test" \
        "bats $TEST_ROOT/fixtures/tests/core/minimal-test.bats"
    
    # Test with full setup
    run_timed_test "Test with full setup" \
        "bats $TEST_ROOT/fixtures/tests/core/simple-test.bats"
}

#######################################
# Test parallel execution performance
#######################################
test_parallel_performance() {
    echo -e "\n${YELLOW}=== Parallel Execution Performance ===${NC}"
    
    # Sequential execution
    run_timed_test "Sequential (all unit tests)" \
        "$TEST_ROOT/runners/run-unit.sh"
    
    # Parallel execution
    run_timed_test "Parallel (4 jobs)" \
        "$TEST_ROOT/runners/run-unit.sh --parallel --jobs 4"
    
    # Parallel execution with more jobs
    run_timed_test "Parallel (8 jobs)" \
        "$TEST_ROOT/runners/run-unit.sh --parallel --jobs 8"
}

#######################################
# Test configuration loading speed
#######################################
test_config_performance() {
    echo -e "\n${YELLOW}=== Configuration Loading ===${NC}"
    
    # Create test script for config loading
    cat > /tmp/config_perf_test.sh << EOF
#!/usr/bin/env bash
export VROOLI_TEST_ROOT="$TEST_ROOT"
source "\$VROOLI_TEST_ROOT/shared/config-simple.bash"
vrooli_config_init
vrooli_config_get "services.ollama.port" >/dev/null
EOF
    chmod +x /tmp/config_perf_test.sh
    
    run_timed_test "Config loading and lookup" \
        "/tmp/config_perf_test.sh"
    
    trash::safe_remove /tmp/config_perf_test.sh --test-cleanup
}

#######################################
# Test assertion performance
#######################################
test_assertion_performance() {
    echo -e "\n${YELLOW}=== Assertion Performance ===${NC}"
    
    # Create assertion performance test
    cat > /tmp/assertion_perf.bats << 'EOF'
#!/usr/bin/env bats

source "${VROOLI_TEST_ROOT:-/home/matthalloran8/Vrooli/scripts/__test}/fixtures/setup.bash"

@test "100 assertions" {
    for i in {1..100}; do
        assert_equals "test" "test"
    done
}
EOF
    
    run_timed_test "100 assertions" \
        "VROOLI_TEST_ROOT=$TEST_ROOT bats /tmp/assertion_perf.bats"
    
    trash::safe_remove /tmp/assertion_perf.bats --test-cleanup
}

#######################################
# Test cleanup performance
#######################################
test_cleanup_performance() {
    echo -e "\n${YELLOW}=== Cleanup Performance ===${NC}"
    
    # Create cleanup test
    cat > /tmp/cleanup_perf.bats << 'EOF'
#!/usr/bin/env bats

source "${VROOLI_TEST_ROOT:-/home/matthalloran8/Vrooli/scripts/__test}/fixtures/setup.bash"

setup() {
    vrooli_setup_unit_test
    # Create resources to clean up
    for i in {1..10}; do
        touch "$VROOLI_TEST_TMPDIR/file_$i"
    done
}

@test "cleanup performance" {
    assert_file_exists "$VROOLI_TEST_TMPDIR/file_1"
}
EOF
    
    run_timed_test "Cleanup of 10 files" \
        "VROOLI_TEST_ROOT=$TEST_ROOT bats /tmp/cleanup_perf.bats"
    
    trash::safe_remove /tmp/cleanup_perf.bats --test-cleanup
}

#######################################
# Generate performance report
#######################################
generate_report() {
    echo -e "\n${YELLOW}=========================================${NC}"
    echo -e "${GREEN}📊 Performance Report${NC}"
    echo "========================================="
    
    # Calculate statistics
    local total_time=0
    local min_time=999999
    local max_time=0
    local test_count=0
    
    for test_name in "${!PERF_METRICS[@]}"; do
        local duration="${PERF_METRICS[$test_name]}"
        total_time=$((total_time + duration))
        test_count=$((test_count + 1))
        
        if [[ $duration -lt $min_time ]]; then
            min_time=$duration
        fi
        if [[ $duration -gt $max_time ]]; then
            max_time=$duration
        fi
    done
    
    local avg_time=$((total_time / test_count))
    
    echo "Test Count: $test_count"
    echo "Total Time: ${total_time}ms"
    echo "Average Time: ${avg_time}ms"
    echo "Min Time: ${min_time}ms"
    echo "Max Time: ${max_time}ms"
    
    # Performance thresholds
    echo -e "\n${YELLOW}Performance Thresholds:${NC}"
    if [[ $avg_time -lt 500 ]]; then
        echo -e "  Average: ${GREEN}EXCELLENT (<500ms)${NC}"
    elif [[ $avg_time -lt 1000 ]]; then
        echo -e "  Average: ${YELLOW}GOOD (<1s)${NC}"
    else
        echo -e "  Average: ${RED}NEEDS OPTIMIZATION (>1s)${NC}"
    fi
    
    # Check for regression
    echo -e "\n${YELLOW}Regression Check:${NC}"
    if [[ -f "$TEST_ROOT/.performance-baseline" ]]; then
        local baseline
        baseline=$(cat "$TEST_ROOT/.performance-baseline")
        local diff=$((avg_time - baseline))
        local percent=$((diff * 100 / baseline))
        
        if [[ $percent -gt 10 ]]; then
            echo -e "  ${RED}⚠️  Performance regression detected: ${percent}% slower${NC}"
        elif [[ $percent -lt -10 ]]; then
            echo -e "  ${GREEN}✅ Performance improved: ${percent#-}% faster${NC}"
        else
            echo -e "  ${GREEN}✅ Performance stable (within ±10%)${NC}"
        fi
    else
        echo "  No baseline found. Creating baseline..."
        echo "$avg_time" > "$TEST_ROOT/.performance-baseline"
        echo -e "  ${GREEN}✅ Baseline created: ${avg_time}ms${NC}"
    fi
}

#######################################
# Main execution
#######################################
main() {
    # Run all performance tests
    test_setup_overhead
    test_config_performance
    test_assertion_performance
    test_cleanup_performance
    # Note: Parallel tests commented out for initial run
    # test_parallel_performance
    
    # Generate report
    generate_report
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi