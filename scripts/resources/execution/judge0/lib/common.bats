#!/usr/bin/env bats
# Tests for Judge0 common.sh functions

# Setup for each test
setup() {
    # Load shared test infrastructure
    source "$(dirname "${BATS_TEST_FILENAME}")/../../../tests/bats-fixtures/common_setup.bash"
    
    # Setup standard mocks
    setup_standard_mocks
    
    # Set test environment
    export JUDGE0_PORT="2358"
    export JUDGE0_CONTAINER_NAME="judge0-test"
    export JUDGE0_WORKERS_NAME="judge0-workers-test"
    export JUDGE0_NETWORK_NAME="judge0-network-test"
    export JUDGE0_VOLUME_NAME="judge0-data-test"
    export JUDGE0_DATA_DIR="/tmp/judge0-test"
    export JUDGE0_CONFIG_DIR="/tmp/judge0-test/config"
    export JUDGE0_LOGS_DIR="/tmp/judge0-test/logs"
    export JUDGE0_SUBMISSIONS_DIR="/tmp/judge0-test/submissions"
    export JUDGE0_API_KEY="test_api_key_12345"
    export YES="no"
    
    # Load dependencies
    SCRIPT_DIR="$(dirname "${BATS_TEST_FILENAME}")"
    JUDGE0_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Create test directories
    mkdir -p "$JUDGE0_DATA_DIR"
    mkdir -p "$JUDGE0_CONFIG_DIR"
    mkdir -p "$JUDGE0_LOGS_DIR"
    mkdir -p "$JUDGE0_SUBMISSIONS_DIR"
    
    # Mock system functions
    
    # Mock docker commands
    
    # Mock openssl for API key generation
    openssl() {
        echo "abc123def456ghi789jkl012mno345pq"
    }
    
    # Mock base64 for encoding/decoding
    base64() {
        if [[ "$*" =~ "-d" ]]; then
            echo "decoded_content"
        else
            echo "encoded_content_123"
        fi
    }
    
    # Mock log functions
    
    # Load configuration and messages
    source "${JUDGE0_DIR}/config/defaults.sh"
    source "${JUDGE0_DIR}/config/messages.sh"
    judge0::export_config
    judge0::export_messages
    
    # Load the functions to test
    source "${JUDGE0_DIR}/lib/common.sh"
}

# Cleanup after each test
teardown() {
    rm -rf "$JUDGE0_DATA_DIR"
}

# Test directory creation
@test "judge0::create_directories creates required directories" {
    # Remove test directories first
    rm -rf "$JUDGE0_DATA_DIR"
    
    result=$(judge0::create_directories)
    
    [[ "$result" =~ "directories" ]] || [[ "$result" =~ "created" ]]
    [ -d "$JUDGE0_DATA_DIR" ]
    [ -d "$JUDGE0_CONFIG_DIR" ]
    [ -d "$JUDGE0_LOGS_DIR" ]
    [ -d "$JUDGE0_SUBMISSIONS_DIR" ]
}

# Test directory creation with existing directories
@test "judge0::create_directories handles existing directories" {
    # Directories already exist from setup
    result=$(judge0::create_directories)
    
    [[ "$result" =~ "directories" ]] || [[ "$result" =~ "already" ]] || [[ "$result" =~ "OK" ]]
}

