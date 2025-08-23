#!/usr/bin/env bash
# Mock Utilities - Centralized logging and common functions for all mocks
# This file provides shared functionality to eliminate boilerplate across mock files

# Prevent duplicate loading
if [[ "${MOCK_UTILS_LOADED:-}" == "true" ]]; then
    return 0
fi
export MOCK_UTILS_LOADED="true"

# Source var.sh for directory variables
# shellcheck disable=SC1091
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Configuration variables (can be overridden for testing)
export MOCK_UTILS_VERBOSE="${MOCK_UTILS_VERBOSE:-true}"
export MOCK_UTILS_AUTO_INIT="${MOCK_UTILS_AUTO_INIT:-true}"

# Only show loading message if verbose
if [[ "$MOCK_UTILS_VERBOSE" == "true" ]]; then
    echo "[MOCK_UTILS] Loading mock utilities"
fi

#######################################
# Initialize mock logging system
# Sets up the centralized logging directory in data/test-outputs/
# Arguments: $1 - optional custom log directory (for testing)
# Returns: 0 on success, 1 on failure
# Environment: MOCK_LOG_DIR will be set to the log directory path
#######################################
mock::init_logging() {
    local custom_log_dir="${1:-}"
    
    # Return early if already initialized (unless custom directory provided)
    if [[ -n "${MOCK_LOG_DIR:-}" && -z "$custom_log_dir" ]]; then
        return 0
    fi
    
    # Use custom directory if provided (for testing)
    if [[ -n "$custom_log_dir" ]]; then
        export MOCK_LOG_DIR="$custom_log_dir"
    else
        # Find project root by traversing up from test directory
        local project_root
        if [[ -n "${VROOLI_TEST_ROOT:-}" ]]; then
            # Use existing test root and go up two levels: scripts/__test -> scripts -> project
            project_root="$(cd "${VROOLI_TEST_ROOT}/../.." && pwd)"
        else
            # Fallback: traverse up looking for package.json
            project_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
            while [[ "$project_root" != "/" ]] && [[ ! -f "$project_root/package.json" ]]; do
                project_root="$(cd "$project_root/.." && pwd)"
            done
        fi
        
        if [[ ! -f "$project_root/package.json" ]]; then
            if [[ "$MOCK_UTILS_VERBOSE" == "true" ]]; then
                echo "[MOCK_UTILS] WARNING: Could not find project root, using fallback" >&2
            fi
            project_root="/tmp"
        fi
        
        # Set up logging directory in project-level data/test-outputs/
        export MOCK_LOG_DIR="$project_root/data/test-outputs/mock-logs"
    fi
    
    # Create directory structure
    if ! mkdir -p "$MOCK_LOG_DIR" 2>/dev/null; then
        if [[ "$MOCK_UTILS_VERBOSE" == "true" ]]; then
            echo "[MOCK_UTILS] ERROR: Could not create mock log directory: $MOCK_LOG_DIR" >&2
        fi
        # Fallback to temp directory
        export MOCK_LOG_DIR="${TMPDIR:-/tmp}/vrooli-mock-logs-$$"
        if ! mkdir -p "$MOCK_LOG_DIR" 2>/dev/null; then
            if [[ "$MOCK_UTILS_VERBOSE" == "true" ]]; then
                echo "[MOCK_UTILS] ERROR: Could not create fallback directory: $MOCK_LOG_DIR" >&2
            fi
            return 1
        fi
        if [[ "$MOCK_UTILS_VERBOSE" == "true" ]]; then
            echo "[MOCK_UTILS] Using fallback directory: $MOCK_LOG_DIR" >&2
        fi
    fi
    
    # Initialize log files with headers
    if ! mock::_create_log_files; then
        return 1
    fi
    
    # Set MOCK_RESPONSES_DIR for backwards compatibility with existing code
    export MOCK_RESPONSES_DIR="$MOCK_LOG_DIR"
    
    if [[ "$MOCK_UTILS_VERBOSE" == "true" ]]; then
        echo "[MOCK_UTILS] Mock logging initialized: $MOCK_LOG_DIR"
    fi
    return 0
}

