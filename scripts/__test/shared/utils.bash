#!/usr/bin/env bash
# Vrooli Test Utilities - Common Helper Functions
# Shared utilities used across the testing infrastructure

# Prevent duplicate loading
if [[ "${VROOLI_UTILS_LOADED:-}" == "true" ]]; then
    return 0
fi
export VROOLI_UTILS_LOADED="true"

echo "[UTILS] Loading Vrooli test utilities" >&2

#######################################
# Path and Directory Utilities
#######################################

#######################################
# Get the root directory of the test suite
# Returns: absolute path to scripts/__test/
#######################################
vrooli_test_root() {
    if [[ -n "${VROOLI_TEST_ROOT:-}" ]]; then
        echo "$VROOLI_TEST_ROOT"
    else
        local script_dir
        script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
        echo "$(dirname "$script_dir")"
    fi
}

#######################################
# Create a unique temporary directory
# Arguments: $1 - optional prefix (defaults to "vrooli_test")
# Returns: absolute path to created directory
#######################################
vrooli_make_temp_dir() {
    local prefix="${1:-vrooli_test}"
    local timestamp
    timestamp=$(date +%s)
    local random
    random=$RANDOM
    
    local tmpdir_base
    if command -v vrooli_config_get >/dev/null 2>&1; then
        tmpdir_base=$(vrooli_config_get "environment.tmpdir_base" "/tmp")
    else
        tmpdir_base="/tmp"
    fi
    
    local temp_dir="${tmpdir_base}/${prefix}_${timestamp}_${random}"
    
    if mkdir -p "$temp_dir"; then
        echo "$temp_dir"
    else
        echo "/tmp/${prefix}_${timestamp}_${random}"
        mkdir -p "/tmp/${prefix}_${timestamp}_${random}"
    fi
}

#######################################
# Find files matching a pattern
# Arguments: $1 - directory, $2 - pattern
# Returns: list of matching files
#######################################
vrooli_find_files() {
    local directory="$1"
    local pattern="$2"
    
    if [[ ! -d "$directory" ]]; then
        echo "[UTILS] ERROR: Directory not found: $directory" >&2
        return 1
    fi
    
    find "$directory" -name "$pattern" -type f 2>/dev/null | sort
}

#######################################
# String and Text Utilities
#######################################

#######################################
# Generate a random string
# Arguments: $1 - length (defaults to 8)
# Returns: random alphanumeric string
#######################################
vrooli_random_string() {
    local length="${1:-8}"
    tr -dc 'a-zA-Z0-9' < /dev/urandom | head -c "$length" 2>/dev/null || echo "test$(date +%s | tail -c 4)"
}

#######################################
# Convert string to uppercase
# Arguments: $1 - string to convert
# Returns: uppercase string
#######################################
vrooli_to_upper() {
    echo "$1" | tr '[:lower:]' '[:upper:]'
}

#######################################
# Convert string to lowercase
# Arguments: $1 - string to convert
# Returns: lowercase string
#######################################
vrooli_to_lower() {
    echo "$1" | tr '[:upper:]' '[:lower:]'
}

#######################################
# Sanitize string for use as identifier
# Arguments: $1 - string to sanitize
# Returns: sanitized string (alphanumeric + underscore only)
#######################################
vrooli_sanitize_identifier() {
    echo "$1" | sed 's/[^a-zA-Z0-9_]/_/g' | sed 's/__*/_/g' | sed 's/^_//;s/_$//'
}

#######################################
# Time and Date Utilities
#######################################

#######################################
# Get current timestamp
# Arguments: $1 - format (optional: "iso", "epoch", "human")
# Returns: formatted timestamp
#######################################
vrooli_timestamp() {
    local format="${1:-iso}"
    
    case "$format" in
        "iso")
            date -u +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || date +"%Y-%m-%d %H:%M:%S"
            ;;
        "epoch")
            date +%s
            ;;
        "human")
            date +"%Y-%m-%d %H:%M:%S"
            ;;
        *)
            date +"%Y%m%d_%H%M%S"
            ;;
    esac
}

