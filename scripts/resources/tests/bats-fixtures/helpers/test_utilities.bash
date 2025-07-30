#!/usr/bin/env bash
# Common Test Utilities for Bats Tests
# Provides helper functions for test setup, execution, and validation

# Create a temporary test file with content
create_test_file() {
    local filename="${1:-test_file.txt}"
    local content="${2:-Test content}"
    local filepath="${BATS_TEST_TMPDIR}/${filename}"
    
    echo "$content" > "$filepath"
    echo "$filepath"
}

# Create a temporary test directory
create_test_dir() {
    local dirname="${1:-test_dir}"
    local dirpath="${BATS_TEST_TMPDIR}/${dirname}"
    
    mkdir -p "$dirpath"
    echo "$dirpath"
}

# Run command with timeout
run_with_timeout() {
    local timeout="$1"
    shift
    
    if command -v timeout >/dev/null 2>&1; then
        timeout "$timeout" "$@"
    else
        # Fallback if timeout command not available
        "$@" &
        local pid=$!
        local count=0
        while kill -0 $pid 2>/dev/null && [[ $count -lt $timeout ]]; do
            sleep 1
            ((count++))
        done
        if kill -0 $pid 2>/dev/null; then
            kill -9 $pid
            return 124  # timeout exit code
        fi
        wait $pid
    fi
}

# Retry command with backoff
retry_with_backoff() {
    local max_attempts="${1:-3}"
    local initial_delay="${2:-1}"
    shift 2
    
    local attempt=1
    local delay=$initial_delay
    
    while [[ $attempt -le $max_attempts ]]; do
        if "$@"; then
            return 0
        fi
        
        if [[ $attempt -lt $max_attempts ]]; then
            echo "Attempt $attempt failed, retrying in ${delay}s..." >&2
            sleep "$delay"
            delay=$((delay * 2))
        fi
        
        ((attempt++))
    done
    
    return 1
}

# Wait for condition with timeout
wait_for_condition() {
    local timeout="${1:-30}"
    local interval="${2:-1}"
    local condition="${3}"
    
    local elapsed=0
    while [[ $elapsed -lt $timeout ]]; do
        if eval "$condition"; then
            return 0
        fi
        sleep "$interval"
        ((elapsed += interval))
    done
    
    echo "Timeout waiting for condition: $condition" >&2
    return 1
}

# Generate random string
generate_random_string() {
    local length="${1:-10}"
    local chars="${2:-abcdefghijklmnopqrstuvwxyz0123456789}"
    
    local result=""
    for ((i=0; i<length; i++)); do
        result+="${chars:$((RANDOM % ${#chars})):1}"
    done
    echo "$result"
}

# Create mock configuration file
create_mock_config() {
    local resource="$1"
    local config_content="${2:-}"
    
    local config_file="${BATS_TEST_TMPDIR}/${resource}_config.json"
    
    if [[ -z "$config_content" ]]; then
        # Default minimal config
        config_content=$(cat <<EOF
{
    "enabled": true,
    "baseUrl": "http://localhost:8080",
    "healthCheck": {
        "intervalMs": 60000,
        "timeoutMs": 5000
    }
}
EOF
)
    fi
    
    echo "$config_content" > "$config_file"
    echo "$config_file"
}

# Capture function output and status
capture_function_output() {
    local func="$1"
    shift
    
    local output_file="${BATS_TEST_TMPDIR}/function_output"
    local status_file="${BATS_TEST_TMPDIR}/function_status"
    
    # Run function and capture output/status
    {
        "$func" "$@"
        echo $? > "$status_file"
    } > "$output_file" 2>&1
    
    # Set variables for use in test
    output=$(cat "$output_file")
    status=$(cat "$status_file")
}

# Mock HTTP server response
setup_mock_http_server() {
    local port="$1"
    local response="${2:-OK}"
    local response_file="${BATS_TEST_TMPDIR}/http_response"
    
    echo "$response" > "$response_file"
    
    # Simple HTTP server using nc (netcat)
    while true; do
        {
            echo -e "HTTP/1.1 200 OK\r\nContent-Length: $(wc -c < "$response_file")\r\n\r\n"
            cat "$response_file"
        } | nc -l -p "$port" -q 1
    done &
    
    echo $!  # Return PID
}

# Stop mock HTTP server
stop_mock_http_server() {
    local pid="$1"
    kill "$pid" 2>/dev/null || true
}

# Parse command line arguments (for testing CLI parsing)
parse_test_args() {
    local -n args_array=$1
    shift
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --*=*)
                args_array["${1%%=*}"]="${1#*=}"
                shift
                ;;
            --*)
                if [[ $# -gt 1 ]] && [[ ! "$2" =~ ^-- ]]; then
                    args_array["$1"]="$2"
                    shift 2
                else
                    args_array["$1"]="true"
                    shift
                fi
                ;;
            *)
                shift
                ;;
        esac
    done
}

# Compare files ignoring whitespace differences
files_equal_ignore_whitespace() {
    local file1="$1"
    local file2="$2"
    
    diff -w -B "$file1" "$file2" >/dev/null 2>&1
}

# Extract function calls from output
extract_function_calls() {
    local output="$1"
    local pattern="${2:-.*}"
    
    echo "$output" | grep -oE "${pattern}::[a-zA-Z0-9_:]+" | sort -u
}

# Validate JSON structure
is_valid_json() {
    local json="$1"
    echo "$json" | jq . >/dev/null 2>&1
}

# Get free port
get_free_port() {
    local start_port="${1:-8000}"
    local end_port="${2:-9000}"
    
    for port in $(seq $start_port $end_port); do
        if ! lsof -i ":$port" >/dev/null 2>&1; then
            echo "$port"
            return 0
        fi
    done
    
    return 1
}

# Create test user/credentials
create_test_credentials() {
    local resource="$1"
    
    cat <<EOF
{
    "username": "test_user_${resource}",
    "password": "test_pass_$(generate_random_string 16)",
    "apiKey": "test_key_$(generate_random_string 32)"
}
EOF
}

# Cleanup test artifacts
cleanup_test_artifacts() {
    local pattern="${1:-test_*}"
    
    if [[ -n "${BATS_TEST_TMPDIR:-}" ]] && [[ -d "$BATS_TEST_TMPDIR" ]]; then
        find "$BATS_TEST_TMPDIR" -name "$pattern" -delete 2>/dev/null || true
    fi
}

# Debug helper - print test environment
debug_test_env() {
    echo "=== Test Environment Debug ===" >&2
    echo "BATS_TEST_FILENAME: ${BATS_TEST_FILENAME:-not set}" >&2
    echo "BATS_TEST_TMPDIR: ${BATS_TEST_TMPDIR:-not set}" >&2
    echo "RESOURCE_NAME: ${RESOURCE_NAME:-not set}" >&2
    echo "RESOURCE_DIR: ${RESOURCE_DIR:-not set}" >&2
    echo "RESOURCES_DIR: ${RESOURCES_DIR:-not set}" >&2
    echo "MOCK_RESPONSES_DIR: ${MOCK_RESPONSES_DIR:-not set}" >&2
    echo "============================" >&2
}