#!/usr/bin/env bash
# Simplified Parallel Test Runner for Vrooli BATS Tests
# Uses simple background jobs and process pools for reliable parallel execution

# Prevent duplicate loading
if [[ "${SIMPLE_PARALLEL_RUNNER_LOADED:-}" == "true" ]]; then
    return 0
fi
export SIMPLE_PARALLEL_RUNNER_LOADED="true"

# Load configuration system if available
if [[ -f "$(dirname "${BASH_SOURCE[0]}")/config-loader.bash" ]]; then
    source "$(dirname "${BASH_SOURCE[0]}")/config-loader.bash"
fi

# Configuration
readonly DEFAULT_MAX_JOBS=4
readonly RESULTS_DIR="/tmp/vrooli-parallel-results"
readonly LOCK_DIR="/tmp/vrooli-parallel-locks"

# State tracking
declare -a ACTIVE_PIDS=()
declare -a COMPLETED_TESTS=()
declare -a FAILED_TESTS=()

#######################################
# Initialize simple parallel execution
#######################################
simple_parallel::init() {
    local max_jobs="${1:-$DEFAULT_MAX_JOBS}"
    
    echo "[SIMPLE_PARALLEL] Initializing (max jobs: $max_jobs)"
    
    # Create directories
    mkdir -p "$RESULTS_DIR" "$LOCK_DIR"
    
    # Clean up any existing results
    rm -f "${RESULTS_DIR}"/result-* "${LOCK_DIR}"/lock-*
    
    # Set up cleanup on exit
    trap 'simple_parallel::cleanup' EXIT INT TERM
    
    export SIMPLE_PARALLEL_MAX_JOBS="$max_jobs"
    export SIMPLE_PARALLEL_INITIALIZED="true"
    
    echo "[SIMPLE_PARALLEL] Initialization complete"
}

#######################################
# Cleanup function
#######################################
simple_parallel::cleanup() {
    echo "[SIMPLE_PARALLEL] Cleaning up..."
    
    # Kill any remaining background jobs
    for pid in "${ACTIVE_PIDS[@]}"; do
        if kill -0 "$pid" 2>/dev/null; then
            kill -TERM "$pid" 2>/dev/null || true
        fi
    done
    
    # Clean up temp files
    rm -rf "${RESULTS_DIR}" "${LOCK_DIR}"
    
    echo "[SIMPLE_PARALLEL] Cleanup complete"
}

