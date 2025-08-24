#!/usr/bin/env bash

# Error Handler Library - Comprehensive error handling and recovery
# Part of the modular loop system
#
# This module provides:
# - Consistent error trap handlers
# - Error recovery mechanisms
# - Cleanup functions
# - Error context tracking
# - Standardized error reporting
#
# Usage:
#   source error-handler.sh
#   error_handler::init
#   error_handler::set_context "processing iteration 5"

set -euo pipefail

# Prevent multiple sourcing
if [[ -n "${_AUTO_ERROR_HANDLER_SOURCED:-}" ]]; then
    return 0
fi
readonly _AUTO_ERROR_HANDLER_SOURCED=1

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
LIB_DIR="${APP_ROOT}/auto/lib"
# shellcheck disable=SC1091
source "$LIB_DIR/constants.sh"
# shellcheck disable=SC1091
source "$LIB_DIR/error-codes.sh"

# Error context stack for better debugging
declare -g -a ERROR_CONTEXT_STACK=()
declare -g ERROR_HANDLER_INITIALIZED=false
declare -g ERROR_CLEANUP_REGISTERED=false
declare -g -a ERROR_CLEANUP_FUNCTIONS=()

# -----------------------------------------------------------------------------
# Function: error_handler::init
# Description: Initialize error handling for the current script
# Parameters: None
# Returns: 0 on success
# Side Effects: Sets up trap handlers for ERR, EXIT, INT, TERM
# -----------------------------------------------------------------------------
error_handler::init() {
    if [[ "$ERROR_HANDLER_INITIALIZED" == "true" ]]; then
        return 0
    fi
    
    # Set up error trap
    trap 'error_handler::on_error $? $LINENO "$BASH_COMMAND" "${BASH_SOURCE[0]}"' ERR
    
    # Set up exit trap for cleanup
    trap 'error_handler::on_exit $?' EXIT
    
    # Set up interrupt handlers
    trap 'error_handler::on_interrupt' INT
    trap 'error_handler::on_terminate' TERM
    
    ERROR_HANDLER_INITIALIZED=true
    return 0
}

# -----------------------------------------------------------------------------
# Function: error_handler::set_context
# Description: Push a context description onto the error context stack
# Parameters:
#   $1 - Context description (e.g., "processing file X", "iteration 5")
# Returns: 0 on success
# Side Effects: Modifies ERROR_CONTEXT_STACK
# -----------------------------------------------------------------------------
error_handler::set_context() {
    local context="${1:-}"
    if [[ -n "$context" ]]; then
        ERROR_CONTEXT_STACK+=("$context")
    fi
    return 0
}

