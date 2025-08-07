#!/usr/bin/env bash
# Vrooli Test Cleanup - Centralized Cleanup Functions
# Ensures proper cleanup of test resources and environments

# Prevent duplicate loading
if [[ "${VROOLI_CLEANUP_LOADED:-}" == "true" ]]; then
    return 0
fi
export VROOLI_CLEANUP_LOADED="true"

echo "[CLEANUP] Loading Vrooli test cleanup functions"

#######################################
# Main cleanup function - call this from teardown()
# Arguments: None
# Returns: 0
#######################################
vrooli_cleanup_test() {
    echo "[CLEANUP] Starting test cleanup"
    
    # Track cleanup steps for debugging
    local cleanup_steps=()
    
    # Cleanup in reverse order of setup
    if _vrooli_cleanup_processes; then
        cleanup_steps+=("processes")
    fi
    
    if _vrooli_cleanup_mocks; then
        cleanup_steps+=("mocks")
    fi
    
    if _vrooli_cleanup_temp_files; then
        cleanup_steps+=("temp_files")
    fi
    
    if _vrooli_cleanup_environment; then
        cleanup_steps+=("environment")
    fi
    
    if _vrooli_cleanup_shared_memory; then
        cleanup_steps+=("shared_memory")
    fi
    
    echo "[CLEANUP] Test cleanup complete (steps: ${cleanup_steps[*]})"
    return 0
}

#######################################
# Cleanup background processes
# Arguments: None
# Returns: 0 on success, 1 if errors occurred
#######################################
_vrooli_cleanup_processes() {
    echo "[CLEANUP] Cleaning up processes"
    
    local errors=0
    
    # Get list of background jobs
    local jobs_count
    jobs_count=$(jobs -p 2>/dev/null | wc -l)
    
    if [[ "$jobs_count" -gt 0 ]]; then
        echo "[CLEANUP] Killing $jobs_count background processes"
        
        # Kill background jobs
        jobs -p | while IFS= read -r pid; do
            if [[ -n "$pid" ]]; then
                if kill "$pid" 2>/dev/null; then
                    echo "[CLEANUP] Killed process: $pid"
                else
                    echo "[CLEANUP] WARNING: Failed to kill process: $pid"
                    ((errors++))
                fi
            fi
        done
        
        # Give processes time to terminate gracefully
        sleep 1
        
        # Force kill any remaining processes
        jobs -p | while IFS= read -r pid; do
            if [[ -n "$pid" ]]; then
                if kill -9 "$pid" 2>/dev/null; then
                    echo "[CLEANUP] Force killed process: $pid"
                else
                    echo "[CLEANUP] WARNING: Failed to force kill process: $pid"
                    ((errors++))
                fi
            fi
        done
    fi
    
    # Clean up any processes in our test namespace
    if [[ -n "${TEST_NAMESPACE:-}" ]]; then
        local test_processes
        test_processes=$(pgrep -f "$TEST_NAMESPACE" 2>/dev/null || true)
        
        if [[ -n "$test_processes" ]]; then
            echo "[CLEANUP] Killing test namespace processes: $test_processes"
            echo "$test_processes" | xargs -r kill 2>/dev/null || true
            sleep 1
            echo "$test_processes" | xargs -r kill -9 2>/dev/null || true
        fi
    fi
    
    return "$errors"
}

#######################################
# Cleanup mock functions and state
# Arguments: None
# Returns: 0
#######################################
_vrooli_cleanup_mocks() {
    echo "[CLEANUP] Cleaning up mocks"
    
    # Call service-specific cleanup functions if they exist
    # Only include functions that follow the pattern: service_name_cleanup (not vrooli_* or _vrooli_*)
    local cleanup_functions
    cleanup_functions=$(declare -F | grep "_cleanup$" | awk '{print $3}' | grep -v "^_\?vrooli_")
    
    for func in $cleanup_functions; do
        echo "[CLEANUP] Calling cleanup function: $func"
        "$func" 2>/dev/null || true
    done
    
    # Reset mock state variables
    unset MOCK_CALL_COUNT READ_CALL_COUNT
    unset MOCK_DOCKER_STATE MOCK_HTTP_STATE
    
    # Clean up any mock response files
    if [[ -n "${MOCK_RESPONSES_DIR:-}" && -d "$MOCK_RESPONSES_DIR" ]]; then
        rm -rf "$MOCK_RESPONSES_DIR"
        echo "[CLEANUP] Removed mock responses directory: $MOCK_RESPONSES_DIR"
    fi
    
    return 0
}

