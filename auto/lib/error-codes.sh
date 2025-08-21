#!/usr/bin/env bash

# Error Codes Library - Unified error classification and handling
# Part of the modular loop system
#
# This module provides centralized error code management and classification
# for worker process exit codes based on output patterns.
#
# EXIT CODE REFERENCE:
# --------------------
# Standard POSIX codes:
#   0   - Success
#   1   - General errors
#   2   - Misuse of shell builtins
#   126 - Command invoked cannot execute
#   127 - Command not found
#   128+n - Fatal error signal "n"
#   130 - Script terminated by Control-C (128 + 2 SIGINT)
#   143 - Terminated by SIGTERM (128 + 15)
#
# Custom application codes (130-255):
#   124 - Command timeout (from GNU timeout utility)
#   137 - Killed by SIGKILL (128 + 9)
#   139 - Segmentation fault (128 + 11 SIGSEGV)
#   141 - Broken pipe (128 + 13 SIGPIPE)
#   142 - Quota/rate limit exhausted (custom)
#   150 - Configuration error (custom)
#   151 - Worker/dependencies unavailable (custom)
#   152 - Lock acquisition failed (custom)
#   153 - Resource limit exceeded (custom)
#   154 - Network connectivity error (custom)
#   155 - Authentication/permission error (custom)
#   156 - Data corruption/validation error (custom)
#   157 - Partial success with warnings (custom)

set -euo pipefail

# Prevent multiple sourcing
if [[ -n "${_AUTO_ERROR_CODES_SOURCED:-}" ]]; then
    return 0
fi
readonly _AUTO_ERROR_CODES_SOURCED=1

# Source constants for exit code definitions
LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "$LIB_DIR/constants.sh"

# -----------------------------------------------------------------------------
# STANDARD EXIT CODES
# -----------------------------------------------------------------------------
readonly EXIT_SUCCESS=0
readonly EXIT_GENERAL_ERROR=1
readonly EXIT_MISUSE=2
readonly EXIT_CANNOT_EXECUTE=126
readonly EXIT_COMMAND_NOT_FOUND=127

# -----------------------------------------------------------------------------
# SIGNAL-BASED EXIT CODES (128 + signal number)
# -----------------------------------------------------------------------------
readonly EXIT_SIGHUP=$((128 + 1))    # Terminal hangup
readonly EXIT_SIGINT=$((128 + 2))    # Interrupt (Ctrl+C)
readonly EXIT_SIGQUIT=$((128 + 3))   # Quit
readonly EXIT_SIGILL=$((128 + 4))    # Illegal instruction
readonly EXIT_SIGTRAP=$((128 + 5))   # Trace trap
readonly EXIT_SIGABRT=$((128 + 6))   # Abort
readonly EXIT_SIGBUS=$((128 + 7))    # Bus error
readonly EXIT_SIGFPE=$((128 + 8))    # Floating point exception
readonly EXIT_SIGKILL=$((128 + 9))   # Kill (cannot be caught)
readonly EXIT_SIGUSR1=$((128 + 10))  # User defined signal 1
readonly EXIT_SIGSEGV=$((128 + 11))  # Segmentation fault
readonly EXIT_SIGUSR2=$((128 + 12))  # User defined signal 2
readonly EXIT_SIGPIPE=$((128 + 13))  # Broken pipe
readonly EXIT_SIGALRM=$((128 + 14))  # Alarm clock
readonly EXIT_SIGTERM=$((128 + 15))  # Termination

# -----------------------------------------------------------------------------
# CUSTOM APPLICATION EXIT CODES (redefine from constants.sh for clarity)
# -----------------------------------------------------------------------------
readonly EXIT_TIMEOUT=124                    # Command timeout (GNU timeout standard)
readonly EXIT_QUOTA_EXHAUSTED=142            # API quota or rate limit hit
# EXIT_CONFIGURATION_ERROR=150 (from constants.sh)
# EXIT_WORKER_UNAVAILABLE=151 (from constants.sh)
readonly EXIT_LOCK_FAILED=152                # Could not acquire lock
readonly EXIT_RESOURCE_LIMIT=153             # System resource limit hit (memory, disk)
readonly EXIT_NETWORK_ERROR=154              # Network connectivity issues
readonly EXIT_AUTH_ERROR=155                 # Authentication or permission denied
readonly EXIT_DATA_ERROR=156                 # Data validation or corruption
readonly EXIT_PARTIAL_SUCCESS=157            # Completed with warnings

