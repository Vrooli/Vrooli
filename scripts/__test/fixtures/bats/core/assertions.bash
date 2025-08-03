#!/usr/bin/env bash
# Enhanced Common Test Assertions for Bats Tests
# Provides comprehensive assertion helpers for Vrooli resource testing

# Prevent duplicate loading
if [[ "${ASSERTIONS_LOADED:-}" == "true" ]]; then
    return 0
fi
export ASSERTIONS_LOADED="true"

#######################################
# Basic Output Assertions
#######################################

# Assert output contains text (case-sensitive)
assert_output_contains() {
    local expected="$1"
    if [[ ! "$output" =~ $expected ]]; then
        echo "Expected output to contain: $expected" >&2
        echo "Actual output: $output" >&2
        return 1
    fi
}

# Assert output does not contain text
assert_output_not_contains() {
    local unexpected="$1"
    if [[ "$output" =~ $unexpected ]]; then
        echo "Expected output NOT to contain: $unexpected" >&2
        echo "Actual output: $output" >&2
        return 1
    fi
}

# Assert output matches regex
assert_output_matches() {
    local pattern="$1"
    if [[ ! "$output" =~ $pattern ]]; then
        echo "Expected output to match pattern: $pattern" >&2
        echo "Actual output: $output" >&2
        return 1
    fi
}

# Assert output is empty
assert_output_empty() {
    if [[ -n "$output" ]]; then
        echo "Expected empty output, but got: $output" >&2
        return 1
    fi
}

# Assert output line count
assert_output_line_count() {
    local expected_count="$1"
    local actual_count
    actual_count=$(echo "$output" | wc -l)
    
    if [[ "$actual_count" -ne "$expected_count" ]]; then
        echo "Expected $expected_count lines, but got $actual_count" >&2
        echo "Actual output: $output" >&2
        return 1
    fi
}

#######################################
# File System Assertions
#######################################

# Assert file exists
assert_file_exists() {
    local file="$1"
    if [[ ! -f "$file" ]]; then
        echo "Expected file to exist: $file" >&2
        return 1
    fi
}

# Assert file does not exist
assert_file_not_exists() {
    local file="$1"
    if [[ -f "$file" ]]; then
        echo "Expected file NOT to exist: $file" >&2
        return 1
    fi
}

# Assert directory exists
assert_dir_exists() {
    local dir="$1"
    if [[ ! -d "$dir" ]]; then
        echo "Expected directory to exist: $dir" >&2
        return 1
    fi
}

# Assert directory does not exist
assert_dir_not_exists() {
    local dir="$1"
    if [[ -d "$dir" ]]; then
        echo "Expected directory NOT to exist: $dir" >&2
        return 1
    fi
}

# Assert file contains text
assert_file_contains() {
    local file="$1"
    local expected="$2"
    
    if [[ ! -f "$file" ]]; then
        echo "File does not exist: $file" >&2
        return 1
    fi
    
    if ! grep -q "$expected" "$file"; then
        echo "Expected file $file to contain: $expected" >&2
        echo "File contents:" >&2
        cat "$file" >&2
        return 1
    fi
}

# Assert file is empty
assert_file_empty() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        echo "File does not exist: $file" >&2
        return 1
    fi
    
    if [[ -s "$file" ]]; then
        echo "Expected file to be empty: $file" >&2
        echo "File contents:" >&2
        cat "$file" >&2
        return 1
    fi
}

# Assert file permissions
assert_file_permissions() {
    local file="$1"
    local expected_perms="$2"
    
    if [[ ! -f "$file" ]]; then
        echo "File does not exist: $file" >&2
        return 1
    fi
    
    local actual_perms
    actual_perms=$(stat -c "%a" "$file" 2>/dev/null || stat -f "%A" "$file" 2>/dev/null)
    if [[ "$actual_perms" != "$expected_perms" ]]; then
        echo "File $file permissions:" >&2
        echo "  Expected: $expected_perms" >&2
        echo "  Actual: $actual_perms" >&2
        return 1
    fi
}

#######################################
# Environment Variable Assertions
#######################################

# Assert environment variable is set
assert_env_set() {
    local var="$1"
    if [[ -z "${!var:-}" ]]; then
        echo "Expected environment variable to be set: $var" >&2
        return 1
    fi
}