#######################################
# Cleanup temporary files and directories
# Arguments: None
# Returns: 0 on success, 1 if errors occurred
#######################################
_vrooli_cleanup_temp_files() {
    echo "[CLEANUP] Cleaning up temporary files"
    
    local errors=0
    local temp_dirs=(
        "${VROOLI_TEST_TMPDIR:-}"
        "${BATS_TEST_TMPDIR:-}"
        "${TEST_TMPDIR:-}"
        "${TMPDIR:-}/vrooli_test_*"
        "/tmp/vrooli_test_*"
    )
    
    for temp_dir in "${temp_dirs[@]}"; do
        if [[ -n "$temp_dir" && -d "$temp_dir" ]]; then
            if rm -rf "$temp_dir" 2>/dev/null; then
                echo "[CLEANUP] Removed temp directory: $temp_dir"
            else
                echo "[CLEANUP] WARNING: Failed to remove temp directory: $temp_dir"
                ((errors++))
            fi
        elif [[ "$temp_dir" =~ \* ]]; then
            # Handle glob patterns
            for dir in $temp_dir; do
                if [[ -d "$dir" ]]; then
                    if rm -rf "$dir" 2>/dev/null; then
                        echo "[CLEANUP] Removed temp directory: $dir"
                    else
                        echo "[CLEANUP] WARNING: Failed to remove temp directory: $dir"
                        ((errors++))
                    fi
                fi
            done
        fi
    done
    
    # Clean up any temporary files in /tmp with our test namespace
    if [[ -n "${TEST_NAMESPACE:-}" ]]; then
        find /tmp -name "*${TEST_NAMESPACE}*" -type f -delete 2>/dev/null || true
        find /tmp -name "*${TEST_NAMESPACE}*" -type d -empty -delete 2>/dev/null || true
    fi
    
    return "$errors"
}

#######################################
# Cleanup environment variables
# Arguments: None
# Returns: 0
#######################################
_vrooli_cleanup_environment() {
    echo "[CLEANUP] Cleaning up environment variables"
    
    # List of environment variables to clean up
    local env_vars=(
        "TEST_NAMESPACE"
        "VROOLI_TEST_TMPDIR"
        "BATS_TEST_TMPDIR"
        "MOCK_RESPONSES_DIR"
        "VROOLI_TEST_PERFORMANCE_MODE"
        "VROOLI_TEST_QUIET"
    )
    
    # Clean up service-specific environment variables
    local service_vars
    service_vars=$(env | grep "^VROOLI_.*_PORT=" | cut -d= -f1)
    env_vars+=($service_vars)
    
    service_vars=$(env | grep "^VROOLI_.*_BASE_URL=" | cut -d= -f1)
    env_vars+=($service_vars)
    
    service_vars=$(env | grep "^VROOLI_.*_CONTAINER_NAME=" | cut -d= -f1)
    env_vars+=($service_vars)
    
    # Unset variables
    for var in "${env_vars[@]}"; do
        if [[ -n "${!var:-}" ]]; then
            unset "$var"
            echo "[CLEANUP] Unset environment variable: $var"
        fi
    done
    
    return 0
}

#######################################
# Cleanup shared memory directories
# This is important for performance tests using /dev/shm
# Arguments: None
# Returns: 0 on success, 1 if errors occurred
#######################################
_vrooli_cleanup_shared_memory() {
    echo "[CLEANUP] Cleaning up shared memory"
    
    local errors=0
    
    # Only proceed if /dev/shm exists and we have a test namespace
    if [[ ! -d "/dev/shm" || -z "${TEST_NAMESPACE:-}" ]]; then
        return 0
    fi
    
    # Find and remove directories in /dev/shm with our test namespace
    local shm_dirs
    shm_dirs=$(find /dev/shm -maxdepth 2 -name "*${TEST_NAMESPACE}*" -type d 2>/dev/null || true)
    
    if [[ -n "$shm_dirs" ]]; then
        echo "[CLEANUP] Found shared memory directories to clean:"
        echo "$shm_dirs"
        
        echo "$shm_dirs" | while IFS= read -r dir; do
            if [[ -d "$dir" ]]; then
                if rm -rf "$dir" 2>/dev/null; then
                    echo "[CLEANUP] Removed shared memory directory: $dir"
                else
                    echo "[CLEANUP] WARNING: Failed to remove shared memory directory: $dir"
                    ((errors++))
                fi
            fi
        done
    fi
    
    # Clean up any files in /dev/shm with our test namespace
    local shm_files
    shm_files=$(find /dev/shm -maxdepth 2 -name "*${TEST_NAMESPACE}*" -type f 2>/dev/null || true)
    
    if [[ -n "$shm_files" ]]; then
        echo "$shm_files" | while IFS= read -r file; do
            if [[ -f "$file" ]]; then
                if rm "$file" 2>/dev/null; then
                    echo "[CLEANUP] Removed shared memory file: $file"
                else
                    echo "[CLEANUP] WARNING: Failed to remove shared memory file: $file"
                    ((errors++))
                fi
            fi
        done
    fi
    
    return "$errors"
}

