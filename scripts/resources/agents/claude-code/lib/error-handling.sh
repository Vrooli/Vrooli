#!/usr/bin/env bash
# Claude Code Centralized Error Handling and Recovery
# Provides unified error interpretation, recovery strategies, and consistent error messaging

# Error codes and their meanings
declare -A CLAUDE_CODE_ERRORS=(
    ["0"]="SUCCESS"
    ["1"]="GENERAL_ERROR" 
    ["2"]="INVALID_ARGUMENTS"
    ["3"]="PERMISSION_DENIED"
    ["4"]="NETWORK_ERROR"
    ["5"]="TIMEOUT_ERROR"
    ["6"]="AUTHENTICATION_ERROR"
    ["7"]="RATE_LIMIT_ERROR"
    ["8"]="SESSION_ERROR"
    ["9"]="TOOL_ERROR"
    ["10"]="RESOURCE_ERROR"
    ["error_max_turns"]="SUCCESS_MAX_TURNS_REACHED"
    ["cancelled"]="USER_CANCELLED"
    ["interrupted"]="PROCESS_INTERRUPTED"
)

# Recovery strategies for different error types
declare -A RECOVERY_STRATEGIES=(
    ["NETWORK_ERROR"]="retry_with_backoff"
    ["TIMEOUT_ERROR"]="increase_timeout_and_retry"
    ["RATE_LIMIT_ERROR"]="wait_and_retry"
    ["SESSION_ERROR"]="create_new_session"
    ["TOOL_ERROR"]="retry_with_different_tools"
    ["RESOURCE_ERROR"]="check_resources_and_retry"
    ["AUTHENTICATION_ERROR"]="refresh_auth_and_retry"
)

# Default retry configurations
DEFAULT_MAX_RETRIES=3
DEFAULT_BASE_DELAY=5
DEFAULT_MAX_DELAY=300
DEFAULT_BACKOFF_MULTIPLIER=2

#######################################
# Parse and interpret Claude Code exit codes
# Arguments:
#   $1 - Exit code or status string
#   $2 - Command that generated the exit code (optional)
# Outputs: Error interpretation and recommendations
# Returns: Normalized exit code (0 for success, 1 for error)
#######################################
claude_code::error_interpret() {
    local exit_code="$1"
    local command="${2:-claude_code}"
    
    # Handle special string codes
    case "$exit_code" in
        "error_max_turns"|"SUCCESS_MAX_TURNS_REACHED")
            log::success "✅ Task completed: Maximum turns reached successfully"
            log::info "This typically indicates the task was completed within the turn limit"
            return 0
            ;;
        "cancelled"|"USER_CANCELLED")
            log::warn "⚠️  Task cancelled by user"
            log::info "Operation was intentionally cancelled"
            return 1
            ;;
        "interrupted"|"PROCESS_INTERRUPTED")
            log::error "❌ Process interrupted unexpectedly"
            log::info "This may be due to system signal or resource constraints"
            return 1
            ;;
    esac
    
    # Handle numeric exit codes
    local error_type="${CLAUDE_CODE_ERRORS[$exit_code]:-UNKNOWN_ERROR}"
    
    case "$error_type" in
        "SUCCESS")
            log::success "✅ $command completed successfully"
            return 0
            ;;
        "GENERAL_ERROR")
            log::error "❌ General error in $command"
            log::info "Check logs for specific error details"
            ;;
        "INVALID_ARGUMENTS")
            log::error "❌ Invalid arguments provided to $command"
            log::info "Review command syntax and parameter requirements"
            ;;
        "PERMISSION_DENIED")
            log::error "❌ Permission denied for $command"
            log::info "Check file permissions and user privileges"
            ;;
        "NETWORK_ERROR")
            log::error "❌ Network error during $command"
            log::info "Check internet connection and firewall settings"
            ;;
        "TIMEOUT_ERROR")
            log::error "❌ Timeout during $command execution"
            log::info "Consider increasing timeout values or simplifying the task"
            ;;
        "AUTHENTICATION_ERROR")
            log::error "❌ Authentication failed for $command"
            log::info "Check API keys and authentication credentials"
            ;;
        "RATE_LIMIT_ERROR")
            log::error "❌ Rate limit exceeded for $command"
            log::info "Wait before retrying or reduce request frequency"
            ;;
        "SESSION_ERROR")
            log::error "❌ Session error during $command"
            log::info "Session may have expired or become corrupted"
            ;;
        "TOOL_ERROR")
            log::error "❌ Tool execution error in $command"
            log::info "Check tool permissions and availability"
            ;;
        "RESOURCE_ERROR")
            log::error "❌ Resource error during $command"
            log::info "Check system resources (disk space, memory, etc.)"
            ;;
        *)
            log::error "❌ Unknown error ($exit_code) in $command"
            log::info "This may be a new error type or system-specific issue"
            ;;
    esac
    
    return 1
}