#######################################
# Wait for a job slot to become available
#######################################
simple_parallel::wait_for_slot() {
    local max_jobs="${SIMPLE_PARALLEL_MAX_JOBS:-$DEFAULT_MAX_JOBS}"
    
    while [[ ${#ACTIVE_PIDS[@]} -ge $max_jobs ]]; do
        # Check for completed jobs
        local new_pids=()
        for pid in "${ACTIVE_PIDS[@]}"; do
            if kill -0 "$pid" 2>/dev/null; then
                new_pids+=("$pid")
            fi
        done
        ACTIVE_PIDS=("${new_pids[@]}")
        
        # Small delay if still at capacity
        if [[ ${#ACTIVE_PIDS[@]} -ge $max_jobs ]]; then
            sleep 0.1
        fi
    done
}

#######################################
# Run a single test with simple background execution
# Arguments: $1 - test file, $2 - timeout, $3 - job id
#######################################
simple_parallel::run_test() {
    local test_file="$1"
    local timeout="${2:-60}"
    local job_id="${3:-$$}"
    local result_file="${RESULTS_DIR}/result-${job_id}"
    
    # Set up isolated environment for this test
    local test_env_file="${RESULTS_DIR}/env-${job_id}"
    cat > "$test_env_file" << EOF
export TEST_NAMESPACE="vrooli_test_${job_id}_$$_${RANDOM}"
export TEST_ENVIRONMENT="ci"
export BATS_TEST_TIMEOUT="$timeout"
EOF
    
    # Run test in background with timeout and environment isolation
    (
        # Source the test environment
        source "$test_env_file"
        
        # Record start time
        local start_time=$(date +%s)
        
        # Run the test with timeout
        if timeout "${timeout}s" bats "$test_file" > "${result_file}.output" 2>&1; then
            echo "PASSED" > "$result_file"
        else
            echo "FAILED" > "$result_file"
        fi
        
        # Record end time and duration
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        echo "$duration" > "${result_file}.duration"
        
        # Clean up environment file
        rm -f "$test_env_file"
        
        # Clean up test namespace resources
        docker ps -a --filter "name=${TEST_NAMESPACE}" --format "{{.ID}}" 2>/dev/null | xargs -r docker rm -f 2>/dev/null || true
        rm -rf "/tmp/${TEST_NAMESPACE}"* "/dev/shm/${TEST_NAMESPACE}"* 2>/dev/null || true
        
    ) &
    
    local test_pid=$!
    ACTIVE_PIDS+=("$test_pid")
    
    echo "[SIMPLE_PARALLEL] Started job $job_id (PID: $test_pid): $(basename "$test_file")"
    
    return 0
}

#######################################
# Run tests in parallel using simple job control
# Arguments: $@ - list of test files
#######################################
simple_parallel::run_tests() {
    local test_files=("$@")
    local total_tests=${#test_files[@]}
    local job_counter=0
    
    echo "[SIMPLE_PARALLEL] Running $total_tests tests in parallel"
    
    # Start tests
    for test_file in "${test_files[@]}"; do
        # Wait for available slot
        simple_parallel::wait_for_slot
        
        # Calculate timeout
        local timeout=60
        if command -v config::get_timeout >/dev/null 2>&1; then
            timeout=$(config::get_timeout "$test_file")
        fi
        
        # Start the test
        job_counter=$((job_counter + 1))
        simple_parallel::run_test "$test_file" "$timeout" "$job_counter"
    done
    
    echo "[SIMPLE_PARALLEL] All tests started, waiting for completion..."
    
    # Wait for all jobs to complete
    while [[ ${#ACTIVE_PIDS[@]} -gt 0 ]]; do
        # Check for completed jobs
        local new_pids=()
        for pid in "${ACTIVE_PIDS[@]}"; do
            if kill -0 "$pid" 2>/dev/null; then
                new_pids+=("$pid")
            fi
        done
        ACTIVE_PIDS=("${new_pids[@]}")
        
        # Progress update
        local completed=$((total_tests - ${#ACTIVE_PIDS[@]}))
        if [[ $completed -gt 0 ]]; then
            local progress=$((completed * 100 / total_tests))
            echo "[SIMPLE_PARALLEL] Progress: $completed/$total_tests completed ($progress%)"
        fi
        
        sleep 1
    done
    
    echo "[SIMPLE_PARALLEL] All tests completed"
    
    # Collect results
    local passed=0
    local failed=0
    local total_duration=0
    
    for i in $(seq 1 $job_counter); do
        local result_file="${RESULTS_DIR}/result-${i}"
        local duration_file="${RESULTS_DIR}/result-${i}.duration"
        
        if [[ -f "$result_file" ]]; then
            local status
            status=$(cat "$result_file" 2>/dev/null || echo "UNKNOWN")
            case "$status" in
                "PASSED") passed=$((passed + 1)) ;;
                "FAILED") failed=$((failed + 1)) ;;
            esac
        fi
        
        if [[ -f "$duration_file" ]]; then
            local duration
            duration=$(cat "$duration_file" 2>/dev/null || echo "0")
            total_duration=$((total_duration + duration))
        fi
    done
    
    echo "[SIMPLE_PARALLEL] Results: $passed passed, $failed failed (total duration: ${total_duration}s)"
    
    # Return success if no failures
    return $(( failed > 0 ? 1 : 0 ))
}

#######################################
# Get test results for reporting
#######################################
simple_parallel::get_results() {
    local passed=0
    local failed=0
    
    for result_file in "${RESULTS_DIR}"/result-*; do
        if [[ -f "$result_file" && ! "$result_file" =~ \.(output|duration)$ ]]; then
            local status
            status=$(cat "$result_file" 2>/dev/null || echo "UNKNOWN")
            case "$status" in
                "PASSED") passed=$((passed + 1)) ;;
                "FAILED") failed=$((failed + 1)) ;;
            esac
        fi
    done
    
    echo "passed:$passed:failed:$failed"
}

#######################################
# Check if parallel execution is supported
#######################################
simple_parallel::is_supported() {
    # Check for required commands
    if ! command -v timeout >/dev/null 2>&1; then
        echo "[SIMPLE_PARALLEL] ERROR: timeout command not available" >&2
        return 1
    fi
    
    if ! command -v bats >/dev/null 2>&1; then
        echo "[SIMPLE_PARALLEL] ERROR: bats command not available" >&2
        return 1
    fi
    
    return 0
}

# Export functions
export -f simple_parallel::init simple_parallel::cleanup simple_parallel::run_tests
export -f simple_parallel::wait_for_slot simple_parallel::run_test simple_parallel::get_results
export -f simple_parallel::is_supported

echo "[SIMPLE_PARALLEL] Simple parallel test runner loaded"