#######################################
# Measure execution time of a command
# Arguments: $@ - command to execute
# Returns: execution time in seconds
#######################################
vrooli_time_command() {
    local start_time
    start_time=$(date +%s.%N 2>/dev/null || date +%s)
    
    "$@"
    local exit_code=$?
    
    local end_time
    end_time=$(date +%s.%N 2>/dev/null || date +%s)
    
    # Calculate duration
    if command -v bc >/dev/null 2>&1; then
        echo "$end_time - $start_time" | bc 2>/dev/null || echo "1"
    else
        # Fallback for systems without bc
        local duration=$((end_time - start_time))
        echo "${duration:-1}"
    fi
    
    return $exit_code
}

#######################################
# Process and System Utilities
#######################################

#######################################
# Check if a process is running
# Arguments: $1 - process name or PID
# Returns: 0 if running, 1 if not
#######################################
vrooli_process_running() {
    local process="$1"
    
    if [[ "$process" =~ ^[0-9]+$ ]]; then
        # It's a PID
        kill -0 "$process" 2>/dev/null
    else
        # It's a process name
        pgrep -x "$process" >/dev/null 2>&1
    fi
}

#######################################
# Wait for a process to finish
# Arguments: $1 - PID, $2 - timeout in seconds (optional, default 30)
# Returns: 0 if process finished, 1 if timeout
#######################################
vrooli_wait_for_process() {
    local pid="$1"
    local timeout="${2:-30}"
    local elapsed=0
    
    while [[ $elapsed -lt $timeout ]]; do
        if ! kill -0 "$pid" 2>/dev/null; then
            return 0
        fi
        sleep 1
        ((elapsed++))
    done
    
    return 1
}

#######################################
# Get available port
# Arguments: $1 - starting port (optional, default 8000)
# Returns: available port number
#######################################
vrooli_get_available_port() {
    local start_port="${1:-8000}"
    local port=$start_port
    
    while [[ $port -lt 65535 ]]; do
        if ! ss -ln 2>/dev/null | grep -q ":${port} " && ! netstat -ln 2>/dev/null | grep -q ":${port} "; then
            echo "$port"
            return 0
        fi
        ((port++))
    done
    
    # Fallback: return start_port + random offset
    echo $((start_port + RANDOM % 1000))
}

#######################################
# Network and HTTP Utilities
#######################################

#######################################
# Wait for service to be ready
# Arguments: $1 - URL, $2 - timeout in seconds (optional, default 30)
# Returns: 0 if service ready, 1 if timeout
#######################################
vrooli_wait_for_service() {
    local url="$1"
    local timeout="${2:-30}"
    local elapsed=0
    
    echo "[UTILS] Waiting for service: $url (timeout: ${timeout}s)"
    
    while [[ $elapsed -lt $timeout ]]; do
        if curl -s -o /dev/null -w "%{http_code}" "$url" 2>/dev/null | grep -q "^[2-3][0-9][0-9]$"; then
            echo "[UTILS] Service ready: $url (${elapsed}s)"
            return 0
        fi
        sleep 2
        ((elapsed+=2))
    done
    
    echo "[UTILS] Service timeout: $url (${elapsed}s)"
    return 1
}

#######################################
# Check if URL is reachable
# Arguments: $1 - URL
# Returns: 0 if reachable, 1 if not
#######################################
vrooli_url_reachable() {
    local url="$1"
    curl -s -o /dev/null -w "%{http_code}" "$url" 2>/dev/null | grep -q "^[2-3][0-9][0-9]$"
}

#######################################
# Make HTTP request with retry
# Arguments: $1 - URL, $2 - method (optional, default GET), $3 - data (optional), $4 - retries (optional, default 3)
# Returns: 0 on success, 1 on failure
#######################################
vrooli_http_request() {
    local url="$1"
    local method="${2:-GET}"
    local data="${3:-}"
    local retries="${4:-3}"
    
    local attempt=1
    while [[ $attempt -le $retries ]]; do
        local curl_args=("-s" "-X" "$method")
        
        if [[ -n "$data" ]]; then
            curl_args+=("-d" "$data" "-H" "Content-Type: application/json")
        fi
        
        if curl "${curl_args[@]}" "$url" 2>/dev/null; then
            return 0
        fi
        
        echo "[UTILS] HTTP request failed (attempt $attempt/$retries): $method $url" >&2
        ((attempt++))
        
        if [[ $attempt -le $retries ]]; then
            sleep $((attempt - 1))
        fi
    done
    
    return 1
}

