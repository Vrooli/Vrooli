#!/usr/bin/env bash
# Parallel Test Runner for Vrooli BATS Tests
# Addresses function visibility issues through script-based execution

# Prevent duplicate loading
if [[ "${PARALLEL_RUNNER_LOADED:-}" == "true" ]]; then
    return 0
fi
export PARALLEL_RUNNER_LOADED="true"

# Load configuration system
RUNNER_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${RUNNER_DIR}/config-loader.bash"

# Configuration
readonly DEFAULT_MAX_PARALLEL_JOBS=$(config::get "global.max_parallel_tests" "4")
readonly DEFAULT_TIMEOUT=$(config::get "global.default_timeout" "60")
readonly LOCK_DIR="/tmp/vrooli-test-locks"
readonly RESULTS_DIR="/tmp/vrooli-test-results"

# Global state
declare -A JOB_PIDS
declare -A JOB_TEST_FILES
declare -A RESOURCE_LOCKS

#######################################
# Initialize parallel test execution environment
#######################################
parallel::init() {
    local max_jobs="${1:-$DEFAULT_MAX_PARALLEL_JOBS}"
    
    echo "[PARALLEL] Initializing parallel test runner (max jobs: $max_jobs)"
    
    # Create necessary directories
    mkdir -p "$LOCK_DIR" "$RESULTS_DIR"
    
    # Set up signal handlers for cleanup
    trap 'parallel::cleanup' EXIT INT TERM
    
    # Clear any existing locks from previous runs
    rm -f "${LOCK_DIR}"/vrooli-test-*
    rm -f "${RESULTS_DIR}"/test-result-*
    
    export MAX_PARALLEL_JOBS="$max_jobs"
    export PARALLEL_RUNNER_INITIALIZED="true"
    
    echo "[PARALLEL] Parallel runner initialized"
}

#######################################
# Cleanup parallel execution resources
#######################################
parallel::cleanup() {
    echo "[PARALLEL] Cleaning up parallel test execution..."
    
    # Kill any remaining background jobs
    for pid in "${JOB_PIDS[@]}"; do
        if kill -0 "$pid" 2>/dev/null; then
            echo "[PARALLEL] Killing job PID: $pid"
            kill -TERM "$pid" 2>/dev/null || true
            sleep 1
            kill -KILL "$pid" 2>/dev/null || true
        fi
    done
    
    # Release all resource locks
    for resource in "${!RESOURCE_LOCKS[@]}"; do
        parallel::release_resource_lock "$resource"
    done
    
    # Clean up temporary files
    rm -f "${LOCK_DIR}"/vrooli-test-*
    rm -f "${RESULTS_DIR}"/test-result-*
    
    echo "[PARALLEL] Cleanup complete"
}

#######################################
# Acquire lock for a resource to prevent conflicts
# Arguments: $1 - resource name
# Returns: 0 if lock acquired, 1 if failed
#######################################
parallel::acquire_resource_lock() {
    local resource="$1"
    local lock_file="${LOCK_DIR}/vrooli-test-resource-${resource}"
    local timeout=30
    local elapsed=0
    
    while [[ $elapsed -lt $timeout ]]; do
        if (set -C; echo $$ > "$lock_file") 2>/dev/null; then
            RESOURCE_LOCKS["$resource"]="$lock_file"
            echo "[PARALLEL] Acquired lock for resource: $resource"
            return 0
        fi
        
        # Check if lock holder is still alive
        if [[ -f "$lock_file" ]]; then
            local lock_pid
            lock_pid=$(cat "$lock_file" 2>/dev/null)
            if [[ -n "$lock_pid" ]] && ! kill -0 "$lock_pid" 2>/dev/null; then
                echo "[PARALLEL] Removing stale lock for resource: $resource"
                rm -f "$lock_file"
                continue
            fi
        fi
        
        sleep 1
        elapsed=$((elapsed + 1))
    done
    
    echo "[PARALLEL] Failed to acquire lock for resource: $resource" >&2
    return 1
}

#######################################
# Release lock for a resource
# Arguments: $1 - resource name
#######################################
parallel::release_resource_lock() {
    local resource="$1"
    local lock_file="${RESOURCE_LOCKS[$resource]:-}"
    
    if [[ -n "$lock_file" && -f "$lock_file" ]]; then
        rm -f "$lock_file"
        unset RESOURCE_LOCKS["$resource"]
        echo "[PARALLEL] Released lock for resource: $resource"
    fi
}

