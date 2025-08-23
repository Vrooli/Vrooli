#!/usr/bin/env bash
# Vrooli Test Assertions - Centralized Assertion Functions
# All test assertions consolidated in one place for consistency

# Prevent duplicate loading
if [[ "${VROOLI_ASSERTIONS_LOADED:-}" == "true" ]]; then
    return 0
fi
export VROOLI_ASSERTIONS_LOADED="true"

echo "[ASSERTIONS] Loading Vrooli test assertions"

#######################################
# Basic Assertions
#######################################

#######################################
# Assert command succeeded (exit code 0)
# Arguments: None (uses $status from run command)
# Returns: 0 on success, 1 on failure
#######################################
assert_success() {
    if [[ "${status:-1}" -eq 0 ]]; then
        return 0
    else
        echo "Expected success (exit code 0), got: ${status:-1}"
        echo "Output: ${output:-}"
        return 1
    fi
}

#######################################
# Assert command failed (exit code != 0)
# Arguments: None (uses $status from run command)
# Returns: 0 on success, 1 on failure
#######################################
assert_failure() {
    if [[ "${status:-0}" -ne 0 ]]; then
        return 0
    else
        echo "Expected failure (exit code != 0), got: ${status:-0}"
        echo "Output: ${output:-}"
        return 1
    fi
}

#######################################
# Assert specific exit code
# Arguments: $1 - expected exit code
# Returns: 0 on success, 1 on failure
#######################################
assert_exit_code() {
    local expected="$1"
    if [[ "${status:-1}" -eq "$expected" ]]; then
        return 0
    else
        echo "Expected exit code $expected, got: ${status:-1}"
        echo "Output: ${output:-}"
        return 1
    fi
}

#######################################
# String and Output Assertions
#######################################

#######################################
# Assert output contains specific text
# Arguments: $1 - expected text
# Returns: 0 on success, 1 on failure
#######################################
assert_output_contains() {
    local expected="$1"
    if [[ "${output:-}" == *"$expected"* ]]; then
        return 0
    else
        echo "Expected output to contain: '$expected'"
        echo "Actual output: '${output:-}'"
        return 1
    fi
}

#######################################
# Assert output does not contain specific text
# Arguments: $1 - text that should not be present
# Returns: 0 on success, 1 on failure
#######################################
assert_output_not_contains() {
    local unexpected="$1"
    if [[ ! "${output:-}" =~ $unexpected ]]; then
        return 0
    else
        echo "Expected output to NOT contain: '$unexpected'"
        echo "Actual output: '${output:-}'"
        return 1
    fi
}

#######################################
# Assert output equals specific text exactly
# Arguments: $1 - expected text
# Returns: 0 on success, 1 on failure
#######################################
assert_output_equals() {
    local expected="$1"
    if [[ "${output:-}" == "$expected" ]]; then
        return 0
    else
        echo "Expected output: '$expected'"
        echo "Actual output: '${output:-}'"
        return 1
    fi
}

#######################################
# Assert output matches a regex pattern
# Arguments: $1 - regex pattern
# Returns: 0 on success, 1 on failure
#######################################
assert_output_matches() {
    local pattern="$1"
    if [[ "${output:-}" =~ $pattern ]]; then
        return 0
    else
        echo "Output does not match pattern: '$pattern'"
        echo "Actual output: '${output:-}'"
        return 1
    fi
}

#######################################
# Assert output is empty
# Arguments: None
# Returns: 0 on success, 1 on failure
#######################################
assert_output_empty() {
    if [[ -z "${output:-}" ]]; then
        return 0
    else
        echo "Expected empty output, but got: '${output:-}'"
        return 1
    fi
}

#######################################
# Assert two strings are equal
# Arguments: $1 - actual value, $2 - expected value
# Returns: 0 on success, 1 on failure
#######################################
assert_equals() {
    local actual="$1"
    local expected="$2"
    if [[ "$actual" == "$expected" ]]; then
        return 0
    else
        echo "Expected: '$expected'"
        echo "Actual: '$actual'"
        return 1
    fi
}

#######################################
# File and Directory Assertions
#######################################

#######################################
# Assert file exists
# Arguments: $1 - file path
# Returns: 0 on success, 1 on failure
#######################################
assert_file_exists() {
    local file="$1"
    if [[ -f "$file" ]]; then
        return 0
    else
        echo "Expected file to exist: '$file'"
        return 1
    fi
}

#######################################
# Assert file does not exist
# Arguments: $1 - file path
# Returns: 0 on success, 1 on failure
#######################################
assert_file_not_exists() {
    local file="$1"
    if [[ ! -f "$file" ]]; then
        return 0
    else
        echo "Expected file to NOT exist: '$file'"
        return 1
    fi
}