# -----------------------------------------------------------------------------
# Function: error_handler::clear_context
# Description: Pop the most recent context from the stack
# Parameters: None
# Returns: 0 on success
# Side Effects: Modifies ERROR_CONTEXT_STACK
# -----------------------------------------------------------------------------
error_handler::clear_context() {
    if [[ ${#ERROR_CONTEXT_STACK[@]} -gt 0 ]]; then
        unset 'ERROR_CONTEXT_STACK[-1]'
    fi
    return 0
}

# -----------------------------------------------------------------------------
# Function: error_handler::on_error
# Description: Handle errors caught by ERR trap
# Parameters:
#   $1 - Exit code
#   $2 - Line number where error occurred
#   $3 - Command that failed
#   $4 - Source file
# Returns: Original exit code
# Side Effects: Logs error details, may trigger cleanup
# -----------------------------------------------------------------------------
error_handler::on_error() {
    local exit_code=$1
    local line_number=$2
    local failed_command=$3
    local source_file=$4
    
    # Skip if error is expected (e.g., in conditional)
    if [[ "${ERROR_HANDLER_SKIP_NEXT:-false}" == "true" ]]; then
        ERROR_HANDLER_SKIP_NEXT=false
        return 0
    fi
    
    # Build error message
    local error_msg="ERROR: Command failed with exit code $exit_code"
    error_msg="$error_msg at ${source_file}:${line_number}"
    error_msg="$error_msg - Command: $failed_command"
    
    # Add context if available
    if [[ ${#ERROR_CONTEXT_STACK[@]} -gt 0 ]]; then
        error_msg="$error_msg - Context: ${ERROR_CONTEXT_STACK[*]}"
    fi
    
    # Log error (check if log function exists)
    if declare -F log_with_timestamp >/dev/null 2>&1; then
        log_with_timestamp "$error_msg"
    else
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] $error_msg" >&2
    fi
    
    # Classify error for better handling
    local classified_code
    classified_code=$(error_codes::classify_worker_exit "$exit_code" "" "error at line $line_number")
    
    # Attempt recovery based on error type
    error_handler::attempt_recovery "$classified_code" "$failed_command"
    
    return "$exit_code"
}

# -----------------------------------------------------------------------------
# Function: error_handler::on_exit
# Description: Handle script exit, run cleanup functions
# Parameters:
#   $1 - Exit code
# Returns: 0
# Side Effects: Runs all registered cleanup functions
# -----------------------------------------------------------------------------
error_handler::on_exit() {
    local exit_code=$1
    
    # Run cleanup functions in reverse order
    local i
    for ((i=${#ERROR_CLEANUP_FUNCTIONS[@]}-1; i>=0; i--)); do
        if [[ -n "${ERROR_CLEANUP_FUNCTIONS[i]}" ]]; then
            # Run cleanup function, ignore errors
            "${ERROR_CLEANUP_FUNCTIONS[i]}" 2>/dev/null || true
        fi
    done
    
    # Clear context stack
    ERROR_CONTEXT_STACK=()
    
    return 0
}

# -----------------------------------------------------------------------------
# Function: error_handler::on_interrupt
# Description: Handle SIGINT (Ctrl+C)
# Parameters: None
# Returns: 130 (standard interrupt exit code)
# Side Effects: Logs interruption, triggers cleanup
# -----------------------------------------------------------------------------
error_handler::on_interrupt() {
    if declare -F log_with_timestamp >/dev/null 2>&1; then
        log_with_timestamp "Process interrupted by user (SIGINT)"
    else
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] Process interrupted by user (SIGINT)" >&2
    fi
    
    # Trigger cleanup
    error_handler::on_exit 130
    exit 130
}

# -----------------------------------------------------------------------------
# Function: error_handler::on_terminate
# Description: Handle SIGTERM
# Parameters: None
# Returns: 143 (standard SIGTERM exit code)
# Side Effects: Logs termination, triggers cleanup
# -----------------------------------------------------------------------------
error_handler::on_terminate() {
    if declare -F log_with_timestamp >/dev/null 2>&1; then
        log_with_timestamp "Process terminated (SIGTERM)"
    else
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] Process terminated (SIGTERM)" >&2
    fi
    
    # Trigger cleanup
    error_handler::on_exit 143
    exit 143
}

# -----------------------------------------------------------------------------
# Function: error_handler::register_cleanup
# Description: Register a cleanup function to run on exit
# Parameters:
#   $1 - Function name to call on cleanup
# Returns: 0 on success
# Side Effects: Adds function to ERROR_CLEANUP_FUNCTIONS array
# -----------------------------------------------------------------------------
error_handler::register_cleanup() {
    local cleanup_func="${1:-}"
    
    if [[ -z "$cleanup_func" ]]; then
        return 1
    fi
    
    # Verify function exists
    if ! declare -F "$cleanup_func" >/dev/null 2>&1; then
        return 1
    fi
    
    ERROR_CLEANUP_FUNCTIONS+=("$cleanup_func")
    ERROR_CLEANUP_REGISTERED=true
    return 0
}

# -----------------------------------------------------------------------------
# Function: error_handler::attempt_recovery
# Description: Attempt to recover from specific error types
# Parameters:
#   $1 - Classified error code
#   $2 - Failed command
# Returns: 0 if recovery possible, 1 otherwise
# Side Effects: May retry commands or adjust environment
# -----------------------------------------------------------------------------
error_handler::attempt_recovery() {
    local error_code="${1:-1}"
    local failed_command="${2:-}"
    
    # Check if error is retryable
    if ! error_codes::is_retryable "$error_code"; then
        return 1
    fi
    
    case "$error_code" in
        124) # Timeout
            if declare -F log_with_timestamp >/dev/null 2>&1; then
                log_with_timestamp "Timeout error - consider increasing TIMEOUT variable"
            fi
            ;;
        142) # Quota exhausted
            if declare -F log_with_timestamp >/dev/null 2>&1; then
                log_with_timestamp "Quota exhausted - waiting before retry"
            fi
            sleep 60  # Wait a minute before allowing retry
            ;;
        "$EXIT_CONFIGURATION_ERROR") # Configuration error
            if declare -F log_with_timestamp >/dev/null 2>&1; then
                log_with_timestamp "Configuration error - check settings and permissions"
            fi
            ;;
        "$EXIT_WORKER_UNAVAILABLE") # Worker unavailable
            if declare -F log_with_timestamp >/dev/null 2>&1; then
                log_with_timestamp "Worker unavailable - check dependencies"
            fi
            ;;
    esac
    
    return 0
}

# -----------------------------------------------------------------------------
# Function: error_handler::with_retry
# Description: Execute a command with retry logic
# Parameters:
#   $1 - Max attempts
#   $2 - Delay between attempts (seconds)
#   $3+ - Command and arguments to execute
# Returns: Exit code of command (0 on success, last failure code on exhaustion)
# Side Effects: Executes provided command multiple times
# -----------------------------------------------------------------------------
error_handler::with_retry() {
    local max_attempts="${1:-3}"
    local delay="${2:-5}"
    shift 2
    local command=("$@")
    
    local attempt=1
    local exit_code=0
    
    while [[ $attempt -le $max_attempts ]]; do
        error_handler::set_context "retry attempt $attempt/$max_attempts"
        
        # Try to execute command
        set +e
        "${command[@]}"
        exit_code=$?
        set -e
        
        error_handler::clear_context
        
        if [[ $exit_code -eq 0 ]]; then
            return 0
        fi
        
        # Check if error is retryable
        if ! error_codes::is_retryable "$exit_code"; then
            return "$exit_code"
        fi
        
        if [[ $attempt -lt $max_attempts ]]; then
            if declare -F log_with_timestamp >/dev/null 2>&1; then
                log_with_timestamp "Command failed (exit=$exit_code), retrying in ${delay}s (attempt $attempt/$max_attempts)"
            fi
            sleep "$delay"
        fi
        
        ((attempt++))
    done
    
    if declare -F log_with_timestamp >/dev/null 2>&1; then
        log_with_timestamp "Command failed after $max_attempts attempts"
    fi
    
    return "$exit_code"
}

# -----------------------------------------------------------------------------
# Function: error_handler::safe_execute
# Description: Execute a command with full error handling
# Parameters:
#   $@ - Command and arguments to execute
# Returns: Exit code of command
# Side Effects: Sets up temporary error handling context
# -----------------------------------------------------------------------------
error_handler::safe_execute() {
    local command=("$@")
    
    # Save current error handling state
    local saved_errexit
    saved_errexit=$(set +o | grep errexit)
    
    # Execute with error handling
    set +e
    "${command[@]}"
    local exit_code=$?
    
    # Restore error handling state
    eval "$saved_errexit"
    
    return "$exit_code"
}

# Export functions for external use
export -f error_handler::init
export -f error_handler::set_context
export -f error_handler::clear_context
export -f error_handler::register_cleanup
export -f error_handler::with_retry
export -f error_handler::safe_execute

# -----------------------------------------------------------------------------
# Auto-initialization
# -----------------------------------------------------------------------------
# Automatically initialize error handling when this module is sourced
# This ensures consistent error handling across all scripts without requiring
# manual initialization. Scripts can still call error_handler::init again if
# they need to reset the handlers.
if [[ "$ERROR_HANDLER_INITIALIZED" != "true" ]]; then
    error_handler::init
fi