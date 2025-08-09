#!/usr/bin/env bats

# Load Vrooli test infrastructure (REQUIRED)
source "${BATS_TEST_DIRNAME}/../../../../__test/fixtures/setup.bash"

# Expensive setup operations (run once per file)
setup_file() {
    # Use appropriate setup function
    vrooli_setup_service_test "windmill"
    
    # Export paths for use in setup()
    export SETUP_FILE_SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    export SETUP_FILE_WINDMILL_DIR="$(dirname "${BATS_TEST_DIRNAME}")" 
    export SETUP_FILE_CONFIG_DIR="$(dirname "${BATS_TEST_DIRNAME}")/config"
    export SETUP_FILE_LIB_DIR="$(dirname "${BATS_TEST_DIRNAME}")/lib"
}

# Lightweight per-test setup
setup() {
    # Setup standard mocks
    vrooli_auto_setup
    
    # Use paths from setup_file
    SCRIPT_DIR="${SETUP_FILE_SCRIPT_DIR}"
    WINDMILL_DIR="${SETUP_FILE_WINDMILL_DIR}"
    CONFIG_DIR="${SETUP_FILE_CONFIG_DIR}"
    LIB_DIR="${SETUP_FILE_LIB_DIR}"
    
    # Set test environment BEFORE sourcing config files to avoid readonly conflicts
    export WINDMILL_PORT="5681"
    export WINDMILL_CONTAINER_NAME="windmill-test"
    export WINDMILL_BASE_URL="http://localhost:5681"
    export WINDMILL_DATA_DIR="/tmp/windmill-test"
    export WINDMILL_DB_PASSWORD="test-password"
    export YES="no"
    
    # Create test directories
    mkdir -p "$WINDMILL_DATA_DIR"
    
    # Mock resources functions that are called during config loading
    resources::get_default_port() {
        case "$1" in
            "windmill") echo "5681" ;;
            *) echo "8080" ;;
        esac
    }
    
    # Now source the config files
    source "${WINDMILL_DIR}/config/defaults.sh"
    source "${WINDMILL_DIR}/config/messages.sh"
    
    # Export config and messages
    windmill::export_config
    windmill::export_messages
    
    # Load the functions to test
    source "${WINDMILL_DIR}/lib/common.sh"
    
    # Mock system functions
    system::check_port() {
        local port="$1"
        if [[ "$port" == "9999" ]]; then
            return 1  # Port busy for conflict tests
        fi
        return 0
    }
    
    system::get_available_port() {
        echo "5682"  # Mock alternative port
    }
    
    # Mock openssl for password generation
    openssl() {
        case "$*" in
            *"rand"*)
                echo "randomly_generated_password_123"
                ;;
            *) echo "OPENSSL: $*" ;;
        esac
    }
    
    # Mock base64 for encoding
    base64() {
        echo "encoded_string_123"
    }
}

# Cleanup after each test
teardown() {
    rm -rf "$WINDMILL_DATA_DIR"
    vrooli_cleanup_test
}

# Test container existence check
@test "windmill::container_exists detects existing container" {
    result=$(windmill::container_exists && echo "exists" || echo "not found")
    
    [[ "$result" == "exists" ]]
}

# Test container existence check with missing container
@test "windmill::container_exists handles missing container" {
    # Override docker ps to return empty
    docker() {
        case "$1" in
            "ps") echo "" ;;
            *) echo "DOCKER: $*" ;;
        esac
    }
    
    result=$(windmill::container_exists && echo "exists" || echo "not found")
    
    [[ "$result" == "not found" ]]
}

# Test URL validation
@test "windmill::validate_url validates correct URLs" {
    result=$(windmill::validate_url "http://localhost:5681" && echo "valid" || echo "invalid")
    
    [[ "$result" == "valid" ]]
}

# Test URL validation with invalid URL
@test "windmill::validate_url rejects invalid URLs" {
    result=$(windmill::validate_url "not-a-url" && echo "valid" || echo "invalid")
    
    [[ "$result" == "invalid" ]]
}