#######################################
# Assert directory exists
# Arguments: $1 - directory path
# Returns: 0 on success, 1 on failure
#######################################
assert_dir_exists() {
    local dir="$1"
    if [[ -d "$dir" ]]; then
        return 0
    else
        echo "Expected directory to exist: '$dir'"
        return 1
    fi
}

#######################################
# Assert directory does not exist
# Arguments: $1 - directory path
# Returns: 0 on success, 1 on failure
#######################################
assert_dir_not_exists() {
    local dir="$1"
    if [[ ! -d "$dir" ]]; then
        return 0
    else
        echo "Expected directory to NOT exist: '$dir'"
        return 1
    fi
}

#######################################
# Assert file contains specific text
# Arguments: $1 - file path, $2 - expected text
# Returns: 0 on success, 1 on failure
#######################################
assert_file_contains() {
    local file="$1"
    local expected="$2"
    
    if [[ ! -f "$file" ]]; then
        echo "File does not exist: '$file'"
        return 1
    fi
    
    if grep -q "$expected" "$file"; then
        return 0
    else
        echo "Expected file '$file' to contain: '$expected'"
        echo "File contents:"
        cat "$file" | head -20
        return 1
    fi
}

#######################################
# Assert file does not contain specific text
# Arguments: $1 - file path, $2 - text that should not be present
# Returns: 0 on success, 1 on failure
#######################################
assert_file_not_contains() {
    local file="$1"
    local unexpected="$2"
    
    if [[ ! -f "$file" ]]; then
        echo "File does not exist: '$file'"
        return 1
    fi
    
    if ! grep -q "$unexpected" "$file"; then
        return 0
    else
        echo "File '$file' unexpectedly contains: '$unexpected'"
        echo "File contents:"
        cat "$file" | head -20
        return 1
    fi
}

#######################################
# Environment and Variable Assertions
#######################################

#######################################
# Assert environment variable is set
# Arguments: $1 - variable name
# Returns: 0 on success, 1 on failure
#######################################
assert_env_set() {
    local var="$1"
    if [[ -n "${!var:-}" ]]; then
        return 0
    else
        echo "Expected environment variable to be set: '$var'"
        return 1
    fi
}

#######################################
# Assert environment variable equals specific value
# Arguments: $1 - variable name, $2 - expected value
# Returns: 0 on success, 1 on failure
#######################################
assert_env_equals() {
    local var="$1"
    local expected="$2"
    local actual="${!var:-}"
    
    if [[ "$actual" == "$expected" ]]; then
        return 0
    else
        echo "Expected $var to equal: '$expected'"
        echo "Actual value: '$actual'"
        return 1
    fi
}

#######################################
# Assert environment variable is not set
# Arguments: $1 - variable name
# Returns: 0 on success, 1 on failure
#######################################
assert_env_not_set() {
    local var="$1"
    
    if [[ ! -v "$var" ]]; then
        return 0
    else
        echo "Expected $var to NOT be set"
        echo "Actual value: '${!var}'"
        return 1
    fi
}

#######################################
# JSON Assertions
#######################################

#######################################
# Assert string is valid JSON
# Arguments: $1 - JSON string
# Returns: 0 on success, 1 on failure
#######################################
assert_json_valid() {
    local json="$1"
    
    if command -v jq >/dev/null 2>&1; then
        if echo "$json" | jq . >/dev/null 2>&1; then
            return 0
        else
            echo "Invalid JSON: '$json'"
            return 1
        fi
    else
        # Fallback: basic JSON validation without jq
        if [[ "$json" =~ ^\{.*\}$ ]] || [[ "$json" =~ ^\[.*\]$ ]]; then
            return 0
        else
            echo "Invalid JSON format: '$json'"
            return 1
        fi
    fi
}

#######################################
# Assert JSON field equals specific value
# Arguments: $1 - JSON string, $2 - field path (e.g., ".status"), $3 - expected value
# Returns: 0 on success, 1 on failure
#######################################
assert_json_field_equals() {
    local json="$1"
    local field="$2"
    local expected="$3"
    
    if ! command -v jq >/dev/null 2>&1; then
        echo "jq not available, cannot test JSON field"
        return 1
    fi
    
    local actual
    actual=$(echo "$json" | jq -r "$field" 2>/dev/null)
    
    if [[ "$actual" == "$expected" ]]; then
        return 0
    else
        echo "Expected JSON field '$field' to equal: '$expected'"
        echo "Actual value: '$actual'"
        echo "JSON: '$json'"
        return 1
    fi
}

