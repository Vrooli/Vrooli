#!/usr/bin/env bash
# Standardized Error Handling Framework for BATS Test Infrastructure
# Provides consistent error reporting and handling across all modules

# Prevent duplicate loading
if [[ "${ERROR_HANDLING_LOADED:-}" == "true" ]]; then
    return 0
fi
export ERROR_HANDLING_LOADED="true"

# Error handling configuration
export ERROR_LOG_ENABLED="${ERROR_LOG_ENABLED:-true}"
export ERROR_STACK_TRACE_ENABLED="${ERROR_STACK_TRACE_ENABLED:-false}"
export ERROR_LOG_FILE="${ERROR_LOG_FILE:-${MOCK_RESPONSES_DIR:-/tmp}/error.log}"

#######################################
# Standardized error reporting function
# Arguments:
#   $1 - module name (e.g., "COMMON_SETUP", "MOCK_REGISTRY", "ASSERTIONS")
#   $2 - error level (ERROR, WARNING, INFO)
#   $3 - error message
#   $4 - optional context/details
# Returns: 1 for ERROR, 0 for WARNING/INFO
#######################################
report_error() {
    local module="$1"
    local level="$2"
    local message="$3"
    local context="${4:-}"
    
    # Format error message consistently
    local formatted_msg="[$module] $level: $message"
    if [[ -n "$context" ]]; then
        formatted_msg="$formatted_msg ($context)"
    fi
    
    # Output to stderr
    echo "$formatted_msg" >&2
    
    # Log to file if enabled
    if [[ "$ERROR_LOG_ENABLED" == "true" && -n "${ERROR_LOG_FILE:-}" ]]; then
        local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
        echo "[$timestamp] $formatted_msg" >> "$ERROR_LOG_FILE" 2>/dev/null || true
    fi
    
    # Add stack trace for errors if enabled
    if [[ "$level" == "ERROR" && "$ERROR_STACK_TRACE_ENABLED" == "true" ]]; then
        print_stack_trace >&2
    fi
    
    # Return appropriate exit code
    case "$level" in
        "ERROR") return 1 ;;
        "WARNING"|"INFO") return 0 ;;
        *) return 1 ;;
    esac
}

#######################################
# Print stack trace for debugging
#######################################
print_stack_trace() {
    local frame=0
    echo "Stack trace:" >&2
    while caller $frame >&2; do
        ((frame++))
    done
}

#######################################
# Common error reporting functions for each module
#######################################

# Common Setup errors
common_setup_error() {
    report_error "COMMON_SETUP" "ERROR" "$1" "${2:-}"
}

common_setup_warning() {
    report_error "COMMON_SETUP" "WARNING" "$1" "${2:-}"
}

common_setup_info() {
    report_error "COMMON_SETUP" "INFO" "$1" "${2:-}"
}

# Mock Registry errors
mock_registry_error() {
    report_error "MOCK_REGISTRY" "ERROR" "$1" "${2:-}"
}

mock_registry_warning() {
    report_error "MOCK_REGISTRY" "WARNING" "$1" "${2:-}"
}

mock_registry_info() {
    report_error "MOCK_REGISTRY" "INFO" "$1" "${2:-}"
}

# Assertion errors
assertion_error() {
    report_error "ASSERTIONS" "ERROR" "$1" "${2:-}"
}

assertion_warning() {
    report_error "ASSERTIONS" "WARNING" "$1" "${2:-}"
}

assertion_info() {
    report_error "ASSERTIONS" "INFO" "$1" "${2:-}"
}

# HTTP Mock errors
http_mock_error() {
    report_error "HTTP_MOCKS" "ERROR" "$1" "${2:-}"
}

http_mock_warning() {
    report_error "HTTP_MOCKS" "WARNING" "$1" "${2:-}"
}

http_mock_info() {
    report_error "HTTP_MOCKS" "INFO" "$1" "${2:-}"
}

# Docker Mock errors
docker_mock_error() {
    report_error "DOCKER_MOCKS" "ERROR" "$1" "${2:-}"
}

docker_mock_warning() {
    report_error "DOCKER_MOCKS" "WARNING" "$1" "${2:-}"
}

docker_mock_info() {
    report_error "DOCKER_MOCKS" "INFO" "$1" "${2:-}"
}

# Generic resource mock errors
resource_mock_error() {
    local resource="$1"
    shift
    report_error "${resource^^}_MOCK" "ERROR" "$1" "${2:-}"
}

resource_mock_warning() {
    local resource="$1"
    shift
    report_error "${resource^^}_MOCK" "WARNING" "$1" "${2:-}"
}

resource_mock_info() {
    local resource="$1"
    shift
    report_error "${resource^^}_MOCK" "INFO" "$1" "${2:-}"
}

#######################################
# Validation functions with standardized errors
#######################################

# Validate required parameter
validate_required_param() {
    local param_name="$1"
    local param_value="$2"
    local caller_module="${3:-VALIDATION}"
    
    if [[ -z "$param_value" ]]; then
        report_error "$caller_module" "ERROR" "Required parameter missing: $param_name"
        return 1
    fi
    
    return 0
}

# Validate file exists
validate_file_exists() {
    local file_path="$1"
    local caller_module="${2:-VALIDATION}"
    
    if [[ ! -f "$file_path" ]]; then
        report_error "$caller_module" "ERROR" "Required file does not exist: $file_path"
        return 1
    fi
    
    return 0
}