#######################################
# Execute command with automatic retry and recovery
# Arguments:
#   $1 - Command to execute
#   $2 - Max retries (optional, default: DEFAULT_MAX_RETRIES)
#   $3 - Recovery strategy (optional, auto-detected)
# Returns: Final exit code after all retry attempts
#######################################
claude_code::error_retry() {
    local command="$1"
    local max_retries="${2:-$DEFAULT_MAX_RETRIES}"
    local recovery_strategy="$3"
    
    local attempt=1
    local delay=$DEFAULT_BASE_DELAY
    
    log::info "🔄 Executing with retry: $command"
    log::info "Max retries: $max_retries"
    
    while [[ $attempt -le $((max_retries + 1)) ]]; do
        if [[ $attempt -gt 1 ]]; then
            log::info "🔄 Retry attempt $((attempt - 1))/$max_retries"
            
            # Apply recovery strategy if specified
            if [[ -n "$recovery_strategy" ]]; then
                claude_code::error_apply_recovery "$recovery_strategy" "$attempt"
            fi
            
            # Wait with exponential backoff
            log::info "⏳ Waiting ${delay}s before retry..."
            sleep "$delay"
            delay=$((delay * DEFAULT_BACKOFF_MULTIPLIER))
            if [[ $delay -gt $DEFAULT_MAX_DELAY ]]; then
                delay=$DEFAULT_MAX_DELAY
            fi
        fi
        
        # Execute command and capture exit code
        local exit_code
        eval "$command"
        exit_code=$?
        
        # Interpret the result
        if claude_code::error_interpret "$exit_code" "$command"; then
            if [[ $attempt -gt 1 ]]; then
                log::success "✅ Command succeeded after $((attempt - 1)) retries"
            fi
            return 0
        fi
        
        # Determine recovery strategy if not provided
        if [[ -z "$recovery_strategy" ]]; then
            recovery_strategy=$(claude_code::error_suggest_recovery "$exit_code")
        fi
        
        # Check if we should continue retrying
        if [[ $attempt -le $max_retries ]] && claude_code::error_should_retry "$exit_code"; then
            log::warn "⚠️  Attempt $attempt failed, will retry..."
        else
            break
        fi
        
        attempt=$((attempt + 1))
    done
    
    if [[ $attempt -gt $((max_retries + 1)) ]]; then
        log::error "❌ Command failed after $max_retries retries: $command"
    else
        log::error "❌ Command failed (not retryable): $command"
    fi
    
    return 1
}

#######################################
# Determine if an error should trigger a retry
# Arguments:
#   $1 - Exit code or error type
# Returns: 0 if should retry, 1 if should not retry
#######################################
claude_code::error_should_retry() {
    local exit_code="$1"
    local error_type="${CLAUDE_CODE_ERRORS[$exit_code]:-UNKNOWN_ERROR}"
    
    case "$error_type" in
        "NETWORK_ERROR"|"TIMEOUT_ERROR"|"RATE_LIMIT_ERROR"|"SESSION_ERROR"|"RESOURCE_ERROR")
            return 0  # Retryable errors
            ;;
        "AUTHENTICATION_ERROR")
            return 0  # Retryable if we can refresh auth
            ;;
        "INVALID_ARGUMENTS"|"PERMISSION_DENIED"|"USER_CANCELLED")
            return 1  # Not retryable
            ;;
        *)
            return 0  # Default to retrying unknown errors
            ;;
    esac
}

#######################################
# Suggest recovery strategy for an error
# Arguments:
#   $1 - Exit code or error type
# Outputs: Suggested recovery strategy
#######################################
claude_code::error_suggest_recovery() {
    local exit_code="$1"
    local error_type="${CLAUDE_CODE_ERRORS[$exit_code]:-UNKNOWN_ERROR}"
    
    echo "${RECOVERY_STRATEGIES[$error_type]:-retry_simple}"
}