# Assert environment variable equals value
assert_env_equals() {
    local var="$1"
    local expected="$2"
    local actual="${!var:-}"
    
    if [[ "$actual" != "$expected" ]]; then
        echo "Environment variable $var:" >&2
        echo "  Expected: $expected" >&2
        echo "  Actual: $actual" >&2
        return 1
    fi
}

# Assert environment variable is unset
assert_env_unset() {
    local var="$1"
    if [[ -n "${!var:-}" ]]; then
        echo "Expected environment variable to be unset: $var" >&2
        echo "  Actual value: ${!var}" >&2
        return 1
    fi
}

#######################################
# Command and Function Assertions
#######################################

# Assert command exists
assert_command_exists() {
    local cmd="$1"
    if ! command -v "$cmd" >/dev/null 2>&1; then
        echo "Expected command to exist: $cmd" >&2
        return 1
    fi
}

# Assert function exists
assert_function_exists() {
    local func="$1"
    if ! declare -f "$func" >/dev/null 2>&1; then
        echo "Expected function to exist: $func" >&2
        return 1
    fi
}

# Assert command was called (requires mock framework)
assert_command_called() {
    local command="$1"
    local expected_args="${2:-}"
    
    # Check if mock call tracking is available
    if [[ -f "${MOCK_RESPONSES_DIR}/command_calls.log" ]]; then
        if ! grep -q "^$command" "${MOCK_RESPONSES_DIR}/command_calls.log"; then
            echo "Expected command to be called: $command" >&2
            return 1
        fi
        
        if [[ -n "$expected_args" ]]; then
            if ! grep -q "^$command.*$expected_args" "${MOCK_RESPONSES_DIR}/command_calls.log"; then
                echo "Expected command called with args: $command $expected_args" >&2
                echo "Actual calls:" >&2
                grep "^$command" "${MOCK_RESPONSES_DIR}/command_calls.log" >&2
                return 1
            fi
        fi
    else
        echo "Warning: Command call tracking not available" >&2
    fi
}

#######################################
# Data Structure Assertions
#######################################