# Validate directory exists
validate_dir_exists() {
    local dir_path="$1"
    local caller_module="${2:-VALIDATION}"
    
    if [[ ! -d "$dir_path" ]]; then
        report_error "$caller_module" "ERROR" "Required directory does not exist: $dir_path"
        return 1
    fi
    
    return 0
}

# Validate command exists
validate_command_exists() {
    local command="$1"
    local caller_module="${2:-VALIDATION}"
    
    if ! command -v "$command" >/dev/null 2>&1; then
        report_error "$caller_module" "ERROR" "Required command not found: $command"
        return 1
    fi
    
    return 0
}

# Validate function exists
validate_function_exists() {
    local function_name="$1"
    local caller_module="${2:-VALIDATION}"
    
    if ! declare -f "$function_name" >/dev/null 2>&1; then
        report_error "$caller_module" "ERROR" "Required function not available: $function_name"
        return 1
    fi
    
    return 0
}

# Validate environment variable is set
validate_env_var() {
    local var_name="$1"
    local caller_module="${2:-VALIDATION}"
    
    if [[ -z "${!var_name:-}" ]]; then
        report_error "$caller_module" "ERROR" "Required environment variable not set: $var_name"
        return 1
    fi
    
    return 0
}

#######################################
# Error recovery and cleanup functions
#######################################

# Execute with error recovery
execute_with_recovery() {
    local recovery_function="$1"
    shift
    local command=("$@")
    
    # Try to execute the command
    if "${command[@]}"; then
        return 0
    else
        local exit_code=$?
        report_error "RECOVERY" "WARNING" "Command failed, attempting recovery: ${command[*]}"
        
        # Execute recovery function if provided
        if [[ -n "$recovery_function" ]] && declare -f "$recovery_function" >/dev/null 2>&1; then
            if "$recovery_function"; then
                report_error "RECOVERY" "INFO" "Recovery successful, retrying command"
                # Retry the command once
                if "${command[@]}"; then
                    return 0
                else
                    report_error "RECOVERY" "ERROR" "Command failed even after recovery"
                    return $exit_code
                fi
            else
                report_error "RECOVERY" "ERROR" "Recovery function failed"
                return $exit_code
            fi
        else
            report_error "RECOVERY" "ERROR" "No recovery function available"
            return $exit_code
        fi
    fi
}

# Safe cleanup with error handling
safe_cleanup() {
    local cleanup_function="$1"
    local caller_module="${2:-CLEANUP}"
    
    if [[ -n "$cleanup_function" ]] && declare -f "$cleanup_function" >/dev/null 2>&1; then
        if ! "$cleanup_function"; then
            report_error "$caller_module" "WARNING" "Cleanup function failed: $cleanup_function"
            return 1
        fi
    else
        report_error "$caller_module" "WARNING" "Cleanup function not available: $cleanup_function"
        return 1
    fi
    
    return 0
}

#######################################
# Error aggregation for batch operations
#######################################

# Initialize error collection
init_error_collection() {
    export ERROR_COLLECTION_ENABLED="true"
    export ERROR_COLLECTION_FILE="${MOCK_RESPONSES_DIR:-/tmp}/collected_errors.log"
    > "$ERROR_COLLECTION_FILE"  # Clear the file
}

# Collect error without immediately failing
collect_error() {
    local module="$1"
    local message="$2"
    local context="${3:-}"
    
    if [[ "${ERROR_COLLECTION_ENABLED:-}" == "true" ]]; then
        local formatted_msg="[$module] ERROR: $message"
        if [[ -n "$context" ]]; then
            formatted_msg="$formatted_msg ($context)"
        fi
        echo "$formatted_msg" >> "$ERROR_COLLECTION_FILE"
        return 0  # Don't fail immediately
    else
        # Fall back to normal error reporting
        report_error "$module" "ERROR" "$message" "$context"
    fi
}

# Report all collected errors and fail if any exist
report_collected_errors() {
    if [[ "${ERROR_COLLECTION_ENABLED:-}" == "true" && -f "$ERROR_COLLECTION_FILE" ]]; then
        local error_count
        error_count=$(wc -l < "$ERROR_COLLECTION_FILE")
        
        if [[ "$error_count" -gt 0 ]]; then
            echo "Collected $error_count errors during batch operation:" >&2
            cat "$ERROR_COLLECTION_FILE" >&2
            return 1
        fi
    fi
    
    return 0
}

# Clear error collection
clear_error_collection() {
    export ERROR_COLLECTION_ENABLED="false"
    if [[ -f "${ERROR_COLLECTION_FILE:-}" ]]; then
        rm -f "$ERROR_COLLECTION_FILE"
    fi
}

#######################################
# Export all functions
#######################################
export -f report_error print_stack_trace
export -f common_setup_error common_setup_warning common_setup_info
export -f mock_registry_error mock_registry_warning mock_registry_info
export -f assertion_error assertion_warning assertion_info
export -f http_mock_error http_mock_warning http_mock_info
export -f docker_mock_error docker_mock_warning docker_mock_info
export -f resource_mock_error resource_mock_warning resource_mock_info
export -f validate_required_param validate_file_exists validate_dir_exists
export -f validate_command_exists validate_function_exists validate_env_var
export -f execute_with_recovery safe_cleanup
export -f init_error_collection collect_error report_collected_errors clear_error_collection

echo "[ERROR_HANDLING] Standardized error handling framework loaded"