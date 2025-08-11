#!/usr/bin/env bash
# n8n Shared Utility Functions
# Common patterns extracted from multiple n8n modules

# Source shared utilities
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../lib/docker-utils.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../lib/http-utils.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../lib/wait-utils.sh" 2>/dev/null || true

#######################################
# Unified API key resolution
# Uses 3-layer resolution: env variable, secrets file, vault
# Returns: API key via stdout, empty if not found
#######################################
n8n::resolve_api_key() {
    http::resolve_api_key "N8N_API_KEY"
}

#######################################
# Standardized HTTP calls with error handling
# Args: $1 - method, $2 - url, $3 - data (optional), $4 - headers (optional)
# Returns: Response body via stdout, HTTP code via return value
#######################################
n8n::safe_curl_call() {
    http::request "$@"
}

#######################################
# Get container environment variable safely
# Args: $1 - variable_name, $2 - container_name (optional, defaults to N8N_CONTAINER_NAME)
# Returns: Variable value or empty string
#######################################
n8n::extract_container_env() {
    local var_name="$1"
    local container_name="${2:-$N8N_CONTAINER_NAME}"
    docker::extract_env "$container_name" "$var_name"
}

#######################################
# Enhanced logging with context and consistent formatting
# Args: $1 - level (info|warn|error|success), $2 - context, $3 - message
#######################################
n8n::log_with_context() {
    local level="$1"
    local context="$2" 
    local message="$3"
    case "$level" in
        "info")
            log::info "[$context] $message"
            ;;
        "warn")
            log::warn "[$context] $message"
            ;;
        "error") 
            log::error "[$context] $message"
            ;;
        "success")
            log::success "[$context] $message"
            ;;
        *)
            log::info "[$context] $message"
            ;;
    esac
}

#######################################
# Validate JSON response from API calls
# Args: $1 - response body, $2 - expected_fields (space-separated)
# Returns: 0 if valid, 1 if invalid
#######################################
n8n::validate_json_response() {
    local response="$1"
    local expected_fields="$2"
    
    if ! http::validate_json "$response" "$expected_fields"; then
        n8n::log_with_context "error" "api" "JSON validation failed"
        return 1
    fi
    
    # Check for error messages in response (n8n specific)
    if echo "$response" | jq -e '.message' >/dev/null 2>&1; then
        local error_msg
        error_msg=$(echo "$response" | jq -r '.message')
        n8n::log_with_context "error" "api" "API error: $error_msg"
        return 1
    fi
    
    return 0
}

#######################################
# Extract ID from API response (common pattern)
# Args: $1 - response body
# Returns: ID value or empty string
#######################################
n8n::extract_response_id() {
    local response="$1"
    # Try .id first, then .data.id
    local id
    id=$(http::extract_json_field "$response" ".id")
    if [[ -n "$id" ]]; then
        echo "$id"
    else
        http::extract_json_field "$response" ".data.id"
    fi
}

#######################################
# Check if API key is configured and valid
# Returns: 0 if valid, 1 if invalid/missing
#######################################
n8n::validate_api_key_setup() {
    local api_key
    api_key=$(n8n::resolve_api_key)
    if [[ -z "$api_key" ]]; then
        n8n::log_with_context "error" "api" "No API key found"
        return 1
    fi
    # Test the API key with a simple request
    local response
    local http_code
    response=$(n8n::safe_curl_call "GET" "${N8N_BASE_URL}/api/v1/workflows?limit=1" "" "X-N8N-API-KEY: $api_key")
    http_code=$?
    case "$http_code" in
        200)
            n8n::log_with_context "success" "api" "API key is valid"
            return 0
            ;;
        401)
            n8n::log_with_context "error" "api" "API key is invalid or expired"
            return 1
            ;;
        403)
            n8n::log_with_context "error" "api" "API key lacks necessary permissions"
            return 1
            ;;
        *)
            n8n::log_with_context "error" "api" "API endpoint not responding (HTTP $http_code)"
            return 1
            ;;
    esac
}

#######################################
# Wait for condition with timeout and progress indication
# Args: $1 - test_command, $2 - timeout_seconds, $3 - description
# Returns: 0 if condition met, 1 if timeout
#######################################
n8n::wait_for_condition() {
    wait::for_condition "$@"
}

#######################################
# Show standardized API setup instructions
#######################################
n8n::show_api_setup_instructions() {
    echo
    n8n::log_with_context "info" "setup" "To set up API access:"
    echo "  1. Access n8n at $N8N_BASE_URL"
    echo "  2. Go to Settings → n8n API"
    echo "  3. Click 'Create an API key'"
    echo "  4. Set a label (e.g., 'CLI Access')"
    echo "  5. Copy the generated API key"
    echo
    echo "Then save it with:"
    echo "  $0 --action save-api-key --api-key YOUR_KEY"
    echo "  OR: export N8N_API_KEY=your_key"
    echo
    n8n::log_with_context "warn" "setup" "The API key is shown only once - make sure to copy it!"
}

#######################################
# Unified error handling with context and suggestions
# Args: $1 - context, $2 - error_msg, $3 - suggestion (optional), $4 - exit_code (optional)
# Returns: exit_code or 1
#######################################
n8n::handle_error() {
    local context="$1"
    local error_msg="$2" 
    local suggestion="${3:-}"
    local exit_code="${4:-1}"
    n8n::log_with_context "error" "$context" "$error_msg"
    if [[ -n "$suggestion" ]]; then
        n8n::log_with_context "info" "fix" "$suggestion"
    fi
    return "$exit_code"
}

