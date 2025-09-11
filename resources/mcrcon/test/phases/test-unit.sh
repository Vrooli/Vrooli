#!/usr/bin/env bash
# mcrcon unit tests - library function validation (<60s)

set -euo pipefail

# Get directories
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_DIR
readonly TEST_DIR="$(dirname "$SCRIPT_DIR")"
readonly RESOURCE_DIR="$(dirname "$TEST_DIR")"

# Source libraries
source "${RESOURCE_DIR}/config/defaults.sh"
source "${RESOURCE_DIR}/lib/core.sh"

# Test counter
TESTS_RUN=0
TESTS_PASSED=0

# Test function
run_test() {
    local test_name="$1"
    local test_function="$2"
    
    TESTS_RUN=$((TESTS_RUN + 1))
    echo -n "  Testing $test_name... "
    
    if $test_function > /dev/null 2>&1; then
        echo "✓"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        echo "✗"
        return 1
    fi
}

# Test: Data directory creation
test_ensure_data_dir() {
    local test_dir="/tmp/mcrcon_test_$$"
    export MCRCON_DATA_DIR="$test_dir"
    
    ensure_data_dir
    
    if [[ -d "$test_dir/bin" ]] && [[ -d "$test_dir/logs" ]]; then
        rm -rf "$test_dir"
        return 0
    else
        rm -rf "$test_dir"
        return 1
    fi
}

# Test: Installation check
test_is_installed() {
    local test_dir="/tmp/mcrcon_test_$$"
    export MCRCON_BINARY="$test_dir/bin/mcrcon"
    
    # Should return false when not installed
    if is_installed; then
        return 1
    fi
    
    # Create fake binary
    mkdir -p "$test_dir/bin"
    touch "$MCRCON_BINARY"
    chmod +x "$MCRCON_BINARY"
    
    # Should return true when installed
    if is_installed; then
        rm -rf "$test_dir"
        return 0
    else
        rm -rf "$test_dir"
        return 1
    fi
}

# Test: Server configuration JSON handling
test_server_config() {
    local test_file="/tmp/mcrcon_servers_$$.json"
    export MCRCON_CONFIG_FILE="$test_file"
    
    # Initialize empty config
    echo '{"servers": []}' > "$test_file"
    
    # Test adding server (basic JSON structure test)
    local temp_file=$(mktemp)
    jq '.servers += [{"name": "test", "host": "localhost", "port": 25575}]' "$test_file" > "$temp_file"
    
    if jq -e '.servers | length == 1' "$temp_file" > /dev/null; then
        rm -f "$test_file" "$temp_file"
        return 0
    else
        rm -f "$test_file" "$temp_file"
        return 1
    fi
}

# Test: Log functions (basic functionality)
test_logging() {
    local test_log="/tmp/mcrcon_log_$$.log"
    export MCRCON_LOG_FILE="$test_log"
    export MCRCON_DEBUG="true"
    
    log_info "Test info message"
    log_debug "Test debug message"
    
    if grep -q "Test info message" "$test_log" && grep -q "Test debug message" "$test_log"; then
        rm -f "$test_log"
        return 0
    else
        rm -f "$test_log"
        return 1
    fi
}

# Main unit test execution
main() {
    echo "Running mcrcon unit tests..."
    echo "=============================="
    
    # Run unit tests
    run_test "ensure_data_dir function" test_ensure_data_dir
    run_test "is_installed function" test_is_installed
    run_test "server config JSON" test_server_config
    run_test "logging functions" test_logging
    
    # Summary
    echo "=============================="
    echo "Unit Tests: ${TESTS_PASSED}/${TESTS_RUN} passed"
    
    if [[ $TESTS_PASSED -eq $TESTS_RUN ]]; then
        echo "All unit tests passed!"
        exit 0
    else
        echo "Some unit tests failed"
        exit 1
    fi
}

main "$@"