#######################################
# Create log files with headers (internal function)
# Arguments: None
# Returns: 0 on success, 1 on failure
#######################################
mock::_create_log_files() {
    if [[ -z "${MOCK_LOG_DIR:-}" ]]; then
        return 1
    fi
    
    local timestamp
    timestamp=$(date '+%Y-%m-%d %H:%M:%S' 2>/dev/null) || timestamp="$(date 2>/dev/null)" || timestamp="unknown"
    
    local log_files=(
        "command_calls.log:Mock call log started at $timestamp"
        "docker_calls.log:Docker mock calls started at $timestamp"
        "http_calls.log:HTTP mock calls started at $timestamp"
        "used_mocks.log:Mock state changes started at $timestamp"
    )
    
    for log_entry in "${log_files[@]}"; do
        local filename="${log_entry%%:*}"
        local header="${log_entry#*:}"
        
        if ! echo "# $header" > "$MOCK_LOG_DIR/$filename" 2>/dev/null; then
            if [[ "$MOCK_UTILS_VERBOSE" == "true" ]]; then
                echo "[MOCK_UTILS] ERROR: Could not create log file: $MOCK_LOG_DIR/$filename" >&2
            fi
            return 1
        fi
    done
    
    return 0
}

#######################################
# Log a mock command call with timestamp and system identification
# Arguments: $1 - system name (docker, http, system, etc), $@ - command details
# Returns: 0 on success, 1 on failure (configurable via MOCK_LOG_CALL_STRICT)
# Environment: MOCK_LOG_CALL_STRICT - if "true", returns 1 on logging failure
#######################################
mock::log_call() {
    local system="$1"
    shift
    local call_details="$*"
    
    # Validate inputs - empty system name should always fail
    if [[ -z "$system" ]]; then
        if [[ "$MOCK_UTILS_VERBOSE" == "true" ]]; then
            echo "[MOCK_UTILS] ERROR: mock::log_call requires system name" >&2
        fi
        return 1
    fi
    
    # Ensure logging is initialized
    if [[ -z "${MOCK_LOG_DIR:-}" ]]; then
        if ! mock::init_logging; then
            if [[ -n "${MOCK_LOG_CALL_STRICT:-}" ]]; then
                return 1
            fi
            return 0
        fi
    fi
    
    # Create timestamp with fallback - use command to avoid circular calls with mock
    local timestamp
    timestamp=$(command date '+%Y-%m-%d %H:%M:%S' 2>/dev/null) || timestamp="$(command date 2>/dev/null)" || timestamp="unknown"
    
    # Determine appropriate log file based on system
    local log_file
    case "$system" in
        "docker"|"docker-compose")
            log_file="$MOCK_LOG_DIR/docker_calls.log"
            ;;
        "http"|"curl"|"wget")
            log_file="$MOCK_LOG_DIR/http_calls.log"
            ;;
        *)
            log_file="$MOCK_LOG_DIR/command_calls.log"
            ;;
    esac
    
    # Log with consistent format: [timestamp] system: details
    local log_entry="[$timestamp] $system: $call_details"
    if ! echo "$log_entry" >> "$log_file" 2>/dev/null; then
        # Log file write failed - this is a failure in strict mode
        if [[ -n "${MOCK_LOG_CALL_STRICT:-}" ]]; then
            if [[ "$MOCK_UTILS_VERBOSE" == "true" ]]; then
                echo "[MOCK_UTILS] ERROR: Failed to log to file in strict mode: $log_file" >&2 2>/dev/null || true
            fi
            return 1
        fi
        
        # Try fallback to stderr in non-strict mode
        if ! echo "$log_entry" >&2 2>/dev/null; then
            # Complete failure - can't even write to stderr
            if [[ "$MOCK_UTILS_VERBOSE" == "true" ]]; then
                echo "[MOCK_UTILS] ERROR: Failed to log mock call: $system: $call_details" >&2 2>/dev/null || true
            fi
            return 0  # Non-fatal in non-strict mode
        fi
        if [[ "$MOCK_UTILS_VERBOSE" == "true" ]]; then
            echo "[MOCK_UTILS] WARNING: Used stderr fallback for logging" >&2 2>/dev/null || true
        fi
    fi
    
    return 0
}