#######################################
# Apply recovery strategy before retry
# Arguments:
#   $1 - Recovery strategy
#   $2 - Retry attempt number
#######################################
claude_code::error_apply_recovery() {
    local strategy="$1"
    local attempt="$2"
    
    case "$strategy" in
        "retry_with_backoff")
            # Already handled by main retry loop
            ;;
        "increase_timeout_and_retry")
            log::info "🔧 Applying recovery: Increasing timeout values"
            export DEFAULT_TIMEOUT_MS=$((DEFAULT_TIMEOUT_MS * 2))
            export MAX_TIMEOUT_MS=$((MAX_TIMEOUT_MS * 2))
            ;;
        "wait_and_retry")
            local wait_time=$((attempt * 60))  # Wait longer for rate limits
            log::info "🔧 Applying recovery: Extended wait for rate limiting (${wait_time}s)"
            sleep "$wait_time"
            ;;
        "create_new_session")
            log::info "🔧 Applying recovery: Creating new session"
            # Clear any existing session variables
            unset CLAUDE_SESSION_ID CLAUDE_SESSION_FILE
            ;;
        "retry_with_different_tools")
            log::info "🔧 Applying recovery: Reducing tool permissions"
            # This would require context about current tools being used
            log::warn "Tool restriction recovery requires manual intervention"
            ;;
        "check_resources_and_retry")
            log::info "🔧 Applying recovery: Checking system resources"
            claude_code::error_check_resources
            ;;
        "refresh_auth_and_retry")
            log::info "🔧 Applying recovery: Refreshing authentication"
            # This would require context about authentication method
            log::warn "Authentication refresh requires manual intervention"
            ;;
        "retry_simple")
            # No special recovery needed
            ;;
        *)
            log::warn "Unknown recovery strategy: $strategy"
            ;;
    esac
}

#######################################
# Check system resources and report issues
# Returns: 0 if resources are adequate, 1 if issues found
#######################################
claude_code::error_check_resources() {
    local issues=0
    
    # Check disk space
    local disk_usage
    disk_usage=$(df . | awk 'NR==2 {print $5}' | sed 's/%//')
    if [[ $disk_usage -gt 90 ]]; then
        log::error "💾 Disk space critically low: ${disk_usage}% used"
        issues=$((issues + 1))
    elif [[ $disk_usage -gt 80 ]]; then
        log::warn "💾 Disk space getting low: ${disk_usage}% used"
    fi
    
    # Check memory usage
    if command -v free >/dev/null 2>&1; then
        local mem_usage
        mem_usage=$(free | awk 'NR==2{printf "%.0f", $3/$2*100}')
        if [[ $mem_usage -gt 90 ]]; then
            log::error "🧠 Memory usage critically high: ${mem_usage}%"
            issues=$((issues + 1))
        elif [[ $mem_usage -gt 80 ]]; then
            log::warn "🧠 Memory usage getting high: ${mem_usage}%"
        fi
    fi
    
    # Check temporary directory space
    if [[ -n "${TMPDIR:-}" ]] && [[ -d "$TMPDIR" ]]; then
        local tmp_usage
        tmp_usage=$(df "$TMPDIR" | awk 'NR==2 {print $5}' | sed 's/%//')
        if [[ $tmp_usage -gt 90 ]]; then
            log::error "📁 Temporary directory space critically low: ${tmp_usage}%"
            issues=$((issues + 1))
        fi
    fi
    
    # Check for zombie processes
    local zombies
    zombies=$(ps aux | awk '$8 ~ /^Z/ { count++ } END { print count+0 }')
    if [[ $zombies -gt 10 ]]; then
        log::warn "🧟 High number of zombie processes: $zombies"
    fi
    
    if [[ $issues -eq 0 ]]; then
        log::info "✅ System resources appear adequate"
        return 0
    else
        log::error "❌ Found $issues critical resource issues"
        return 1
    fi
}

#######################################
# Safe wrapper for Claude Code execution with full error handling
# Arguments:
#   $1 - Prompt or command
#   $2 - Allowed tools (optional)
#   $3 - Max turns (optional)
#   $4 - Max retries (optional)
# Returns: 0 on success, 1 on failure after all retries
#######################################
claude_code::safe_execute() {
    local prompt="$1"
    local allowed_tools="${2:-Read,Edit,Write}"
    local max_turns="${3:-20}"
    local max_retries="${4:-$DEFAULT_MAX_RETRIES}"
    
    log::header "🛡️ Safe Claude Code Execution"
    log::info "Prompt: ${prompt:0:100}..."
    log::info "Tools: $allowed_tools"
    log::info "Max turns: $max_turns"
    log::info "Max retries: $max_retries"
    
    # Pre-execution checks
    if ! claude_code::error_check_resources; then
        log::warn "⚠️  Resource issues detected, proceeding with caution"
    fi
    
    # Construct command
    local cmd="claude_code::run \"$prompt\" \"$allowed_tools\" \"$max_turns\""
    
    # Execute with retry and recovery
    claude_code::error_retry "$cmd" "$max_retries"
}