#######################################
# Extract resource dependencies from test file
# Arguments: $1 - test file path
# Output: space-separated list of resources
#######################################
parallel::extract_test_resources() {
    local test_file="$1"
    local resources=()
    
    # Extract resource name from file path
    if [[ "$test_file" =~ /([^/]+)\.bats$ ]]; then
        local test_name="${BASH_REMATCH[1]}"
        
        # Map test names to resources
        case "$test_name" in
            *ollama*) resources+=("ollama") ;;
            *whisper*) resources+=("whisper") ;;
            *n8n*) resources+=("n8n") ;;
            *qdrant*) resources+=("qdrant") ;;
            *redis*) resources+=("redis") ;;
            *postgres*) resources+=("postgres") ;;
            *docker*) resources+=("docker") ;;
            *api*) resources+=("http") ;;
            *install*) resources+=("docker" "filesystem") ;;
            *common*) resources+=("filesystem") ;;
        esac
    fi
    
    # Get dependencies from configuration if available
    for resource in "${resources[@]}"; do
        local deps
        deps=$(config::get_resource_dependencies "$resource")
        if [[ -n "$deps" ]]; then
            resources+=($deps)
        fi
    done
    
    # Remove duplicates and return
    printf '%s\n' "${resources[@]}" | sort -u | tr '\n' ' '
}

#######################################
# Run a single test in isolation (script-based, not function-based)
# Arguments: $1 - test file, $2 - timeout, $3 - job id
#######################################
parallel::run_isolated_test() {
    local test_file="$1"
    local timeout="$2"
    local job_id="$3"
    local result_file="${RESULTS_DIR}/test-result-${job_id}"
    
    # Create isolated execution script
    local exec_script="/tmp/run-test-${job_id}.sh"
    cat > "$exec_script" << 'EOF'
#!/usr/bin/env bash
set -euo pipefail

# Arguments passed to script
TEST_FILE="$1"
TIMEOUT="$2"
JOB_ID="$3"
RESULT_FILE="$4"

# Initialize isolated environment
export TEST_NAMESPACE="vrooli_test_parallel_${JOB_ID}_$$_${RANDOM}"
export TEST_ENVIRONMENT="${TEST_ENVIRONMENT:-ci}"  # Use CI environment for parallel tests

# Determine test type and create appropriate setup
TEST_DIR="$(dirname "$TEST_FILE")"
BATS_ROOT="$(cd "$TEST_DIR/../.." && pwd)"

# Create test result structure
TEST_RESULT="{
    \"test_file\": \"$TEST_FILE\",
    \"job_id\": \"$JOB_ID\",
    \"start_time\": \"$(date -Iseconds)\",
    \"timeout\": $TIMEOUT,
    \"namespace\": \"$TEST_NAMESPACE\",
    \"status\": \"running\"
}"

echo "$TEST_RESULT" > "$RESULT_FILE.tmp"

# Run the test with timeout
EXEC_START=$(date +%s.%N)
if timeout "${TIMEOUT}s" bats "$TEST_FILE" > "$RESULT_FILE.output" 2>&1; then
    TEST_STATUS="passed"
    EXIT_CODE=0
else
    TEST_STATUS="failed"
    EXIT_CODE=$?
fi
EXEC_END=$(date +%s.%N)

# Calculate execution time
EXEC_TIME=$(echo "$EXEC_END - $EXEC_START" | bc -l 2>/dev/null || echo "0")

# Update result
TEST_RESULT="{
    \"test_file\": \"$TEST_FILE\",
    \"job_id\": \"$JOB_ID\",
    \"start_time\": \"$(date -Iseconds -d "@${EXEC_START%.*}")\",
    \"end_time\": \"$(date -Iseconds)\",
    \"execution_time\": $EXEC_TIME,
    \"timeout\": $TIMEOUT,
    \"namespace\": \"$TEST_NAMESPACE\",
    \"status\": \"$TEST_STATUS\",
    \"exit_code\": $EXIT_CODE
}"

echo "$TEST_RESULT" > "$RESULT_FILE"
rm -f "$RESULT_FILE.tmp"

# Cleanup namespace resources
if command -v docker >/dev/null 2>&1; then
    docker ps -a --filter "name=${TEST_NAMESPACE}" --format "{{.ID}}" | xargs -r docker rm -f 2>/dev/null || true
fi

# Cleanup temp directories
rm -rf "/tmp/${TEST_NAMESPACE}"* 2>/dev/null || true
rm -rf "/dev/shm/${TEST_NAMESPACE}"* 2>/dev/null || true

exit $EXIT_CODE
EOF

    # Make script executable
    chmod +x "$exec_script"
    
    # Execute isolated test
    "$exec_script" "$test_file" "$timeout" "$job_id" "$result_file" &
    local test_pid=$!
    
    # Store job info
    JOB_PIDS["$job_id"]="$test_pid"
    JOB_TEST_FILES["$job_id"]="$test_file"
    
    echo "[PARALLEL] Started test job $job_id (PID: $test_pid): $(basename "$test_file")"
    
    # Cleanup script after test completes
    (
        wait "$test_pid"
        rm -f "$exec_script"
    ) &
    
    return 0
}

#######################################
# Wait for job completion and return status
# Arguments: $1 - job id
# Returns: 0 if test passed, 1 if failed
#######################################
parallel::wait_for_job() {
    local job_id="$1"
    local pid="${JOB_PIDS[$job_id]:-}"
    
    if [[ -z "$pid" ]]; then
        echo "[PARALLEL] ERROR: No PID found for job $job_id" >&2
        return 1
    fi
    
    # Wait for job completion
    if wait "$pid"; then
        echo "[PARALLEL] Job $job_id completed successfully"
        unset JOB_PIDS["$job_id"]
        unset JOB_TEST_FILES["$job_id"]
        return 0
    else
        local exit_code=$?
        echo "[PARALLEL] Job $job_id failed with exit code: $exit_code"
        unset JOB_PIDS["$job_id"]
        unset JOB_TEST_FILES["$job_id"]
        return 1
    fi
}