#######################################
# Emergency cleanup - for when regular cleanup fails
# Arguments: None
# Returns: 0
#######################################
vrooli_emergency_cleanup() {
    echo "[CLEANUP] EMERGENCY CLEANUP - Forcing cleanup of all resources"
    
    # Kill all processes containing 'vrooli' or 'test'
    pkill -f "vrooli" 2>/dev/null || true
    pkill -f "bats" 2>/dev/null || true
    
    # Remove all temp directories
    rm -rf /tmp/vrooli_test_* 2>/dev/null || true
    rm -rf /dev/shm/vrooli_test_* 2>/dev/null || true
    
    # Clean up Docker containers if Docker is available
    if command -v docker >/dev/null 2>&1; then
        echo "[CLEANUP] Cleaning up Docker containers"
        docker ps -a --filter "name=vrooli_test" --format "{{.ID}}" | xargs -r docker rm -f 2>/dev/null || true
        docker ps -a --filter "name=test_" --format "{{.ID}}" | xargs -r docker rm -f 2>/dev/null || true
    fi
    
    echo "[CLEANUP] Emergency cleanup complete"
    return 0
}

#######################################
# Cleanup validation - verify cleanup was successful
# Arguments: None
# Returns: 0 if clean, 1 if issues remain
#######################################
vrooli_validate_cleanup() {
    echo "[CLEANUP] Validating cleanup"
    
    local issues=0
    
    # Check for remaining processes
    local remaining_processes
    remaining_processes=$(pgrep -f "vrooli_test" 2>/dev/null | wc -l)
    if [[ "$remaining_processes" -gt 0 ]]; then
        echo "[CLEANUP] WARNING: $remaining_processes processes still running"
        ((issues++))
    fi
    
    # Check for remaining temp directories
    local remaining_temps
    remaining_temps=$(find /tmp -name "vrooli_test_*" 2>/dev/null | wc -l)
    if [[ "$remaining_temps" -gt 0 ]]; then
        echo "[CLEANUP] WARNING: $remaining_temps temp directories remain"
        ((issues++))
    fi
    
    # Check for remaining shared memory
    if [[ -d "/dev/shm" ]]; then
        local remaining_shm
        remaining_shm=$(find /dev/shm -name "vrooli_test_*" 2>/dev/null | wc -l)
        if [[ "$remaining_shm" -gt 0 ]]; then
            echo "[CLEANUP] WARNING: $remaining_shm shared memory items remain"
            ((issues++))
        fi
    fi
    
    if [[ "$issues" -eq 0 ]]; then
        echo "[CLEANUP] Cleanup validation passed"
        return 0
    else
        echo "[CLEANUP] Cleanup validation failed ($issues issues)"
        return 1
    fi
}

#######################################
# Graceful cleanup with retry
# Attempts normal cleanup, retries if needed, falls back to emergency cleanup
# Arguments: None
# Returns: 0
#######################################
vrooli_graceful_cleanup() {
    echo "[CLEANUP] Starting graceful cleanup"
    
    # First attempt: normal cleanup
    if vrooli_cleanup_test; then
        if vrooli_validate_cleanup; then
            echo "[CLEANUP] Graceful cleanup successful"
            return 0
        fi
    fi
    
    echo "[CLEANUP] Normal cleanup incomplete, retrying..."
    sleep 2
    
    # Second attempt: normal cleanup with longer delays
    if vrooli_cleanup_test; then
        sleep 3
        if vrooli_validate_cleanup; then
            echo "[CLEANUP] Graceful cleanup successful (retry)"
            return 0
        fi
    fi
    
    echo "[CLEANUP] Normal cleanup failed, performing emergency cleanup"
    vrooli_emergency_cleanup
    
    echo "[CLEANUP] Graceful cleanup complete (with emergency measures)"
    return 0
}

#######################################
# Backward compatibility functions
# These provide compatibility with older test patterns
#######################################

#######################################
# Backward compatibility wrapper for cleanup_mocks
# This function is called by older test teardown() functions
# Arguments: None
# Returns: 0
#######################################
cleanup_mocks() {
    echo "[CLEANUP] Using backward compatibility cleanup_mocks wrapper"
    vrooli_cleanup_test
}

#######################################
# Export cleanup functions
#######################################
export -f vrooli_cleanup_test vrooli_emergency_cleanup
export -f vrooli_validate_cleanup vrooli_graceful_cleanup
export -f cleanup_mocks
export -f _vrooli_cleanup_processes _vrooli_cleanup_mocks _vrooli_cleanup_temp_files
export -f _vrooli_cleanup_environment _vrooli_cleanup_shared_memory

echo "[CLEANUP] Vrooli test cleanup functions loaded successfully"