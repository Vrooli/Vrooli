#!/usr/bin/env bash
# Lifecycle Engine - Parallel Execution Module
# Handles parallel execution of steps with concurrency control

set -euo pipefail

# Determine script directory
_HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${_HERE}/../../utils/var.sh"

# Guard against re-sourcing
[[ -n "${_PARALLEL_MODULE_LOADED:-}" ]] && return 0
declare -gr _PARALLEL_MODULE_LOADED=1

# Source dependencies
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "$var_LIB_LIFECYCLE_DIR/lib/executor.sh"

# Parallel execution state
declare -ga PARALLEL_PIDS=()
declare -gA PARALLEL_RESULTS=()
declare -g MAX_PARALLEL="${MAX_PARALLEL:-10}"

#######################################
# Execute steps in parallel
# Arguments:
#   $1 - JSON array of steps
# Returns:
#   0 if all succeed, 1 if any fail
#######################################
parallel::execute() {
    local steps="$1"
    
    local step_count
    step_count=$(echo "$steps" | jq 'length')
    
    if [[ $step_count -eq 0 ]]; then
        return 0
    fi
    
    log::info "Starting $step_count parallel steps (max concurrent: $MAX_PARALLEL)"
    
    # Clear previous state
    PARALLEL_PIDS=()
    PARALLEL_RESULTS=()
    
    # Create temporary directory for results
    local result_dir
    result_dir=$(mktemp -d -t lifecycle-parallel.XXXXXX)
    trap 'rm -rf "$result_dir"' EXIT
    
    # Start steps with concurrency control
    local i=0
    while [[ $i -lt $step_count ]]; do
        # Wait if we've reached max parallel
        while [[ ${#PARALLEL_PIDS[@]} -ge $MAX_PARALLEL ]]; do
            parallel::wait_for_slot
        done
        
        local step
        step=$(echo "$steps" | jq ".[$i]")
        
        # Start step in background
        parallel::start_step "$step" "$result_dir/step_$i" &
        local pid=$!
        PARALLEL_PIDS+=("$pid")
        
        ((i++))
    done
    
    # Wait for all steps to complete
    parallel::wait_all "$result_dir"
    
    # Check results
    local failed=0
    for result_file in "$result_dir"/step_*; do
        if [[ -f "$result_file" ]]; then
            local exit_code
            exit_code=$(cat "$result_file")
            if [[ $exit_code -ne 0 ]]; then
                failed=1
            fi
        fi
    done
    
    if [[ $failed -eq 0 ]]; then
        log::success "All parallel steps completed successfully"
        return 0
    else
        log::error "Some parallel steps failed"
        return 1
    fi
}

#######################################
# Execute parallel group with dependencies
# Arguments:
#   $1 - JSON object with parallel configuration
# Returns:
#   0 if all succeed, 1 if any fail
#######################################
parallel::execute_group() {
    local group_config="$1"
    
    local group_name steps max_concurrent strategy
    group_name=$(echo "$group_config" | jq -r '.name // "parallel-group"')
    steps=$(echo "$group_config" | jq -r '.steps // []')
    max_concurrent=$(echo "$group_config" | jq -r '.max_concurrent // 0')
    strategy=$(echo "$group_config" | jq -r '.strategy // "all"')
    
    # Set max concurrent if specified
    if [[ $max_concurrent -gt 0 ]]; then
        local old_max=$MAX_PARALLEL
        MAX_PARALLEL=$max_concurrent
    fi
    
    log::info "Executing parallel group: $group_name (strategy: $strategy)"
    
    local exit_code=0
    case "$strategy" in
        all)
            # Execute all steps, wait for all
            parallel::execute "$steps" || exit_code=$?
            ;;
        race)
            # Execute all, return when first completes
            parallel::execute_race "$steps" || exit_code=$?
            ;;
        fail_fast)
            # Stop on first failure
            parallel::execute_fail_fast "$steps" || exit_code=$?
            ;;
        *)
            log::error "Unknown parallel strategy: $strategy"
            exit_code=1
            ;;
    esac
    
    # Restore max concurrent
    if [[ $max_concurrent -gt 0 ]]; then
        MAX_PARALLEL=$old_max
    fi
    
    return $exit_code
}