# -----------------------------------------------------------------------------
# ERROR PATTERNS FOR CLASSIFICATION
# -----------------------------------------------------------------------------
readonly ERROR_PATTERN_COMMAND_NOT_FOUND="command not found|No such file or directory"
readonly ERROR_PATTERN_PERMISSION_DENIED="permission denied|access denied|Permission denied|Access denied"
readonly ERROR_PATTERN_TIMEOUT="timeout|timed out|Timeout"
readonly ERROR_PATTERN_QUOTA="quota|rate limit|too many requests|quota exhausted"
readonly ERROR_PATTERN_NETWORK="connection refused|network unreachable|Connection refused|Network unreachable|Could not resolve"
readonly ERROR_PATTERN_WORKER_CRASH="Segmentation fault|core dumped|Killed|Aborted"
readonly ERROR_PATTERN_LOCK="lock.*failed|already running|could not acquire lock"
readonly ERROR_PATTERN_RESOURCE="out of memory|no space left|resource temporarily unavailable"
readonly ERROR_PATTERN_AUTH="authentication|unauthorized|forbidden|401|403"
readonly ERROR_PATTERN_DATA="invalid.*data|corrupt|malformed|parse error|validation failed"

# Classify worker exit code based on output patterns
# Args: $1 - exit code, $2 - output file path (optional), $3 - context (optional, e.g., "iteration #5")
# Returns: classified exit code
# Outputs: classified exit code to stdout
error_codes::classify_worker_exit() {
    local exitc="${1:-1}"
    local output_file="${2:-}"
    local context="${3:-}"
    
    # If exit code is 0, no classification needed
    if [[ $exitc -eq 0 ]]; then
        echo "$exitc"
        return 0
    fi
    
    # Check for timeout exit code (standard timeout command exit code)
    if [[ $exitc -eq 124 ]]; then
        echo "$exitc"
        return 0
    fi
    
    # If no output file provided, return original exit code
    if [[ -z "$output_file" ]] || [[ ! -f "$output_file" ]]; then
        echo "$exitc"
        return 0
    fi
    
    # Classify based on output patterns
    local classified_code="$exitc"
    local classification_reason=""
    
    # Check patterns in priority order
    if grep -qE "$ERROR_PATTERN_COMMAND_NOT_FOUND" "$output_file" 2>/dev/null; then
        classified_code=$EXIT_WORKER_UNAVAILABLE
        classification_reason="worker unavailable (command not found)"
    elif grep -qE "$ERROR_PATTERN_AUTH" "$output_file" 2>/dev/null; then
        classified_code=$EXIT_AUTH_ERROR
        classification_reason="authentication/permission error"
    elif grep -qE "$ERROR_PATTERN_PERMISSION_DENIED" "$output_file" 2>/dev/null; then
        classified_code=$EXIT_AUTH_ERROR
        classification_reason="permission denied"
    elif grep -qE "$ERROR_PATTERN_QUOTA" "$output_file" 2>/dev/null; then
        classified_code=$EXIT_QUOTA_EXHAUSTED
        classification_reason="quota exhausted"
    elif grep -qE "$ERROR_PATTERN_NETWORK" "$output_file" 2>/dev/null; then
        classified_code=$EXIT_NETWORK_ERROR
        classification_reason="network connectivity error"
    elif grep -qE "$ERROR_PATTERN_WORKER_CRASH" "$output_file" 2>/dev/null; then
        classified_code=$EXIT_WORKER_UNAVAILABLE
        classification_reason="worker crashed"
    elif grep -qE "$ERROR_PATTERN_LOCK" "$output_file" 2>/dev/null; then
        classified_code=$EXIT_LOCK_FAILED
        classification_reason="lock acquisition failed"
    elif grep -qE "$ERROR_PATTERN_RESOURCE" "$output_file" 2>/dev/null; then
        classified_code=$EXIT_RESOURCE_LIMIT
        classification_reason="resource limit exceeded"
    elif grep -qE "$ERROR_PATTERN_DATA" "$output_file" 2>/dev/null; then
        classified_code=$EXIT_DATA_ERROR
        classification_reason="data validation/corruption error"
    fi
    
    # Log classification if it changed
    if [[ "$classified_code" != "$exitc" ]] && [[ -n "$context" ]]; then
        if declare -F log_with_timestamp >/dev/null 2>&1; then
            log_with_timestamp "WARNING: Exit code $exitc classified as $classified_code - $classification_reason ($context)"
        fi
    fi
    
    echo "$classified_code"
    return 0
}

