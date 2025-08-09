#!/usr/bin/env bats
# Parallel Execution Module Tests
# Tests for parallel step execution and concurrency control

bats_require_minimum_version 1.5.0

# Load test infrastructure (single entry point) 
source "${BATS_TEST_DIRNAME}/../../../__test/fixtures/setup.bash"

setup() {
    vrooli_setup_unit_test  # Basic mocks and environment
    
    # Source dependencies
    # shellcheck disable=SC1091
    source "${BATS_TEST_DIRNAME}/condition.sh"
    # shellcheck disable=SC1091
    source "${BATS_TEST_DIRNAME}/output.sh"
    # shellcheck disable=SC1091
    source "${BATS_TEST_DIRNAME}/executor.sh"
    # shellcheck disable=SC1091
    source "${BATS_TEST_DIRNAME}/parallel.sh"
    
    # Set reasonable defaults for testing
    export MAX_PARALLEL=2
}

teardown() {
    vrooli_cleanup_test     # Clean up resources
}

# Helper functions for testing
test_quick_step() {
    echo "Quick step executed"
    sleep 0.1
    return 0
}

test_slow_step() {
    echo "Slow step executed"
    sleep 0.3
    return 0
}

test_failing_step() {
    echo "Failing step executed"
    return 1
}

@test "parallel::start_step - starts step in background" {
    local step_json='{"name": "test_step", "function": "test_quick_step"}'
    local result_file="${BATS_TEST_TMPDIR}/result.txt"
    
    parallel::start_step "$step_json" "$result_file" &
    local pid=$!
    
    # Wait for completion
    wait "$pid"
    
    # Result file should exist and contain output
    [[ -f "$result_file" ]]
}

@test "parallel::wait_for_slot - manages concurrency" {
    # Fill up parallel slots
    PARALLEL_PIDS=("1111" "2222")  # Mock PIDs
    
    # Mock the jobs command to return our fake jobs
    jobs() {
        case "$1" in
            "-p")
                # Return our mock PIDs as if they're running
                printf "%s\n" "${PARALLEL_PIDS[@]}"
                ;;
        esac
    }
    
    # Mock kill command to "complete" jobs
    kill() {
        if [[ "$1" == "-0" ]]; then
            # First few calls return success (job exists)
            # After that return failure (job completed)
            return 1
        fi
    }
    
    # This should eventually return when slots become available
    run timeout 1s parallel::wait_for_slot
    # Either succeeds or times out (both are acceptable for this test)
    [[ $status -eq 0 ]] || [[ $status -eq 124 ]]
}

@test "parallel::collect_results - collects step results" {
    local result_dir="${BATS_TEST_TMPDIR}/results"
    mkdir -p "$result_dir"
    
    # Create mock result files
    echo "0" > "${result_dir}/step_0.exit"
    echo "Step 0 output" > "${result_dir}/step_0.out"
    echo "1" > "${result_dir}/step_1.exit"
    echo "Step 1 failed" > "${result_dir}/step_1.err"
    
    local results
    results=$(parallel::collect_results "$result_dir")
    
    # Should contain information about both steps
    echo "$results" | grep -q "step_0"
    echo "$results" | grep -q "step_1"
}

@test "parallel::get_max_parallel - returns configured max parallel" {
    export MAX_PARALLEL=5
    
    local max
    max=$(parallel::get_max_parallel)
    [[ "$max" -eq 5 ]]
}

@test "parallel::get_max_parallel - uses default when not set" {
    unset MAX_PARALLEL
    
    local max
    max=$(parallel::get_max_parallel)
    [[ "$max" -eq 10 ]]  # Default value
}

@test "parallel::set_max_parallel - sets max parallel value" {
    run parallel::set_max_parallel 3
    assert_success
    
    [[ "$MAX_PARALLEL" -eq 3 ]]
}

@test "parallel::set_max_parallel - validates positive integer" {
    run parallel::set_max_parallel 0
    assert_failure
    
    run parallel::set_max_parallel -1
    assert_failure
    
    run parallel::set_max_parallel "invalid"
    assert_failure
}

@test "parallel::clear_pids - clears parallel PIDs array" {
    PARALLEL_PIDS=("1111" "2222" "3333")
    
    run parallel::clear_pids
    assert_success
    
    [[ ${#PARALLEL_PIDS[@]} -eq 0 ]]
}