#######################################
# Assert JSON contains specific field
# Arguments: $1 - JSON string, $2 - field path
# Returns: 0 on success, 1 on failure
#######################################
assert_json_has_field() {
    local json="$1"
    local field="$2"
    
    if ! command -v jq >/dev/null 2>&1; then
        echo "jq not available, cannot test JSON field"
        return 1
    fi
    
    if echo "$json" | jq -e "$field" >/dev/null 2>&1; then
        return 0
    else
        echo "Expected JSON to have field: '$field'"
        echo "JSON: '$json'"
        return 1
    fi
}

#######################################
# Service and Resource Assertions
#######################################

#######################################
# Assert service is healthy (generic health check)
# Arguments: $1 - service name
# Returns: 0 on success, 1 on failure
#######################################
assert_service_healthy() {
    local service="$1"
    local port
    
    # Try to get port from configuration
    if command -v vrooli_config_get_port >/dev/null 2>&1; then
        port=$(vrooli_config_get_port "$service")
    fi
    
    # Try to get port from environment variable
    if [[ -z "$port" ]]; then
        local port_var="${service^^}_PORT"
        port_var="${port_var//-/_}"
        port="${!port_var:-}"
    fi
    
    if [[ -z "$port" ]]; then
        echo "Cannot determine port for service: $service"
        return 1
    fi
    
    # Check if service responds
    if curl -s "http://localhost:$port/health" >/dev/null 2>&1; then
        return 0
    else
        echo "Service not healthy: $service (port: $port)"
        return 1
    fi
}

#######################################
# Assert Docker container is running
# Arguments: $1 - container name
# Returns: 0 on success, 1 on failure
#######################################
assert_docker_container_running() {
    local container="$1"
    
    if docker ps --format "{{.Names}}" | grep -q "^${container}$"; then
        return 0
    else
        echo "Expected Docker container to be running: '$container'"
        echo "Running containers:"
        docker ps --format "table {{.Names}}\t{{.Status}}"
        return 1
    fi
}

#######################################
# Function Existence Assertions
#######################################

#######################################
# Assert function exists
# Arguments: $1 - function name
# Returns: 0 on success, 1 on failure
#######################################
assert_function_exists() {
    local func="$1"
    if declare -f "$func" >/dev/null 2>&1; then
        return 0
    else
        echo "Expected function to exist: '$func'"
        return 1
    fi
}

#######################################
# Assert command exists
# Arguments: $1 - command name
# Returns: 0 on success, 1 on failure
#######################################
assert_command_exists() {
    local cmd="$1"
    if command -v "$cmd" >/dev/null 2>&1; then
        return 0
    else
        echo "Expected command to exist: '$cmd'"
        return 1
    fi
}

#######################################
# Numeric Assertions
#######################################

#######################################
# Assert value is greater than
# Arguments: $1 - actual value, $2 - minimum value
# Returns: 0 on success, 1 on failure
#######################################
assert_greater_than() {
    local actual="$1"
    local minimum="$2"
    
    if [[ "$actual" =~ ^[0-9]+$ ]] && [[ "$minimum" =~ ^[0-9]+$ ]]; then
        if [[ "$actual" -gt "$minimum" ]]; then
            return 0
        else
            echo "Expected $actual to be greater than $minimum"
            return 1
        fi
    else
        echo "Non-numeric values provided: actual='$actual', minimum='$minimum'"
        return 1
    fi
}

#######################################
# Assert value is less than
# Arguments: $1 - actual value, $2 - maximum value
# Returns: 0 on success, 1 on failure
#######################################
assert_less_than() {
    local actual="$1"
    local maximum="$2"
    
    if [[ "$actual" =~ ^[0-9]+$ ]] && [[ "$maximum" =~ ^[0-9]+$ ]]; then
        if [[ "$actual" -lt "$maximum" ]]; then
            return 0
        else
            echo "Expected $actual to be less than $maximum"
            return 1
        fi
    else
        echo "Non-numeric values provided: actual='$actual', maximum='$maximum'"
        return 1
    fi
}

#######################################
# Export all assertion functions
#######################################
export -f assert_success assert_failure assert_exit_code
export -f assert_output_contains assert_output_not_contains assert_output_equals assert_equals
export -f assert_output_matches assert_output_empty
export -f assert_file_exists assert_file_not_exists assert_dir_exists assert_dir_not_exists
export -f assert_file_contains assert_file_not_contains
export -f assert_env_set assert_env_equals assert_env_not_set
export -f assert_json_valid assert_json_field_equals assert_json_has_field
export -f assert_service_healthy assert_docker_container_running
export -f assert_function_exists assert_command_exists
export -f assert_greater_than assert_less_than

echo "[ASSERTIONS] Vrooli test assertions loaded successfully"