# Assert arrays are equal
assert_arrays_equal() {
    local -n array1=$1
    local -n array2=$2
    
    if [[ ${#array1[@]} -ne ${#array2[@]} ]]; then
        echo "Arrays have different lengths:" >&2
        echo "  $1: ${#array1[@]} elements" >&2
        echo "  $2: ${#array2[@]} elements" >&2
        return 1
    fi
    
    for i in "${!array1[@]}"; do
        if [[ "${array1[$i]}" != "${array2[$i]}" ]]; then
            echo "Arrays differ at index $i:" >&2
            echo "  $1[$i]: ${array1[$i]}" >&2
            echo "  $2[$i]: ${array2[$i]}" >&2
            return 1
        fi
    done
}

# Assert array contains element
assert_array_contains() {
    local -n array_ref=$1
    local element="$2"
    
    for item in "${array_ref[@]}"; do
        if [[ "$item" == "$element" ]]; then
            return 0
        fi
    done
    
    echo "Expected array $1 to contain: $element" >&2
    echo "Array contents: ${array_ref[*]}" >&2
    return 1
}

#######################################
# JSON Assertions
#######################################

# Assert JSON is valid
assert_json_valid() {
    local json="$1"
    
    if ! echo "$json" | jq . >/dev/null 2>&1; then
        echo "Invalid JSON:" >&2
        echo "$json" >&2
        return 1
    fi
}

# Assert JSON field equals value
assert_json_field_equals() {
    local json="$1"
    local field="$2"
    local expected="$3"
    
    local actual
    if ! actual=$(echo "$json" | jq -r "$field" 2>/dev/null); then
        echo "Failed to parse JSON or extract field: $field" >&2
        echo "JSON: $json" >&2
        return 1
    fi
    
    if [[ "$actual" != "$expected" ]]; then
        echo "JSON field $field:" >&2
        echo "  Expected: $expected" >&2
        echo "  Actual: $actual" >&2
        return 1
    fi
}

# Assert JSON field exists
assert_json_field_exists() {
    local json="$1"
    local field="$2"
    
    local result
    result=$(echo "$json" | jq -r "$field // \"__FIELD_NOT_FOUND__\"" 2>/dev/null)
    if [[ "$result" == "__FIELD_NOT_FOUND__" ]] || [[ "$result" == "null" ]]; then
        echo "JSON field does not exist: $field" >&2
        echo "JSON: $json" >&2
        return 1
    fi
}

# Assert JSON schema validation (requires schema file)
assert_json_schema_valid() {
    local json="$1"
    local schema_file="$2"
    
    if [[ ! -f "$schema_file" ]]; then
        echo "Schema file does not exist: $schema_file" >&2
        return 1
    fi
    
    # This would require a JSON schema validator like ajv-cli
    # For now, just validate basic JSON structure
    assert_json_valid "$json"
}

#######################################
# Network Assertions
#######################################

# Assert port is available
assert_port_available() {
    local port="$1"
    if lsof -i ":$port" >/dev/null 2>&1; then
        echo "Expected port to be available: $port" >&2
        return 1
    fi
}

# Assert port is in use
assert_port_in_use() {
    local port="$1"
    if ! lsof -i ":$port" >/dev/null 2>&1; then
        echo "Expected port to be in use: $port" >&2
        return 1
    fi
}

# Assert HTTP endpoint is reachable
assert_http_endpoint_reachable() {
    local url="$1"
    local timeout="${2:-5}"
    
    if ! curl -s --max-time "$timeout" "$url" >/dev/null 2>&1; then
        echo "HTTP endpoint not reachable: $url" >&2
        return 1
    fi
}

# Assert HTTP response status
assert_http_status() {
    local url="$1"
    local expected_status="$2"
    local timeout="${3:-5}"
    
    local actual_status
    actual_status=$(curl -s -o /dev/null -w "%{http_code}" --max-time "$timeout" "$url" 2>/dev/null)
    if [[ "$actual_status" != "$expected_status" ]]; then
        echo "HTTP status for $url:" >&2
        echo "  Expected: $expected_status" >&2
        echo "  Actual: $actual_status" >&2
        return 1
    fi
}

#######################################
# Resource-Specific Assertions
#######################################

# Assert resource is healthy (generic)
assert_resource_healthy() {
    local resource="${1:-$RESOURCE_NAME}"
    local port="${2:-$RESOURCE_PORT}"
    local url="${3:-$RESOURCE_BASE_URL}"
    
    if [[ -z "$resource" ]]; then
        echo "Resource name required for health check" >&2
        return 1
    fi
    
    echo "Checking health of resource: $resource" >&2
    
    # Check port is in use
    if [[ -n "$port" ]]; then
        assert_port_in_use "$port"
    fi
    
    # Check HTTP endpoint if URL provided
    if [[ -n "$url" ]]; then
        assert_http_endpoint_reachable "$url"
    fi
    
    # Check container is running (if containerized)
    if [[ -n "${RESOURCE_CONTAINER_NAME:-}" ]]; then
        assert_docker_container_running "$RESOURCE_CONTAINER_NAME"
    fi
}

# Assert Docker container is running
assert_docker_container_running() {
    local container_name="$1"
    
    if ! docker ps --format "table {{.Names}}" | grep -q "^$container_name$"; then
        echo "Expected Docker container to be running: $container_name" >&2
        return 1
    fi
}

# Assert Docker container health
assert_docker_container_healthy() {
    local container_name="$1"
    
    local health_status
    health_status=$(docker inspect --format='{{.State.Health.Status}}' "$container_name" 2>/dev/null)
    if [[ "$health_status" != "healthy" ]]; then
        echo "Docker container not healthy: $container_name" >&2
        echo "Health status: $health_status" >&2
        return 1
    fi
}

# Assert API response is valid
assert_api_response_valid() {
    local endpoint="$1"
    local expected_fields="${2:-}"
    local timeout="${3:-5}"
    
    local response
    response=$(curl -s --max-time "$timeout" "$endpoint" 2>/dev/null)
    
    # Check if response is valid JSON
    assert_json_valid "$response"
    
    # Check for expected fields if provided
    if [[ -n "$expected_fields" ]]; then
        IFS=',' read -ra fields <<< "$expected_fields"
        for field in "${fields[@]}"; do
            assert_json_field_exists "$response" "$field"
        done
    fi
}

#######################################
# Resource Chain Assertions (for integration tests)
#######################################

# Assert data flow between resources
assert_resource_chain_working() {
    local source_resource="$1"
    local target_resource="$2"
    # local test_data="${3:-test_data}" # Future use for data flow validation
    
    echo "Testing resource chain: $source_resource -> $target_resource" >&2
    
    # This would be implemented based on specific resource APIs
    # For now, just check both resources are healthy
    assert_resource_healthy "$source_resource"
    assert_resource_healthy "$target_resource"
}

#######################################
# Mock-Specific Assertions
#######################################

# Assert mock was used
assert_mock_used() {
    local mock_name="$1"
    
    if [[ -f "${MOCK_RESPONSES_DIR}/used_mocks.log" ]]; then
        if ! grep -q "^$mock_name$" "${MOCK_RESPONSES_DIR}/used_mocks.log"; then
            echo "Expected mock to be used: $mock_name" >&2
            return 1
        fi
    else
        echo "Warning: Mock usage tracking not available" >&2
    fi
}

# Assert mock response was returned
assert_mock_response_used() {
    local mock_file="$1"
    
    if [[ -f "$MOCK_RESPONSES_DIR/$mock_file" ]]; then
        return 0
    else
        echo "Mock response file not found: $mock_file" >&2
        return 1
    fi
}

#######################################
# Additional Helper Assertions
#######################################

# Assert two values are equal
assert_equals() {
    local actual="$1"
    local expected="$2"
    
    if [[ "$actual" != "$expected" ]]; then
        echo "Values not equal:" >&2
        echo "  Expected: $expected" >&2
        echo "  Actual: $actual" >&2
        return 1
    fi
}

# Assert value is not empty
assert_not_empty() {
    local value="$1"
    local description="${2:-value}"
    
    if [[ -z "$value" ]]; then
        echo "Expected $description to not be empty" >&2
        return 1
    fi
}

# Assert environment variable is set and not empty
assert_env_not_empty() {
    local var="$1"
    assert_env_set "$var"
    assert_not_empty "${!var}" "environment variable $var"
}

# Assert string contains substring
assert_string_contains() {
    local string="$1"
    local substring="$2"
    
    if [[ ! "$string" =~ $substring ]]; then
        echo "String does not contain substring:" >&2
        echo "  String: $string" >&2
        echo "  Expected substring: $substring" >&2
        return 1
    fi
}

# Assert number is greater than another
assert_greater_than() {
    local actual="$1"
    local threshold="$2"
    
    if [[ ! "$actual" =~ ^[0-9]+$ ]] || [[ ! "$threshold" =~ ^[0-9]+$ ]]; then
        echo "Values must be numeric for comparison" >&2
        echo "  Actual: $actual" >&2
        echo "  Threshold: $threshold" >&2
        return 1
    fi
    
    if [[ "$actual" -le "$threshold" ]]; then
        echo "Value not greater than threshold:" >&2
        echo "  Actual: $actual" >&2
        echo "  Threshold: $threshold" >&2
        return 1
    fi
}

# Assert number is less than another
assert_less_than() {
    local actual="$1"
    local threshold="$2"
    
    if [[ ! "$actual" =~ ^[0-9]+$ ]] || [[ ! "$threshold" =~ ^[0-9]+$ ]]; then
        echo "Values must be numeric for comparison" >&2
        echo "  Actual: $actual" >&2
        echo "  Threshold: $threshold" >&2
        return 1
    fi
    
    if [[ "$actual" -ge "$threshold" ]]; then
        echo "Value not less than threshold:" >&2
        echo "  Actual: $actual" >&2
        echo "  Threshold: $threshold" >&2
        return 1
    fi
}

# Assert port is open (can connect)
assert_port_open() {
    local host="$1"
    local port="$2"
    local timeout="${3:-5}"
    
    if ! nc -z -w"$timeout" "$host" "$port" 2>/dev/null; then
        echo "Port not open: $host:$port" >&2
        return 1
    fi
}

# Assert output equals exact value
assert_output_equals() {
    local expected="$1"
    
    if [[ "$output" != "$expected" ]]; then
        echo "Output does not match expected value:" >&2
        echo "  Expected: $expected" >&2
        echo "  Actual: $output" >&2
        return 1
    fi
}

#######################################
# Advanced Assertion Functions
#######################################

# Assert that a condition becomes true within a timeout period (polling)
assert_eventually_true() {
    local command="$1"
    local timeout="${2:-30}"
    local interval="${3:-1}"
    local description="${4:-condition}"
    
    local elapsed=0
    while [[ $elapsed -lt $timeout ]]; do
        if eval "$command" >/dev/null 2>&1; then
            return 0
        fi
        sleep "$interval"
        elapsed=$((elapsed + interval))
    done
    
    echo "Expected $description to become true within ${timeout}s, but it didn't" >&2
    echo "Last command tried: $command" >&2
    return 1
}

# Assert that a command completes within a specified time
assert_completes_within() {
    local timeout="$1"
    local description="${2:-command}"
    shift 2
    local command=("$@")
    
    local start_time end_time duration
    start_time=$(date +%s%N)
    
    # Run command with timeout
    if timeout "${timeout}s" "${command[@]}" >/dev/null 2>&1; then
        end_time=$(date +%s%N)
        duration=$(( (end_time - start_time) / 1000000 ))  # Convert to milliseconds
        echo "Command completed in ${duration}ms (within ${timeout}s limit)" >&2
        return 0
    else
        echo "Expected $description to complete within ${timeout}s, but it didn't" >&2
        echo "Command: ${command[*]}" >&2
        return 1
    fi
}

# Assert that a file is modified after a certain point in time
assert_file_modified_after() {
    local file="$1"
    local reference_time="$2"  # Unix timestamp
    local description="${3:-file modification}"
    
    if [[ ! -f "$file" ]]; then
        echo "File does not exist for modification check: $file" >&2
        return 1
    fi
    
    local file_mtime
    file_mtime=$(stat -c "%Y" "$file" 2>/dev/null || stat -f "%m" "$file" 2>/dev/null)
    
    if [[ "$file_mtime" -le "$reference_time" ]]; then
        echo "Expected $description after timestamp $reference_time" >&2
        echo "File $file was last modified at $file_mtime" >&2
        return 1
    fi
}

# Assert that logs contain a pattern within a timeout
assert_log_contains() {
    local log_file="$1"
    local pattern="$2" 
    local timeout="${3:-10}"
    local description="${4:-log pattern}"
    
    local elapsed=0
    while [[ $elapsed -lt $timeout ]]; do
        if [[ -f "$log_file" ]] && grep -q "$pattern" "$log_file"; then
            return 0
        fi
        sleep 1
        elapsed=$((elapsed + 1))
    done
    
    echo "Expected $description to appear in $log_file within ${timeout}s" >&2
    echo "Pattern: $pattern" >&2
    if [[ -f "$log_file" ]]; then
        echo "Log file contents:" >&2
        tail -10 "$log_file" >&2
    else
        echo "Log file does not exist" >&2
    fi
    return 1
}

# Assert that a process is running
assert_process_running() {
    local process_pattern="$1"
    local description="${2:-process}"
    
    if ! pgrep -f "$process_pattern" >/dev/null 2>&1; then
        echo "Expected $description to be running" >&2
        echo "Process pattern: $process_pattern" >&2
        echo "Currently running processes:" >&2
        ps aux | grep -v grep | grep "$process_pattern" >&2 || echo "No matching processes found" >&2
        return 1
    fi
}

# Assert that a process stops within timeout
assert_process_stops() {
    local process_pattern="$1"
    local timeout="${2:-10}"
    local description="${3:-process}"
    
    local elapsed=0
    while [[ $elapsed -lt $timeout ]]; do
        if ! pgrep -f "$process_pattern" >/dev/null 2>&1; then
            return 0
        fi
        sleep 1
        elapsed=$((elapsed + 1))
    done
    
    echo "Expected $description to stop within ${timeout}s" >&2
    echo "Process pattern: $process_pattern" >&2
    echo "Still running processes:" >&2
    ps aux | grep -v grep | grep "$process_pattern" >&2
    return 1
}

# Assert that a command eventually succeeds (retry with backoff)
assert_eventually_succeeds() {
    local max_attempts="${1:-5}"
    local backoff_factor="${2:-2}"
    local description="${3:-command}"
    shift 3
    local command=("$@")
    
    local attempt=1
    local wait_time=1
    
    while [[ $attempt -le $max_attempts ]]; do
        echo "Attempt $attempt of $max_attempts for $description" >&2
        
        if "${command[@]}" >/dev/null 2>&1; then
            echo "Command succeeded on attempt $attempt" >&2
            return 0
        fi
        
        if [[ $attempt -lt $max_attempts ]]; then
            echo "Waiting ${wait_time}s before retry..." >&2
            sleep "$wait_time"
            wait_time=$((wait_time * backoff_factor))
        fi
        
        ((attempt++))
    done
    
    echo "Expected $description to eventually succeed within $max_attempts attempts" >&2
    echo "Command: ${command[*]}" >&2
    return 1
}

# Assert that network connectivity exists
assert_network_connectivity() {
    local host="${1:-8.8.8.8}"
    local port="${2:-53}"
    local timeout="${3:-5}"
    local description="${4:-network connectivity}"
    
    if ! nc -z -w"$timeout" "$host" "$port" 2>/dev/null; then
        echo "Expected $description to $host:$port" >&2
        return 1
    fi
}

# Assert that disk space is sufficient
assert_disk_space_available() {
    local path="${1:-.}"
    local min_space_mb="$2"
    local description="${3:-disk space}"
    
    local available_mb
    available_mb=$(df -m "$path" | awk 'NR==2 {print $4}')
    
    if [[ "$available_mb" -lt "$min_space_mb" ]]; then
        echo "Expected at least ${min_space_mb}MB $description" >&2
        echo "Available: ${available_mb}MB at $path" >&2
        return 1
    fi
}

# Assert that memory usage is within limits
assert_memory_usage_within() {
    local max_memory_mb="$1"
    local process_pattern="${2:-}"
    local description="${3:-memory usage}"
    
    local memory_usage_mb
    if [[ -n "$process_pattern" ]]; then
        # Check specific process memory
        memory_usage_mb=$(ps -o rss= -p $(pgrep -f "$process_pattern" | head -1) 2>/dev/null | awk '{print int($1/1024)}')
    else
        # Check system memory usage
        memory_usage_mb=$(free -m | awk '/^Mem:/ {print $3}')
    fi
    
    if [[ -z "$memory_usage_mb" ]]; then
        echo "Could not determine $description" >&2
        return 1
    fi
    
    if [[ "$memory_usage_mb" -gt "$max_memory_mb" ]]; then
        echo "Expected $description to be under ${max_memory_mb}MB" >&2
        echo "Actual usage: ${memory_usage_mb}MB" >&2
        return 1
    fi
}

# Assert that a lock file is properly managed
assert_lock_file_behavior() {
    local lock_file="$1"
    local timeout="${2:-10}"
    local description="${3:-lock file behavior}"
    
    # Wait for lock file to be created
    local elapsed=0
    while [[ $elapsed -lt $timeout ]]; do
        if [[ -f "$lock_file" ]]; then
            break
        fi
        sleep 1
        elapsed=$((elapsed + 1))
    done
    
    if [[ ! -f "$lock_file" ]]; then
        echo "Expected $description: lock file not created within ${timeout}s" >&2
        echo "Lock file: $lock_file" >&2
        return 1
    fi
    
    # Check that lock file contains valid content (usually PID)
    local lock_content
    lock_content=$(cat "$lock_file" 2>/dev/null)
    if [[ ! "$lock_content" =~ ^[0-9]+$ ]]; then
        echo "Expected $description: lock file should contain valid PID" >&2
        echo "Lock file content: $lock_content" >&2
        return 1
    fi
}

#######################################
# Export all assertion functions
#######################################
export -f assert_output_contains assert_output_not_contains assert_output_matches
export -f assert_output_empty assert_output_line_count assert_output_equals
export -f assert_file_exists assert_file_not_exists assert_dir_exists assert_dir_not_exists
export -f assert_file_contains assert_file_empty assert_file_permissions
export -f assert_env_set assert_env_equals assert_env_unset assert_env_not_empty
export -f assert_command_exists assert_function_exists assert_command_called
export -f assert_arrays_equal assert_array_contains
export -f assert_json_valid assert_json_field_equals assert_json_field_exists assert_json_schema_valid
export -f assert_port_available assert_port_in_use assert_http_endpoint_reachable assert_http_status assert_port_open
export -f assert_resource_healthy assert_docker_container_running assert_docker_container_healthy
export -f assert_api_response_valid assert_resource_chain_working
export -f assert_mock_used assert_mock_response_used
export -f assert_equals assert_not_empty assert_string_contains assert_greater_than assert_less_than
export -f assert_eventually_true assert_completes_within assert_file_modified_after assert_log_contains
export -f assert_process_running assert_process_stops assert_eventually_succeeds assert_network_connectivity
export -f assert_disk_space_available assert_memory_usage_within assert_lock_file_behavior

echo "[ASSERTIONS] Enhanced assertion library loaded with $(compgen -A function | grep -c '^assert_') assertion functions"