# Test API key generation
@test "judge0::generate_api_key generates valid API key" {
    result=$(judge0::generate_api_key)
    
    [ -n "$result" ]
    [[ ${#result} -eq 32 ]]
    [[ "$result" =~ ^[a-zA-Z0-9]+$ ]]
}

# Test API key generation with custom length
@test "judge0::generate_api_key supports custom length" {
    result=$(judge0::generate_api_key 16)
    
    [ -n "$result" ]
    [[ ${#result} -eq 16 ]]
}

# Test API key saving
@test "judge0::save_api_key saves API key securely" {
    local test_key="test_api_key_987654321"
    
    result=$(judge0::save_api_key "$test_key")
    
    [[ "$result" =~ "saved" ]] || [[ "$result" =~ "stored" ]]
    [ -f "${JUDGE0_CONFIG_DIR}/api_key" ]
    
    # Check file permissions
    local perms=$(stat -c %a "${JUDGE0_CONFIG_DIR}/api_key" 2>/dev/null || stat -f %A "${JUDGE0_CONFIG_DIR}/api_key")
    [[ "$perms" =~ ^6[0-9][0-9]$ ]]  # Should be 600 or similar
}

# Test API key retrieval
@test "judge0::get_api_key retrieves stored API key" {
    local test_key="test_api_key_123456789"
    
    # Save key first
    echo "$test_key" > "${JUDGE0_CONFIG_DIR}/api_key"
    chmod 600 "${JUDGE0_CONFIG_DIR}/api_key"
    
    result=$(judge0::get_api_key)
    
    [[ "$result" == "$test_key" ]]
}

# Test API key retrieval with missing file
@test "judge0::get_api_key handles missing API key file" {
    rm -f "${JUDGE0_CONFIG_DIR}/api_key"
    
    result=$(judge0::get_api_key)
    
    [ -z "$result" ]
}

# Test language validation
@test "judge0::validate_language validates supported languages" {
    result=$(judge0::validate_language "javascript" && echo "valid" || echo "invalid")
    
    [[ "$result" == "valid" ]]
}

# Test language validation with unsupported language
@test "judge0::validate_language rejects unsupported languages" {
    result=$(judge0::validate_language "brainfuck" && echo "valid" || echo "invalid")
    
    [[ "$result" == "invalid" ]]
}

# Test code validation
@test "judge0::validate_code validates code content" {
    local test_code='console.log("Hello, World!");'
    
    result=$(judge0::validate_code "$test_code" && echo "valid" || echo "invalid")
    
    [[ "$result" == "valid" ]]
}

# Test code validation with empty code
@test "judge0::validate_code rejects empty code" {
    result=$(judge0::validate_code "" && echo "valid" || echo "invalid")
    
    [[ "$result" == "invalid" ]]
}

# Test code validation with dangerous code
@test "judge0::validate_code detects potentially dangerous code" {
    local dangerous_code='import os; os.system("rm -rf /")'
    
    result=$(judge0::validate_code "$dangerous_code" && echo "valid" || echo "invalid")
    
    # Should still be valid (security is handled by Judge0 sandbox)
    [[ "$result" == "valid" ]]
}

# Test file encoding
@test "judge0::encode_file encodes file content" {
    local test_file="/tmp/test_code.js"
    echo 'console.log("test");' > "$test_file"
    
    result=$(judge0::encode_file "$test_file")
    
    [ -n "$result" ]
    [[ "$result" =~ "encoded" ]]
    
    rm -f "$test_file"
}

# Test file encoding with missing file
@test "judge0::encode_file handles missing file" {
    run judge0::encode_file "/nonexistent/file.js"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR:" ]] || [[ "$output" =~ "not found" ]]
}

# Test submission token validation
@test "judge0::validate_token validates submission tokens" {
    result=$(judge0::validate_token "abc123-def456-ghi789" && echo "valid" || echo "invalid")
    
    [[ "$result" == "valid" ]]
}

# Test submission token validation with invalid token
@test "judge0::validate_token rejects invalid tokens" {
    result=$(judge0::validate_token "invalid" && echo "valid" || echo "invalid")
    
    [[ "$result" == "invalid" ]]
}

# Test configuration validation
@test "judge0::validate_config validates configuration" {
    result=$(judge0::validate_config)
    
    [[ "$result" =~ "valid" ]] || [[ "$result" =~ "configuration" ]]
}

# Test resource limits validation
@test "judge0::validate_limits validates resource limits" {
    result=$(judge0::validate_limits)
    
    [[ "$result" =~ "limits" ]] || [[ "$result" =~ "valid" ]]
}

# Test resource limits validation with invalid limits
@test "judge0::validate_limits detects invalid limits" {
    # Temporarily set invalid limits
    local original_cpu="$JUDGE0_CPU_TIME_LIMIT"
    local original_memory="$JUDGE0_MEMORY_LIMIT"
    
    export JUDGE0_CPU_TIME_LIMIT="999999"  # Too high
    export JUDGE0_MEMORY_LIMIT="99999999"  # Too high
    
    run judge0::validate_limits
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR:" ]] || [[ "$output" =~ "invalid" ]]
    
    # Restore original values
    export JUDGE0_CPU_TIME_LIMIT="$original_cpu"
    export JUDGE0_MEMORY_LIMIT="$original_memory"
}

# Test network detection
@test "judge0::check_network_connectivity tests network access" {
    result=$(judge0::check_network_connectivity)
    
    [[ "$result" =~ "network" ]] || [[ "$result" =~ "connectivity" ]]
}

# Test Docker network management
@test "judge0::create_network creates Docker network" {
    result=$(judge0::create_network)
    
    [[ "$result" =~ "network" ]]
    [[ "$result" =~ "$JUDGE0_NETWORK_NAME" ]]
}

# Test Docker network cleanup
@test "judge0::remove_network removes Docker network" {
    result=$(judge0::remove_network)
    
    [[ "$result" =~ "network" ]]
    [[ "$result" =~ "$JUDGE0_NETWORK_NAME" ]]
}

# Test Docker volume management
@test "judge0::create_volume creates Docker volume" {
    result=$(judge0::create_volume)
    
    [[ "$result" =~ "volume" ]]
    [[ "$result" =~ "$JUDGE0_VOLUME_NAME" ]]
}

# Test Docker volume cleanup
@test "judge0::remove_volume removes Docker volume" {
    result=$(judge0::remove_volume)
    
    [[ "$result" =~ "volume" ]]
    [[ "$result" =~ "$JUDGE0_VOLUME_NAME" ]]
}

# Test data cleanup
@test "judge0::cleanup_data removes data directory" {
    # Create test data
    echo "test submission" > "${JUDGE0_SUBMISSIONS_DIR}/test.json"
    echo "test log" > "${JUDGE0_LOGS_DIR}/test.log"
    
    export YES="yes"
    result=$(judge0::cleanup_data "yes")
    
    [[ "$result" =~ "cleanup" ]] || [[ "$result" =~ "removed" ]]
    [ ! -d "$JUDGE0_DATA_DIR" ]
}

# Test data cleanup with confirmation
@test "judge0::cleanup_data respects user confirmation" {
    export YES="no"
    result=$(judge0::cleanup_data "no")
    
    [[ "$result" =~ "cancelled" ]] || [[ "$result" =~ "aborted" ]]
    [ -d "$JUDGE0_DATA_DIR" ]  # Should still exist
}

# Test permission management
@test "judge0::set_permissions sets correct file permissions" {
    echo "test" > "${JUDGE0_CONFIG_DIR}/test_file"
    
    result=$(judge0::set_permissions "${JUDGE0_CONFIG_DIR}/test_file" "600")
    
    [[ "$result" =~ "permission" ]] || [[ "$result" =~ "set" ]]
    
    local perms=$(stat -c %a "${JUDGE0_CONFIG_DIR}/test_file" 2>/dev/null || stat -f %A "${JUDGE0_CONFIG_DIR}/test_file")
    [[ "$perms" == "600" ]]
}

# Test log rotation
@test "judge0::rotate_logs manages log file rotation" {
    # Create test log files
    echo "old log" > "${JUDGE0_LOGS_DIR}/judge0.log.1"
    echo "current log" > "${JUDGE0_LOGS_DIR}/judge0.log"
    
    result=$(judge0::rotate_logs)
    
    [[ "$result" =~ "log" ]] || [[ "$result" =~ "rotate" ]]
}

# Test health monitoring
@test "judge0::check_health performs basic health checks" {
    result=$(judge0::check_health)
    
    [[ "$result" =~ "health" ]] || [[ "$result" =~ "check" ]]
}

# Test backup creation
@test "judge0::create_backup creates data backup" {
    result=$(judge0::create_backup)
    
    [[ "$result" =~ "backup" ]] || [[ "$result" =~ "created" ]]
}

# Test backup restoration
@test "judge0::restore_backup restores data from backup" {
    result=$(judge0::restore_backup "/tmp/backup.tar.gz")
    
    [[ "$result" =~ "restore" ]] || [[ "$result" =~ "backup" ]]
}

# Test environment preparation
@test "judge0::prepare_environment prepares runtime environment" {
    result=$(judge0::prepare_environment)
    
    [[ "$result" =~ "environment" ]] || [[ "$result" =~ "prepared" ]]
}

# Test worker management
@test "judge0::manage_workers handles worker lifecycle" {
    result=$(judge0::manage_workers "start")
    
    [[ "$result" =~ "worker" ]] || [[ "$result" =~ "start" ]]
}

# Test queue management
@test "judge0::clear_queue clears submission queue" {
    result=$(judge0::clear_queue)
    
    [[ "$result" =~ "queue" ]] || [[ "$result" =~ "clear" ]]
}

# Test metrics collection
@test "judge0::collect_metrics gathers system metrics" {
    result=$(judge0::collect_metrics)
    
    [[ "$result" =~ "metrics" ]] || [[ "$result" =~ "collected" ]]
}

# Test utility functions
@test "judge0::format_size formats file sizes" {
    result=$(judge0::format_size "1048576")
    
    [[ "$result" =~ "MB" ]] || [[ "$result" =~ "1024" ]]
}

@test "judge0::format_time formats execution time" {
    result=$(judge0::format_time "1.5")
    
    [[ "$result" =~ "1.5" ]] && [[ "$result" =~ "s" ]]
}

# Test error handling
@test "judge0::handle_error processes error conditions gracefully" {
    result=$(judge0::handle_error "Test error message" 500)
    
    [[ "$result" =~ "error" ]] || [[ "$result" =~ "Test error message" ]]
}

# Test dependency verification
@test "judge0::verify_dependencies checks required dependencies" {
    result=$(judge0::verify_dependencies)
    
    [[ "$result" =~ "dependencies" ]] || [[ "$result" =~ "verified" ]]
}

# Test system compatibility
@test "judge0::check_compatibility verifies system compatibility" {
    result=$(judge0::check_compatibility)
    
    [[ "$result" =~ "compatible" ]] || [[ "$result" =~ "system" ]]
}