#######################################
# Validation Utilities
#######################################

#######################################
# Validate required parameter
# Arguments: $1 - parameter name, $2 - parameter value, $3 - context (optional)
# Returns: 0 if valid, 1 if invalid
#######################################
vrooli_validate_required() {
    local param_name="$1"
    local param_value="$2"
    local context="${3:-}"
    
    if [[ -z "$param_value" ]]; then
        echo "[UTILS] ERROR: Required parameter missing: $param_name${context:+ (context: $context)}" >&2
        return 1
    fi
    
    return 0
}

#######################################
# Validate function exists
# Arguments: $1 - function name, $2 - context (optional)
# Returns: 0 if exists, 1 if not
#######################################
vrooli_validate_function() {
    local func_name="$1"
    local context="${2:-}"
    
    if ! declare -f "$func_name" >/dev/null 2>&1; then
        echo "[UTILS] ERROR: Function not found: $func_name${context:+ (context: $context)}" >&2
        return 1
    fi
    
    return 0
}

#######################################
# Validate command exists
# Arguments: $1 - command name, $2 - context (optional)
# Returns: 0 if exists, 1 if not
#######################################
vrooli_validate_command() {
    local cmd_name="$1"
    local context="${2:-}"
    
    if ! command -v "$cmd_name" >/dev/null 2>&1; then
        echo "[UTILS] ERROR: Command not found: $cmd_name${context:+ (context: $context)}" >&2
        return 1
    fi
    
    return 0
}

#######################################
# File and JSON Utilities
#######################################

#######################################
# Read JSON field from file
# Arguments: $1 - file path, $2 - field path (jq syntax)
# Returns: field value or empty string
#######################################
vrooli_json_get_field() {
    local file="$1"
    local field="$2"
    
    if [[ ! -f "$file" ]]; then
        echo ""
        return 1
    fi
    
    if command -v jq >/dev/null 2>&1; then
        jq -r "$field" "$file" 2>/dev/null || echo ""
    else
        echo ""
    fi
}

#######################################
# Write JSON field to file
# Arguments: $1 - file path, $2 - field path, $3 - value
# Returns: 0 on success, 1 on failure
#######################################
vrooli_json_set_field() {
    local file="$1"
    local field="$2"
    local value="$3"
    
    if ! command -v jq >/dev/null 2>&1; then
        echo "[UTILS] ERROR: jq not available for JSON manipulation" >&2
        return 1
    fi
    
    local temp_file
    temp_file=$(mktemp)
    
    if [[ -f "$file" ]]; then
        if jq "$field = \"$value\"" "$file" > "$temp_file" 2>/dev/null; then
            mv "$temp_file" "$file"
            return 0
        fi
    else
        echo "{}" | jq "$field = \"$value\"" > "$temp_file" 2>/dev/null
        if [[ $? -eq 0 ]]; then
            mv "$temp_file" "$file"
            return 0
        fi
    fi
    
    rm -f "$temp_file"
    return 1
}

#######################################
# Export utility functions
#######################################
export -f vrooli_test_root vrooli_make_temp_dir vrooli_find_files
export -f vrooli_random_string vrooli_to_upper vrooli_to_lower vrooli_sanitize_identifier
export -f vrooli_timestamp vrooli_time_command
export -f vrooli_process_running vrooli_wait_for_process vrooli_get_available_port
export -f vrooli_wait_for_service vrooli_url_reachable vrooli_http_request
export -f vrooli_validate_required vrooli_validate_function vrooli_validate_command
export -f vrooli_json_get_field vrooli_json_set_field

echo "[UTILS] Vrooli test utilities loaded successfully" >&2