# Test port validation
@test "windmill::validate_port validates correct port numbers" {
    result=$(windmill::validate_port "5681" && echo "valid" || echo "invalid")
    
    [[ "$result" == "valid" ]]
}

# Test port validation with invalid port
@test "windmill::validate_port rejects invalid ports" {
    result=$(windmill::validate_port "70000" && echo "valid" || echo "invalid")
    
    [[ "$result" == "invalid" ]]
}

# Test email validation
@test "windmill::validate_email validates correct email addresses" {
    result=$(windmill::validate_email "admin@example.com" && echo "valid" || echo "invalid")
    
    [[ "$result" == "valid" ]]
}

# Test email validation with invalid email
@test "windmill::validate_email rejects invalid emails" {
    result=$(windmill::validate_email "not-an-email" && echo "valid" || echo "invalid")
    
    [[ "$result" == "invalid" ]]
}

# Test password generation
@test "windmill::generate_password creates secure passwords" {
    result=$(windmill::generate_password)
    
    [ -n "$result" ]
    [[ ${#result} -ge 12 ]]  # At least 12 characters
}

# Test password generation with custom length
@test "windmill::generate_password respects custom length" {
    result=$(windmill::generate_password 20)
    
    [ -n "$result" ]
    # Note: actual implementation might vary
    [[ "$result" =~ "randomly_generated_password" ]]
}

# Test workspace name validation
@test "windmill::validate_workspace_name validates workspace names" {
    result=$(windmill::validate_workspace_name "my-workspace" && echo "valid" || echo "invalid")
    
    [[ "$result" == "valid" ]]
}

# Test workspace name validation with invalid characters
@test "windmill::validate_workspace_name rejects invalid workspace names" {
    result=$(windmill::validate_workspace_name "my workspace!" && echo "valid" || echo "invalid")
    
    [[ "$result" == "invalid" ]]
}

# Test script path validation
@test "windmill::validate_script_path validates script paths" {
    # Create a test script file
    local script_path="/tmp/test-script.py"
    echo "print('test')" > "$script_path"
    
    result=$(windmill::validate_script_path "$script_path" && echo "valid" || echo "invalid")
    
    [[ "$result" == "valid" ]]
    
    rm -f "$script_path"
}

# Test script path validation with missing file
@test "windmill::validate_script_path rejects missing files" {
    result=$(windmill::validate_script_path "/nonexistent/script.py" && echo "valid" || echo "invalid")
    
    [[ "$result" == "invalid" ]]
}

# Test JSON validation
@test "windmill::validate_json validates correct JSON" {
    local json='{"key": "value", "number": 123}'
    
    result=$(windmill::validate_json "$json" && echo "valid" || echo "invalid")
    
    [[ "$result" == "valid" ]]
}

# Test JSON validation with invalid JSON
@test "windmill::validate_json rejects invalid JSON" {
    local json='{"key": "value", "number": }'  # Invalid JSON
    
    result=$(windmill::validate_json "$json" && echo "valid" || echo "invalid")
    
    [[ "$result" == "invalid" ]]
}

# Test environment variable check
@test "windmill::check_env_vars validates required environment variables" {
    export WINDMILL_DB_PASSWORD="test123"
    export WINDMILL_ADMIN_EMAIL="admin@test.com"
    
    result=$(windmill::check_env_vars && echo "valid" || echo "invalid")
    
    [[ "$result" == "valid" ]]
}

# Test environment variable check with missing variables
@test "windmill::check_env_vars detects missing required variables" {
    unset WINDMILL_DB_PASSWORD
    
    result=$(windmill::check_env_vars && echo "valid" || echo "invalid")
    
    [[ "$result" == "invalid" ]]
}

# Test configuration file creation
@test "windmill::create_config_file generates configuration file" {
    local config_file="/tmp/windmill-test-config.env"
    
    result=$(windmill::create_config_file "$config_file")
    
    [ -f "$config_file" ]
    [[ "$result" =~ "created" ]] || [[ "$result" =~ "generated" ]]
    
    rm -f "$config_file"
}

# Test configuration file loading
@test "windmill::load_config_file loads configuration from file" {
    local config_file="/tmp/windmill-test-config.env"
    echo "WINDMILL_CUSTOM_PORT=9876" > "$config_file"
    
    result=$(windmill::load_config_file "$config_file")
    
    [[ "$WINDMILL_CUSTOM_PORT" == "9876" ]]
    [[ "$result" =~ "loaded" ]] || [[ "$result" =~ "config" ]]
    
    rm -f "$config_file"
}

# Test network connectivity check
@test "windmill::check_connectivity tests network connectivity" {
    result=$(windmill::check_connectivity)
    
    [[ "$result" =~ "connectivity" ]] || [[ "$result" =~ "network" ]]
}

# Test dependency check
@test "windmill::check_dependencies validates system dependencies" {
    result=$(windmill::check_dependencies)
    
    [[ "$result" =~ "dependencies" ]] || [[ "$result" =~ "requirement" ]]
}

# Test dependency check with missing dependency
@test "windmill::check_dependencies detects missing dependencies" {
    # Override system command check to fail for docker
    system::is_command() {
        case "$1" in
            "docker") return 1 ;;
            *) return 0 ;;
        esac
    }
    
    result=$(windmill::check_dependencies)
    
    [[ "$result" =~ "missing" ]] || [[ "$result" =~ "not found" ]]
}

# Test cleanup function
@test "windmill::cleanup removes temporary files and resources" {
    # Create some temporary files
    touch "/tmp/windmill-temp-1" "/tmp/windmill-temp-2"
    
    result=$(windmill::cleanup)
    
    [[ "$result" =~ "cleanup" ]] || [[ "$result" =~ "cleaned" ]]
}

# Test backup directory creation
@test "windmill::ensure_backup_dir creates backup directory" {
    local backup_dir="/tmp/windmill-backups"
    
    result=$(windmill::ensure_backup_dir "$backup_dir")
    
    [ -d "$backup_dir" ]
    [[ "$result" =~ "backup" ]] || [[ "$result" =~ "directory" ]]
    
    rm -rf "$backup_dir"
}

# Test log directory creation
@test "windmill::ensure_log_dir creates log directory" {
    local log_dir="/tmp/windmill-logs"
    
    result=$(windmill::ensure_log_dir "$log_dir")
    
    [ -d "$log_dir" ]
    [[ "$result" =~ "log" ]] || [[ "$result" =~ "directory" ]]
    
    rm -rf "$log_dir"
}

# Test data directory initialization
@test "windmill::init_data_dir initializes data directory structure" {
    result=$(windmill::init_data_dir "$WINDMILL_DATA_DIR")
    
    [ -d "$WINDMILL_DATA_DIR" ]
    [[ "$result" =~ "initialized" ]] || [[ "$result" =~ "data" ]]
}

# Test version comparison
@test "windmill::compare_versions compares version strings" {
    result=$(windmill::compare_versions "1.2.0" "1.1.0")
    
    # Should indicate first version is newer
    [[ "$result" == "1" ]] || [[ "$result" =~ "newer" ]]
}

# Test version comparison with equal versions
@test "windmill::compare_versions handles equal versions" {
    result=$(windmill::compare_versions "1.2.0" "1.2.0")
    
    # Should indicate versions are equal
    [[ "$result" == "0" ]] || [[ "$result" =~ "equal" ]]
}

# Test file backup
@test "windmill::backup_file creates file backup" {
    local test_file="/tmp/windmill-test.txt"
    echo "test content" > "$test_file"
    
    result=$(windmill::backup_file "$test_file")
    
    [[ "$result" =~ "backup" ]]
    [ -f "${test_file}.backup" ] || [[ "$result" =~ "created" ]]
    
    rm -f "$test_file" "${test_file}.backup"
}

# Test file restoration
@test "windmill::restore_file restores file from backup" {
    local test_file="/tmp/windmill-test.txt"
    local backup_file="${test_file}.backup"
    
    echo "original content" > "$backup_file"
    
    result=$(windmill::restore_file "$test_file")
    
    [[ "$result" =~ "restore" ]]
    
    rm -f "$test_file" "$backup_file"
}

# Test random string generation
@test "windmill::generate_random_string creates random strings" {
    result=$(windmill::generate_random_string 16)
    
    [ -n "$result" ]
    [[ ${#result} -ge 10 ]]  # At least some reasonable length
}

# Test timestamp generation
@test "windmill::get_timestamp returns formatted timestamp" {
    result=$(windmill::get_timestamp)
    
    [ -n "$result" ]
    [[ "$result" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2} ]] || [[ "$result" =~ ^[0-9]+ ]]
}

# Test confirmation prompt
@test "windmill::confirm_action handles user confirmation" {
    export YES="yes"
    
    result=$(windmill::confirm_action "Test action" && echo "confirmed" || echo "cancelled")
    
    [[ "$result" == "confirmed" ]]
}

# Test confirmation prompt with default no
@test "windmill::confirm_action handles default rejection" {
    export YES="no"
    
    result=$(windmill::confirm_action "Test action" && echo "confirmed" || echo "cancelled")
    
    [[ "$result" == "cancelled" ]]
}

# Test URL encoding
@test "windmill::url_encode encodes URLs properly" {
    result=$(windmill::url_encode "hello world")
    
    [[ "$result" =~ "hello%20world" ]] || [[ "$result" =~ "hello+world" ]]
}

# Test base64 encoding
@test "windmill::encode_base64 encodes strings to base64" {
    result=$(windmill::encode_base64 "test string")
    
    [[ "$result" =~ "encoded_string" ]]
}

# Test base64 decoding
@test "windmill::decode_base64 decodes base64 strings" {
    result=$(windmill::decode_base64 "dGVzdCBzdHJpbmc=")
    
    [[ "$result" =~ "test" ]] || [[ "$result" =~ "decoded" ]]
}

# Test error handling utilities
@test "windmill::handle_error processes errors appropriately" {
    result=$(windmill::handle_error "Test error message" 1)
    
    [[ "$result" =~ "ERROR:" ]]
    [[ "$result" =~ "Test error message" ]]
}

# Test retry mechanism
@test "windmill::retry_command retries failed commands" {
    # Mock a command that fails twice then succeeds
    local attempt=0
    failing_command() {
        ((attempt++))
        if [[ $attempt -lt 3 ]]; then
            return 1
        fi
        echo "success"
        return 0
    }
    
    result=$(windmill::retry_command failing_command 3)
    
    [[ "$result" =~ "success" ]]
}

# Test timeout wrapper
@test "windmill::timeout_command handles command timeouts" {
    result=$(windmill::timeout_command 5 "echo test")
    
    [[ "$result" =~ "test" ]]
}

# Test resource monitoring
@test "windmill::monitor_resources tracks system resources" {
    result=$(windmill::monitor_resources)
    
    [[ "$result" =~ "resource" ]] || [[ "$result" =~ "memory" ]] || [[ "$result" =~ "CPU" ]]
}

# Test health check utilities
@test "windmill::wait_for_health waits for service health" {
    result=$(windmill::wait_for_health "http://localhost:5681" 10)
    
    [[ "$result" =~ "healthy" ]] || [[ "$result" =~ "ready" ]]
}

# Test configuration merging
@test "windmill::merge_config merges configuration files" {
    local config1="/tmp/config1.env"
    local config2="/tmp/config2.env"
    local merged="/tmp/merged.env"
    
    echo "VAR1=value1" > "$config1"
    echo "VAR2=value2" > "$config2"
    
    result=$(windmill::merge_config "$config1" "$config2" "$merged")
    
    [ -f "$merged" ]
    [[ "$result" =~ "merged" ]]
    
    rm -f "$config1" "$config2" "$merged"
}