#######################################
# Enhanced error handling with automatic diagnosis
# Args: $1 - context, $2 - error_msg, $3 - log_output (optional)
# Returns: 1 (always indicates error)
#######################################
n8n::handle_error_with_diagnosis() {
    local context="$1"
    local error_msg="$2"
    local log_output="${3:-}"
    n8n::log_with_context "error" "$context" "$error_msg"
    # Try to diagnose and suggest fixes if log output provided
    if [[ -n "$log_output" ]]; then
        if ! n8n::detect_and_suggest_fix "$log_output" >/dev/null; then
            n8n::log_with_context "info" "diagnosis" "Check logs for more details"
        fi
    fi
    return 1
}

#######################################
# Rollback context management
# Args: $1 - operation_name
#######################################
n8n::start_operation_context() {
    local operation="$1"
    export N8N_CURRENT_OPERATION="$operation"
    N8N_OPERATION_ROLLBACK_ACTIONS=()
    n8n::log_with_context "info" "operation" "Starting: $operation"
}

#######################################
# Add rollback action with operation context
# Args: $1 - description, $2 - command, $3 - priority (optional)
#######################################
n8n::add_rollback_with_context() {
    local description="$1"
    local command="$2"
    local priority="${3:-10}"
    local full_description="[$N8N_CURRENT_OPERATION] $description"
    # Use the standard resources rollback system if available
    if command -v resources::add_rollback_action &>/dev/null; then
        resources::add_rollback_action "$full_description" "$command" "$priority"
    else
        # Fallback to local tracking
        N8N_OPERATION_ROLLBACK_ACTIONS+=("$full_description: $command")
    fi
    n8n::log_with_context "debug" "rollback" "Added: $description"
}

#######################################
# Execute operation rollback
#######################################
n8n::execute_operation_rollback() {
    if [[ -n "${N8N_CURRENT_OPERATION:-}" ]]; then
        n8n::log_with_context "warn" "rollback" "Executing rollback for: $N8N_CURRENT_OPERATION"
        # Use standard resources rollback if available
        if command -v resources::execute_rollback &>/dev/null; then
            resources::execute_rollback
        else
            # Fallback to local rollback
            for action in "${N8N_OPERATION_ROLLBACK_ACTIONS[@]:-}"; do
                local cmd="${action#*: }"
                n8n::log_with_context "info" "rollback" "Executing: ${action%%: *}"
                eval "$cmd" 2>/dev/null || true
            done
        fi
        # Clear context
        unset N8N_CURRENT_OPERATION
        N8N_OPERATION_ROLLBACK_ACTIONS=()
    fi
}

#######################################
# Complete operation successfully
#######################################
n8n::complete_operation() {
    if [[ -n "${N8N_CURRENT_OPERATION:-}" ]]; then
        n8n::log_with_context "success" "operation" "Completed: $N8N_CURRENT_OPERATION"
        # Clear rollback context on success
        if command -v resources::clear_rollback_context &>/dev/null; then
            ROLLBACK_ACTIONS=()
            OPERATION_ID=""
        fi
        unset N8N_CURRENT_OPERATION
        N8N_OPERATION_ROLLBACK_ACTIONS=()
    fi
}

#######################################
# Detect and handle common n8n error patterns
# Args: $1 - log_output or error_message
# Returns: 0 if handled, 1 if unknown error  
#######################################
n8n::detect_and_suggest_fix() {
    local error_output="$1"
    # Database corruption patterns
    if echo "$error_output" | grep -qi "SQLITE_READONLY\|database.*locked\|database.*corrupted"; then
        n8n::log_with_context "error" "diagnosis" "Database corruption detected"
        n8n::log_with_context "info" "fix" "Try: $0 --action restart (includes automatic recovery)"
        return 0
    fi
    # Permission errors
    if echo "$error_output" | grep -qi "EACCES\|permission denied"; then
        n8n::log_with_context "error" "diagnosis" "Permission errors detected"
        n8n::log_with_context "info" "fix" "Check data directory permissions: $N8N_DATA_DIR"
        return 0
    fi
    # Port binding issues
    if echo "$error_output" | grep -qi "port.*already.*use\|bind.*address.*already.*use"; then
        n8n::log_with_context "error" "diagnosis" "Port binding conflict detected"
        n8n::log_with_context "info" "fix" "Check if port $N8N_PORT is already in use: lsof -i :$N8N_PORT"
        return 0
    fi
    # Docker daemon issues
    if echo "$error_output" | grep -qi "cannot connect.*docker daemon\|docker.*not running"; then
        n8n::log_with_context "error" "diagnosis" "Docker daemon not accessible"
        n8n::log_with_context "info" "fix" "Start Docker: sudo systemctl start docker"
        return 0
    fi
    return 1  # Unknown error pattern
}

#######################################
# Docker container helper functions
# Consolidates repeated docker patterns
#######################################

# Check if container is running
n8n::container_running() {
    local container_name="${1:-$N8N_CONTAINER_NAME}"
    docker::is_running "$container_name"
}

# Check if container exists (running or stopped)
n8n::container_exists_any() {
    local container_name="${1:-$N8N_CONTAINER_NAME}"
    docker::container_exists "$container_name"
}

# Check if postgres container is running
n8n::postgres_running() {
    docker::is_running "$N8N_DB_CONTAINER_NAME"
}

# Check if postgres container exists
n8n::postgres_exists() {
    docker::container_exists "$N8N_DB_CONTAINER_NAME"
}

# Unified error check wrapper - reduces if/then/return patterns
n8n::require() {
    if ! "$@"; then
        n8n::log_with_context "error" "requirement" "${*} failed"
        return 1
    fi
    return 0
}