#######################################
# Log mock state changes (container states, endpoint responses, etc)
# Arguments: $1 - state type, $2 - identifier, $3 - new state
# Returns: 0 on success, 1 on failure (configurable via MOCK_LOG_STATE_STRICT)
# Environment: MOCK_LOG_STATE_STRICT - if "true", returns 1 on logging failure
#######################################
mock::log_state() {
    local state_type="$1"
    local identifier="$2" 
    local new_state="$3"
    
    # Validate inputs - empty state_type should always fail
    if [[ -z "$state_type" ]]; then
        if [[ "$MOCK_UTILS_VERBOSE" == "true" ]]; then
            echo "[MOCK_UTILS] ERROR: mock::log_state requires state_type" >&2
        fi
        return 1
    fi
    
    # Ensure logging is initialized
    if [[ -z "${MOCK_LOG_DIR:-}" ]]; then
        if ! mock::init_logging; then
            if [[ -n "${MOCK_LOG_STATE_STRICT:-}" ]]; then
                return 1
            fi
            return 0
        fi
    fi
    
    # Create timestamp with fallback - use command to avoid circular calls with mock
    local timestamp
    timestamp=$(command date '+%Y-%m-%d %H:%M:%S' 2>/dev/null) || timestamp="$(command date 2>/dev/null)" || timestamp="unknown"
    
    # Format state entry
    local state_entry="[$timestamp] ${state_type}:${identifier:-}:${new_state:-}"
    
    # Attempt to log to file
    if ! echo "$state_entry" >> "$MOCK_LOG_DIR/used_mocks.log" 2>/dev/null; then
        # State log file write failed
        if [[ "$MOCK_UTILS_VERBOSE" == "true" ]]; then
            echo "[MOCK_UTILS] ERROR: Failed to log state change: $state_entry" >&2 2>/dev/null || true
        fi
        if [[ -n "${MOCK_LOG_STATE_STRICT:-}" ]]; then
            return 1
        fi
        return 0
    fi
    
    return 0
}

#######################################
# Combined logging and verification recording
# This replaces the common pattern of both logging and calling mock::verify::record_call
# Arguments: $1 - system name, $@ - command details
# Returns: 0 on success, 1 if any operation fails
#######################################
mock::log_and_verify() {
    local system="$1"
    shift
    local call_details="$*"
    local result=0
    
    # Validate inputs
    if [[ -z "$system" ]]; then
        if [[ "$MOCK_UTILS_VERBOSE" == "true" ]]; then
            echo "[MOCK_UTILS] ERROR: mock::log_and_verify requires system name" >&2
        fi
        return 1
    fi
    
    # Log the call
    if ! mock::log_call "$system" "$call_details"; then
        result=1
        if [[ "$MOCK_UTILS_VERBOSE" == "true" ]]; then
            echo "[MOCK_UTILS] WARNING: Logging failed for: $system $call_details" >&2
        fi
    fi
    
    # Record for verification if verification system is available
    if command -v mock::verify::record_call &>/dev/null; then
        if ! mock::verify::record_call "$system" "$call_details"; then
            result=1
            if [[ "$MOCK_UTILS_VERBOSE" == "true" ]]; then
                echo "[MOCK_UTILS] WARNING: Verification recording failed for: $system $call_details" >&2
            fi
        fi
    fi
    
    return $result
}

#######################################
# Get the current mock log directory
# Arguments: None
# Returns: Prints the log directory path, returns 0 on success, 1 if no directory
#######################################
mock::get_log_dir() {
    if [[ -z "${MOCK_LOG_DIR:-}" ]]; then
        if ! mock::init_logging >/dev/null 2>&1; then
            if [[ "$MOCK_UTILS_VERBOSE" == "true" ]]; then
                echo "[MOCK_UTILS] ERROR: Could not initialize logging to get log directory" >&2
            fi
            return 1
        fi
    fi
    
    if [[ -n "${MOCK_LOG_DIR:-}" ]]; then
        echo "$MOCK_LOG_DIR"
        return 0
    else
        return 1
    fi
}