# -----------------------------------------------------------------------------
# Function: error_codes::describe
# Description: Get human-readable error description
# Parameters:
#   $1 - Exit code to describe
# Returns: Description string to stdout
# Side Effects: None
# -----------------------------------------------------------------------------
error_codes::describe() {
    local code="${1:-0}"
    
    case "$code" in
        # Standard POSIX codes
        0) echo "success" ;;
        1) echo "general error" ;;
        2) echo "misuse of shell builtin" ;;
        126) echo "command cannot execute" ;;
        127) echo "command not found" ;;
        
        # Signal-based codes
        "$EXIT_SIGHUP") echo "terminal hangup" ;;
        "$EXIT_SIGINT") echo "interrupted (Ctrl+C)" ;;
        "$EXIT_SIGQUIT") echo "quit signal" ;;
        "$EXIT_SIGILL") echo "illegal instruction" ;;
        "$EXIT_SIGTRAP") echo "trace trap" ;;
        "$EXIT_SIGABRT") echo "aborted" ;;
        "$EXIT_SIGBUS") echo "bus error" ;;
        "$EXIT_SIGFPE") echo "floating point exception" ;;
        "$EXIT_SIGKILL") echo "killed (SIGKILL)" ;;
        "$EXIT_SIGUSR1") echo "user signal 1" ;;
        "$EXIT_SIGSEGV") echo "segmentation fault" ;;
        "$EXIT_SIGUSR2") echo "user signal 2" ;;
        "$EXIT_SIGPIPE") echo "broken pipe" ;;
        "$EXIT_SIGALRM") echo "alarm clock" ;;
        "$EXIT_SIGTERM") echo "terminated (SIGTERM)" ;;
        
        # Custom application codes
        "$EXIT_TIMEOUT") echo "timeout" ;;
        "$EXIT_QUOTA_EXHAUSTED") echo "quota exhausted" ;;
        "$EXIT_CONFIGURATION_ERROR") echo "configuration error" ;;
        "$EXIT_WORKER_UNAVAILABLE") echo "worker unavailable" ;;
        "$EXIT_LOCK_FAILED") echo "lock acquisition failed" ;;
        "$EXIT_RESOURCE_LIMIT") echo "resource limit exceeded" ;;
        "$EXIT_NETWORK_ERROR") echo "network error" ;;
        "$EXIT_AUTH_ERROR") echo "authentication error" ;;
        "$EXIT_DATA_ERROR") echo "data error" ;;
        "$EXIT_PARTIAL_SUCCESS") echo "partial success with warnings" ;;
        
        *) echo "unknown error ($code)" ;;
    esac
}

# -----------------------------------------------------------------------------
# Function: error_codes::is_retryable
# Description: Check if exit code represents a retryable error
# Parameters:
#   $1 - Exit code to check
# Returns: 0 if retryable, 1 if not retryable
# Side Effects: None
# -----------------------------------------------------------------------------
error_codes::is_retryable() {
    local code="${1:-0}"
    
    case "$code" in
        # Definitely retryable
        "$EXIT_TIMEOUT"|"$EXIT_QUOTA_EXHAUSTED"|"$EXIT_NETWORK_ERROR"|"$EXIT_RESOURCE_LIMIT")
            return 0 ;;
        
        # Possibly retryable after delay
        "$EXIT_LOCK_FAILED"|"$EXIT_PARTIAL_SUCCESS")
            return 0 ;;
        
        # Not retryable without intervention
        "$EXIT_CONFIGURATION_ERROR"|"$EXIT_WORKER_UNAVAILABLE"|"$EXIT_AUTH_ERROR")
            return 1 ;;
        
        # Fatal errors - not retryable
        "$EXIT_SIGSEGV"|"$EXIT_SIGBUS"|"$EXIT_SIGILL"|"$EXIT_DATA_ERROR")
            return 1 ;;
        
        # Command not found or cannot execute - not retryable
        126|127)
            return 1 ;;
        
        # Success - no need to retry
        0)
            return 1 ;;
        
        # Unknown errors - assume potentially retryable
        *)
            return 0 ;;
    esac
}

# Export functions for external use
export -f error_codes::classify_worker_exit
export -f error_codes::describe
export -f error_codes::is_retryable