#######################################
# Get number of currently running jobs
#######################################
parallel::get_running_jobs_count() {
    local count=0
    for pid in "${JOB_PIDS[@]}"; do
        if kill -0 "$pid" 2>/dev/null; then
            count=$((count + 1))
        fi
    done
    echo "$count"
}

#######################################
# Wait for a job slot to become available
#######################################
parallel::wait_for_slot() {
    local max_jobs="${MAX_PARALLEL_JOBS:-4}"
    
    while [[ $(parallel::get_running_jobs_count) -ge $max_jobs ]]; do
        # Find a completed job
        for job_id in "${!JOB_PIDS[@]}"; do
            local pid="${JOB_PIDS[$job_id]}"
            if ! kill -0 "$pid" 2>/dev/null; then
                # Job completed, clean it up
                wait "$pid" 2>/dev/null || true
                unset JOB_PIDS["$job_id"]
                unset JOB_TEST_FILES["$job_id"]
                echo "[PARALLEL] Job slot freed: $job_id"
                return 0
            fi
        done
        sleep 0.1
    done
}

#######################################
# Run tests in parallel with proper resource locking
# Arguments: $@ - list of test files
#######################################
parallel::run_tests() {
    local test_files=("$@")
    local total_tests=${#test_files[@]}
    local completed=0
    local passed=0
    local failed=0
    local job_counter=0
    
    if [[ $total_tests -eq 0 ]]; then
        echo "[PARALLEL] No test files provided"
        return 1
    fi
    
    echo "[PARALLEL] Running $total_tests tests in parallel (max concurrent: ${MAX_PARALLEL_JOBS:-4})"
    
    # Start tests
    for test_file in "${test_files[@]}"; do
        # Wait for available slot
        parallel::wait_for_slot
        
        # Determine resources needed for this test
        local resources
        resources=$(parallel::extract_test_resources "$test_file")
        
        # Acquire resource locks
        local locks_acquired=()
        local lock_failed=false
        for resource in $resources; do
            if parallel::acquire_resource_lock "$resource"; then
                locks_acquired+=("$resource")
            else
                echo "[PARALLEL] Failed to acquire lock for $resource, skipping test: $test_file"
                lock_failed=true
                break
            fi
        done
        
        if [[ "$lock_failed" == "true" ]]; then
            # Release any locks we did acquire
            for resource in "${locks_acquired[@]}"; do
                parallel::release_resource_lock "$resource"
            done
            continue
        fi
        
        # Calculate timeout for this test
        local timeout
        timeout=$(config::get_timeout "$test_file")
        
        # Start the test
        job_counter=$((job_counter + 1))
        parallel::run_isolated_test "$test_file" "$timeout" "$job_counter"
        
        # Schedule lock release after job completion
        (
            parallel::wait_for_job "$job_counter"
            for resource in "${locks_acquired[@]}"; do
                parallel::release_resource_lock "$resource"
            done
        ) &
    done
    
    # Wait for all jobs to complete
    echo "[PARALLEL] Waiting for all tests to complete..."
    while [[ $(parallel::get_running_jobs_count) -gt 0 ]]; do
        sleep 1
        
        # Update progress
        local current_completed=0
        for result_file in "${RESULTS_DIR}"/test-result-*; do
            [[ -f "$result_file" ]] && current_completed=$((current_completed + 1))
        done
        
        if [[ $current_completed -ne $completed ]]; then
            completed=$current_completed
            local progress=$((completed * 100 / total_tests))
            echo "[PARALLEL] Progress: $completed/$total_tests tests completed ($progress%)"
        fi
    done
    
    # Collect results
    for result_file in "${RESULTS_DIR}"/test-result-*; do
        if [[ -f "$result_file" ]]; then
            local status
            status=$(grep '"status"' "$result_file" | cut -d'"' -f4)
            if [[ "$status" == "passed" ]]; then
                passed=$((passed + 1))
            else
                failed=$((failed + 1))
            fi
        fi
    done
    
    echo "[PARALLEL] Test execution completed:"
    echo "  Total: $total_tests"
    echo "  Passed: $passed"
    echo "  Failed: $failed"
    
    # Return success if no failures
    return $(( failed > 0 ? 1 : 0 ))
}

#######################################
# Export all functions for use in scripts
#######################################
export -f parallel::init parallel::cleanup parallel::run_tests
export -f parallel::acquire_resource_lock parallel::release_resource_lock
export -f parallel::extract_test_resources parallel::run_isolated_test
export -f parallel::wait_for_job parallel::get_running_jobs_count parallel::wait_for_slot

echo "[PARALLEL] Parallel test runner loaded"