#######################################
# Start a step in background
# Arguments:
#   $1 - Step JSON
#   $2 - Result file path
#######################################
parallel::start_step() {
    local step="$1"
    local result_file="$2"
    
    # Handle delay if specified
    local delay
    delay=$(echo "$step" | jq -r '.delay // empty')
    if [[ -n "$delay" ]]; then
        local delay_seconds
        delay_seconds=$(executor::parse_duration "$delay")
        sleep "$delay_seconds"
    fi
    
    # Execute step and save result
    local exit_code=0
    executor::run_step "$step" || exit_code=$?
    
    echo "$exit_code" > "$result_file"
    exit $exit_code
}

#######################################
# Wait for a slot to become available
#######################################
parallel::wait_for_slot() {
    local new_pids=()
    
    for pid in "${PARALLEL_PIDS[@]}"; do
        if kill -0 "$pid" 2>/dev/null; then
            # Still running
            new_pids+=("$pid")
        fi
    done
    
    PARALLEL_PIDS=("${new_pids[@]}")
    
    # If still at max, wait a bit
    if [[ ${#PARALLEL_PIDS[@]} -ge $MAX_PARALLEL ]]; then
        sleep 0.1
    fi
}

#######################################
# Wait for all parallel steps
# Arguments:
#   $1 - Result directory
#######################################
parallel::wait_all() {
    local result_dir="$1"
    
    log::info "Waiting for ${#PARALLEL_PIDS[@]} parallel steps to complete..."
    
    for pid in "${PARALLEL_PIDS[@]}"; do
        if ! wait "$pid"; then
            log::warning "Step with PID $pid failed"
        fi
    done
    
    PARALLEL_PIDS=()
}

#######################################
# Execute with race strategy (first to complete wins)
# Arguments:
#   $1 - JSON array of steps
# Returns:
#   Exit code of first completed step
#######################################
parallel::execute_race() {
    local steps="$1"
    
    local step_count
    step_count=$(echo "$steps" | jq 'length')
    
    log::info "Starting race between $step_count steps"
    
    # Create result directory
    local result_dir
    result_dir=$(mktemp -d -t lifecycle-race.XXXXXX)
    trap 'rm -rf "$result_dir"' EXIT
    
    # Start all steps
    local i=0
    while [[ $i -lt $step_count ]]; do
        local step
        step=$(echo "$steps" | jq ".[$i]")
        
        parallel::start_step "$step" "$result_dir/step_$i" &
        PARALLEL_PIDS+=($!)
        
        ((i++))
    done
    
    # Wait for first to complete
    local winner_pid=""
    local winner_exit_code=1
    
    while [[ ${#PARALLEL_PIDS[@]} -gt 0 ]]; do
        for pid in "${PARALLEL_PIDS[@]}"; do
            if ! kill -0 "$pid" 2>/dev/null; then
                # This one finished
                winner_pid=$pid
                wait "$pid" || winner_exit_code=$?
                break 2
            fi
        done
        sleep 0.1
    done
    
    # Kill remaining processes
    for pid in "${PARALLEL_PIDS[@]}"; do
        if [[ "$pid" != "$winner_pid" ]]; then
            kill "$pid" 2>/dev/null || true
        fi
    done
    
    log::info "Race winner: PID $winner_pid (exit code: $winner_exit_code)"
    
    return $winner_exit_code
}

#######################################
# Execute with fail-fast strategy
# Arguments:
#   $1 - JSON array of steps
# Returns:
#   0 if all succeed, 1 on first failure
#######################################
parallel::execute_fail_fast() {
    local steps="$1"
    
    local step_count
    step_count=$(echo "$steps" | jq 'length')
    
    log::info "Starting $step_count steps with fail-fast strategy"
    
    # Create result directory
    local result_dir
    result_dir=$(mktemp -d -t lifecycle-failfast.XXXXXX)
    trap 'rm -rf "$result_dir"' EXIT
    
    # Start all steps
    local i=0
    while [[ $i -lt $step_count ]]; do
        local step
        step=$(echo "$steps" | jq ".[$i]")
        
        parallel::start_step "$step" "$result_dir/step_$i" &
        PARALLEL_PIDS+=($!)
        
        ((i++))
    done
    
    # Monitor for failures
    local failed=0
    while [[ ${#PARALLEL_PIDS[@]} -gt 0 ]]; do
        local new_pids=()
        
        for pid in "${PARALLEL_PIDS[@]}"; do
            if kill -0 "$pid" 2>/dev/null; then
                # Still running
                new_pids+=("$pid")
            else
                # Finished - check result
                local exit_code=0
                wait "$pid" || exit_code=$?
                
                if [[ $exit_code -ne 0 ]]; then
                    log::error "Step failed (PID: $pid), stopping all others"
                    failed=1
                    break
                fi
            fi
        done
        
        if [[ $failed -eq 1 ]]; then
            # Kill all remaining
            for pid in "${new_pids[@]}"; do
                kill "$pid" 2>/dev/null || true
            done
            return 1
        fi
        
        PARALLEL_PIDS=("${new_pids[@]}")
        
        if [[ ${#PARALLEL_PIDS[@]} -gt 0 ]]; then
            sleep 0.1
        fi
    done
    
    log::success "All steps completed successfully"
    return 0
}

#######################################
# Execute sequential steps (for completeness)
# Arguments:
#   $1 - JSON array of steps
# Returns:
#   0 if all succeed, 1 if any fail
#######################################
parallel::execute_sequential() {
    local steps="$1"
    
    local step_count
    step_count=$(echo "$steps" | jq 'length')
    
    local i=0
    while [[ $i -lt $step_count ]]; do
        local step
        step=$(echo "$steps" | jq ".[$i]")
        
        if ! executor::run_step "$step"; then
            return 1
        fi
        
        ((i++))
    done
    
    return 0
}

#######################################
# Kill all parallel processes
#######################################
parallel::kill_all() {
    if [[ ${#PARALLEL_PIDS[@]} -gt 0 ]]; then
        log::info "Killing ${#PARALLEL_PIDS[@]} background processes..."
        
        for pid in "${PARALLEL_PIDS[@]}"; do
            if kill -0 "$pid" 2>/dev/null; then
                kill "$pid" 2>/dev/null || true
            fi
        done
        
        PARALLEL_PIDS=()
    fi
}

#######################################
# Get parallel execution statistics
#######################################
parallel::get_stats() {
    echo "Parallel Execution Stats:"
    echo "  Active processes: ${#PARALLEL_PIDS[@]}"
    echo "  Max concurrent: $MAX_PARALLEL"
    
    if [[ ${#PARALLEL_RESULTS[@]} -gt 0 ]]; then
        echo "  Results:"
        for key in "${!PARALLEL_RESULTS[@]}"; do
            echo "    $key: ${PARALLEL_RESULTS[$key]}"
        done
    fi
}

#######################################
# Logging functions
#######################################
log::info() {
    echo "[PARALLEL] $*" >&2
}

log::success() {
    echo "[PARALLEL-SUCCESS] $*" >&2
}

log::warning() {
    echo "[PARALLEL-WARNING] $*" >&2
}

log::error() {
    echo "[PARALLEL-ERROR] $*" >&2
}

# Cleanup on exit
trap parallel::kill_all EXIT

# If sourced for testing, don't auto-execute
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    echo "This script should be sourced, not executed directly" >&2
    exit 1
fi