#######################################
# Clean up mock logs (useful for test cleanup)
# Arguments: $1 - "reset" to also unset MOCK_LOG_DIR (optional)
# Returns: 0 on success, 1 on failure
#######################################
mock::cleanup_logs() {
    local reset_mode="$1"
    local result=0
    
    if [[ -n "${MOCK_LOG_DIR:-}" ]]; then
        if [[ -d "$MOCK_LOG_DIR" ]]; then
            # Try to remove log files
            if ! trash::safe_remove "$MOCK_LOG_DIR"/*.log --test-cleanup 2>/dev/null; then
                if [[ "$MOCK_UTILS_VERBOSE" == "true" ]]; then
                    echo "[MOCK_UTILS] WARNING: Could not remove all log files from: $MOCK_LOG_DIR" >&2
                fi
                result=1
            fi
            
            if [[ "$MOCK_UTILS_VERBOSE" == "true" ]]; then
                echo "[MOCK_UTILS] Cleaned up mock logs in: $MOCK_LOG_DIR"
            fi
        else
            if [[ "$MOCK_UTILS_VERBOSE" == "true" ]]; then
                echo "[MOCK_UTILS] WARNING: Mock log directory does not exist: $MOCK_LOG_DIR" >&2
            fi
        fi
        
        # Reset the directory variable if requested
        if [[ "$reset_mode" == "reset" ]]; then
            unset MOCK_LOG_DIR
            unset MOCK_RESPONSES_DIR
        fi
    else
        if [[ "$MOCK_UTILS_VERBOSE" == "true" ]]; then
            echo "[MOCK_UTILS] No mock log directory to clean up"
        fi
    fi
    
    return $result
}

#######################################
# Print summary of mock activity
# Arguments: None
# Returns: 0 on success, 1 if no logs available
#######################################
mock::print_log_summary() {
    # If logging was never initialized, there are no logs to summarize
    if [[ -z "${MOCK_LOG_DIR:-}" ]]; then
        echo "[MOCK_UTILS] No mock logs available (logging not initialized)"
        return 1
    fi
    
    if [[ ! -d "$MOCK_LOG_DIR" ]]; then
        echo "[MOCK_UTILS] No mock logs available (directory not found: ${MOCK_LOG_DIR:-unset})"
        return 1
    fi
    
    echo "[MOCK_UTILS] Mock Log Summary"
    echo "============================="
    echo "Log Directory: $MOCK_LOG_DIR"
    echo ""
    
    local found_logs=false
    for log_file in "$MOCK_LOG_DIR"/*.log; do
        if [[ -f "$log_file" ]]; then
            local filename=$(basename "$log_file")
            local count=$(grep -c "^\[" "$log_file" 2>/dev/null || echo "0")
            echo "  $filename: $count entries"
            found_logs=true
        fi
    done
    
    if [[ "$found_logs" != "true" ]]; then
        echo "  No log files found"
        return 1
    fi
    
    echo ""
    echo "View logs with:"
    echo "  cat $MOCK_LOG_DIR/command_calls.log"
    echo "  cat $MOCK_LOG_DIR/docker_calls.log"
    echo "  cat $MOCK_LOG_DIR/http_calls.log"
    echo "  cat $MOCK_LOG_DIR/used_mocks.log"
    
    return 0
}

#######################################
# Export functions for use in other scripts
#######################################
export -f mock::init_logging
export -f mock::_create_log_files
export -f mock::log_call
export -f mock::log_state
export -f mock::log_and_verify
export -f mock::get_log_dir
export -f mock::cleanup_logs
export -f mock::print_log_summary

#######################################
# Add utility functions for testing
#######################################

# Reset all mock utils state (for testing)
mock::reset_all() {
    if [[ -n "${MOCK_LOG_DIR:-}" ]]; then
        mock::cleanup_logs >/dev/null 2>&1 || true
    fi
    
    unset MOCK_LOG_DIR
    unset MOCK_RESPONSES_DIR
    # Don't unset MOCK_UTILS_LOADED as it prevents reloading
    return 0
}

# Check if logging is initialized (for testing)
mock::is_initialized() {
    [[ -n "${MOCK_LOG_DIR:-}" && -d "${MOCK_LOG_DIR:-}" ]]
}

export -f mock::reset_all
export -f mock::is_initialized

#######################################
# Auto-initialize logging when this file is sourced (if enabled)
#######################################
if [[ "$MOCK_UTILS_AUTO_INIT" == "true" ]]; then
    mock::init_logging >/dev/null 2>&1 || true
fi

if [[ "$MOCK_UTILS_VERBOSE" == "true" ]]; then
    echo "[MOCK_UTILS] Mock utilities loaded successfully"
fi