#######################################
# Generate error report for debugging
# Arguments:
#   $1 - Error context (command, operation, etc.)
#   $2 - Exit code
#   $3 - Additional details (optional)
# Outputs: Formatted error report
#######################################
claude_code::error_report() {
    local context="$1"
    local exit_code="$2"
    local details="${3:-}"
    local timestamp=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
    
    log::header "🔍 Error Report"
    echo "Timestamp: $timestamp"
    echo "Context: $context"
    echo "Exit Code: $exit_code"
    echo "Error Type: ${CLAUDE_CODE_ERRORS[$exit_code]:-UNKNOWN_ERROR}"
    echo "Recovery Strategy: $(claude_code::error_suggest_recovery "$exit_code")"
    echo "Retryable: $(claude_code::error_should_retry "$exit_code" && echo "Yes" || echo "No")"
    
    if [[ -n "$details" ]]; then
        echo "Additional Details:"
        echo "$details" | sed 's/^/  /'
    fi
    
    echo
    echo "System Information:"
    echo "  Disk Usage: $(df . | awk 'NR==2 {print $5}')"
    if command -v free >/dev/null 2>&1; then
        echo "  Memory Usage: $(free | awk 'NR==2{printf "%.0f%%", $3/$2*100}')"
    fi
    echo "  Load Average: $(uptime | awk -F'load average:' '{print $2}')"
    
    echo
    echo "Environment:"
    echo "  DEFAULT_TIMEOUT_MS: ${DEFAULT_TIMEOUT_MS:-not set}"
    echo "  MAX_TIMEOUT_MS: ${MAX_TIMEOUT_MS:-not set}"
    echo "  MCP_TOOL_TIMEOUT: ${MCP_TOOL_TIMEOUT:-not set}"
    echo "  CLAUDE_SESSION_ID: ${CLAUDE_SESSION_ID:-not set}"
}

#######################################
# Validate error handling configuration
# Returns: 0 if configuration is valid, 1 if issues found
#######################################
claude_code::error_validate_config() {
    local issues=0
    
    log::header "🔧 Validating Error Handling Configuration"
    
    # Check timeout values
    if [[ -n "${DEFAULT_TIMEOUT_MS:-}" ]]; then
        if [[ ! "$DEFAULT_TIMEOUT_MS" =~ ^[0-9]+$ ]] || [[ $DEFAULT_TIMEOUT_MS -lt 1000 ]]; then
            log::error "Invalid DEFAULT_TIMEOUT_MS: $DEFAULT_TIMEOUT_MS (should be >= 1000)"
            issues=$((issues + 1))
        fi
    fi
    
    if [[ -n "${MAX_TIMEOUT_MS:-}" ]]; then
        if [[ ! "$MAX_TIMEOUT_MS" =~ ^[0-9]+$ ]] || [[ $MAX_TIMEOUT_MS -lt 5000 ]]; then
            log::error "Invalid MAX_TIMEOUT_MS: $MAX_TIMEOUT_MS (should be >= 5000)"
            issues=$((issues + 1))
        fi
    fi
    
    # Check retry configuration
    if [[ ! "$DEFAULT_MAX_RETRIES" =~ ^[0-9]+$ ]] || [[ $DEFAULT_MAX_RETRIES -lt 1 ]]; then
        log::error "Invalid DEFAULT_MAX_RETRIES: $DEFAULT_MAX_RETRIES (should be >= 1)"
        issues=$((issues + 1))
    fi
    
    if [[ ! "$DEFAULT_BASE_DELAY" =~ ^[0-9]+$ ]] || [[ $DEFAULT_BASE_DELAY -lt 1 ]]; then
        log::error "Invalid DEFAULT_BASE_DELAY: $DEFAULT_BASE_DELAY (should be >= 1)"
        issues=$((issues + 1))
    fi
    
    # Check required commands
    local required_commands=("jq" "awk" "sed" "grep" "df" "ps")
    for cmd in "${required_commands[@]}"; do
        if ! command -v "$cmd" >/dev/null 2>&1; then
            log::error "Required command not found: $cmd"
            issues=$((issues + 1))
        fi
    done
    
    if [[ $issues -eq 0 ]]; then
        log::success "✅ Error handling configuration is valid"
        return 0
    else
        log::error "❌ Found $issues configuration issues"
        return 1
    fi
}

# Export functions for external use
export -f claude_code::error_interpret
export -f claude_code::error_retry
export -f claude_code::error_should_retry
export -f claude_code::error_suggest_recovery
export -f claude_code::error_apply_recovery
export -f claude_code::error_check_resources
export -f claude_code::safe_execute
export -f claude_code::error_report
export -f claude_code::error_validate_config