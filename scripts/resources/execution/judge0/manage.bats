#!/usr/bin/env bats
# Judge0 Resource Management Tests
# Tests for Judge0 installation, configuration, and management

# Load Vrooli test infrastructure
source "$(dirname "${BATS_TEST_FILENAME}")/../../../__test/fixtures/setup.bash"

# Expensive setup operations run once per file
setup_file() {
    # Use Vrooli service test setup
    vrooli_setup_service_test "judge0"
    
    # Load resource specific configuration once per file
    SCRIPT_DIR="$(dirname "${BATS_TEST_FILENAME}")"
    JUDGE0_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Load configuration and manage script once
    source "${JUDGE0_DIR}/config/defaults.sh"
    source "${JUDGE0_DIR}/config/messages.sh"
    source "${SCRIPT_DIR}/manage.sh"
}

# Lightweight per-test setup
setup() {
    # Setup standard Vrooli mocks
    vrooli_auto_setup
    
    # Set test environment variables (lightweight per-test)
    export JUDGE0_CUSTOM_PORT="9999"
    export JUDGE0_CONTAINER_NAME="judge0-test"
    export JUDGE0_WORKERS_NAME="judge0-workers-test"
    export JUDGE0_NETWORK_NAME="judge0-network-test"
    export JUDGE0_VOLUME_NAME="judge0-data-test"
    export JUDGE0_BASE_URL="http://localhost:9999"
    export JUDGE0_DATA_DIR="${VROOLI_TEST_TMPDIR}/judge0"
    export JUDGE0_CONFIG_DIR="${JUDGE0_DATA_DIR}/config"
    export FORCE="no"
    export YES="no"
    export TEST_MODE="true"
    export JUDGE0_TEST_MODE="true"
    
    # Create test directories
    mkdir -p "$JUDGE0_CONFIG_DIR"
    
    # Export config functions
    judge0::export_config
    judge0::export_messages
    
    # Mock log functions
    log::header() { echo "=== $* ==="; }
    log::info() { echo "[INFO] $*"; }
    log::error() { echo "[ERROR] $*" >&2; }
    log::success() { echo "[SUCCESS] $*"; }
    log::warning() { echo "[WARNING] $*" >&2; }
    export -f log::header log::info log::error log::success log::warning
}

# BATS teardown function - runs after each test
teardown() {
    vrooli_cleanup_test
}

# Helper function to run manage actions
run_manage() {
    local action="$1"
    shift
    judge0::handle_action "$action" "$@"
}

# =============================================================================
# Basic Functionality Tests
# =============================================================================

@test "Judge0: Script exists and is executable" {
    [ -f "${BATS_TEST_DIRNAME}/manage.sh" ]
    [ -x "${BATS_TEST_DIRNAME}/manage.sh" ]
}

@test "Judge0: Help command displays usage" {
    run judge0::show_help
    assert_success
    [[ "$output" =~ "Judge0 Code Execution" ]]
    [[ "$output" =~ "USAGE:" ]]
    [[ "$output" =~ "OPTIONS:" ]]
}

@test "Judge0: Version information available" {
    # Mock docker for version info
    docker() {
        case "$*" in
            *"exec"*"version"*) echo "Judge0" ;;
            *) return 0 ;;
        esac
    }
    export -f docker
    
    run judge0::get_info
    assert_success
    [[ "$output" =~ "Judge0" ]]
}

# =============================================================================
# Installation Tests
# =============================================================================

@test "Judge0: Installation checks requirements" {
    skip "Full installation test - requires Docker"
    
    # Mock Docker check failure
    docker() { return 1; }
    export -f docker
    
    run judge0::install
    assert_failure
    [[ "$output" =~ "Docker is not running" ]]
}

@test "Judge0: Installation creates required directories" {
    # Just test directory creation logic
    run bash -c "source ${BATS_TEST_DIRNAME}/lib/common.sh && judge0::create_directories"
    assert_success
    
    [ -d "$JUDGE0_DATA_DIR" ]
    [ -d "$JUDGE0_CONFIG_DIR" ]
    [ -d "${JUDGE0_DATA_DIR}/logs" ]
    [ -d "${JUDGE0_DATA_DIR}/submissions" ]
}

@test "Judge0: API key generation" {
    run bash -c "source ${BATS_TEST_DIRNAME}/lib/common.sh && judge0::generate_api_key"
    assert_success
    
    # Check key length (should be 32 chars by default)
    [ ${#output} -eq 32 ]
}

@test "Judge0: API key storage and retrieval" {
    local test_key="test-api-key-12345"
    
    # Save key
    run bash -c "source ${BATS_TEST_DIRNAME}/lib/common.sh && judge0::save_api_key '$test_key'"
    assert_success
    
    # Retrieve key
    run bash -c "source ${BATS_TEST_DIRNAME}/lib/common.sh && judge0::get_api_key"
    assert_success
    assert_output "$test_key"
    
    # Check file permissions
    [ -f "${JUDGE0_CONFIG_DIR}/api_key" ]
    local perms=$(stat -c %a "${JUDGE0_CONFIG_DIR}/api_key" 2>/dev/null || stat -f %A "${JUDGE0_CONFIG_DIR}/api_key")
    [ "$perms" = "600" ]
}

# =============================================================================
# Configuration Tests
# =============================================================================

@test "Judge0: Configuration defaults are loaded" {
    run bash -c "source ${BATS_TEST_DIRNAME}/config/defaults.sh && echo \$JUDGE0_PORT"
    assert_success
    assert_output "2358"
}

@test "Judge0: Security limits configuration" {
    run bash -c "source ${BATS_TEST_DIRNAME}/config/defaults.sh && echo \$JUDGE0_CPU_TIME_LIMIT"
    assert_success
    assert_output "10"
    
    run bash -c "source ${BATS_TEST_DIRNAME}/config/defaults.sh && echo \$JUDGE0_MEMORY_LIMIT"
    assert_success
    assert_output "262144"
}

@test "Judge0: Language ID lookup" {
    run bash -c "source ${BATS_TEST_DIRNAME}/config/defaults.sh && judge0::get_language_id 'javascript'"
    assert_success
    assert_output "93"
    
    run bash -c "source ${BATS_TEST_DIRNAME}/config/defaults.sh && judge0::get_language_id 'python'"
    assert_success
    assert_output "92"
}

# =============================================================================
# Docker Management Tests
# =============================================================================

@test "Judge0: Docker compose file generation" {
    skip "Requires full environment setup"
    
    # Generate API key first
    export JUDGE0_API_KEY="test-key-12345"
    
    # Test compose file generation
    run bash -c "source ${BATS_TEST_DIRNAME}/lib/install.sh && judge0::install::create_compose_file"
    assert_success
    
    [ -f "${JUDGE0_CONFIG_DIR}/docker-compose.yml" ]
    
    # Check compose file contains required services
    grep -q "judge0-server:" "${JUDGE0_CONFIG_DIR}/docker-compose.yml"
    grep -q "judge0-workers:" "${JUDGE0_CONFIG_DIR}/docker-compose.yml"
    grep -q "judge0-db:" "${JUDGE0_CONFIG_DIR}/docker-compose.yml"
    grep -q "judge0-redis:" "${JUDGE0_CONFIG_DIR}/docker-compose.yml"
}

# =============================================================================
# Status and Health Check Tests  
# =============================================================================

@test "Judge0: Status command when not installed" {
    # Mock docker to return not installed
    docker() {
        case "$*" in
            *"container inspect"*) return 1 ;;
            *) return 1 ;;
        esac
    }
    export -f docker
    
    run judge0::show_status
    assert_failure
    [[ "$output" =~ "not installed" ]]
}

@test "Judge0: Health check JSON format" {
    run bash -c "source ${BATS_TEST_DIRNAME}/lib/status.sh && judge0::status::get_health_json"
    assert_success
    
    # Verify JSON structure
    echo "$output" | jq -e '.status' >/dev/null
    echo "$output" | jq -e '.api' >/dev/null
    echo "$output" | jq -e '.workers' >/dev/null
}

# =============================================================================
# API Tests (Mocked)
# =============================================================================

@test "Judge0: Language list parsing" {
    # Test language validation
    run judge0::validate_language 'javascript'
    assert_success
}

@test "Judge0: Code submission parameter validation" {
    skip "Requires API mock"
    
    # Test that submission requires code
    run judge0::submit_code "" "javascript"
    assert_failure
    [[ "$output" =~ "Code is required" ]]
}

# =============================================================================
# Security Tests
# =============================================================================

@test "Judge0: Security configuration validation" {
    run bash -c "source ${BATS_TEST_DIRNAME}/config/defaults.sh && echo \$JUDGE0_ENABLE_NETWORK"
    assert_success
    assert_output "false"
    
    run bash -c "source ${BATS_TEST_DIRNAME}/config/defaults.sh && echo \$JUDGE0_ENABLE_CALLBACKS"
    assert_success  
    assert_output "false"
}

@test "Judge0: Resource limits enforcement" {
    # Check default limits are reasonable
    run bash -c "source ${BATS_TEST_DIRNAME}/config/defaults.sh && test \$JUDGE0_CPU_TIME_LIMIT -le 30"
    assert_success
    
    run bash -c "source ${BATS_TEST_DIRNAME}/config/defaults.sh && test \$JUDGE0_MEMORY_LIMIT -le 524288"
    assert_success
}

# =============================================================================
# Usage and Examples Tests
# =============================================================================

@test "Judge0: Usage information completeness" {
    # Mock docker to return not running
    docker() { return 1; }
    export -f docker
    
    run judge0::show_usage
    assert_failure  # Not running
    [[ "$output" =~ "not running" ]]
}

@test "Judge0: Example files exist" {
    [ -f "${BATS_TEST_DIRNAME}/examples/basic/hello-world.js" ]
    [ -f "${BATS_TEST_DIRNAME}/examples/basic/input-output.py" ]
    [ -f "${BATS_TEST_DIRNAME}/examples/test-judge0.sh" ]
}

@test "Judge0: Example test script is executable" {
    [ -x "${BATS_TEST_DIRNAME}/examples/test-judge0.sh" ]
}

# =============================================================================
# Error Handling Tests
# =============================================================================

@test "Judge0: Handles invalid action gracefully" {
    run judge0::handle_action "invalid-action"
    assert_failure
    [[ "$output" =~ "Unknown action" ]]
}

@test "Judge0: Handles missing required parameters" {
    run judge0::submit_code "" ""
    assert_failure
    [[ "$output" =~ "Code is required" ]]
}

@test "Judge0: Handles invalid language gracefully" {
    skip "Requires running instance"
    
    run judge0::submit_code "print('test')" "invalid-lang"
    assert_failure
    [[ "$output" =~ "Language not supported" ]]
}

# =============================================================================
# Integration Tests (Skipped by default)
# =============================================================================

@test "Judge0: Full installation and execution flow" {
    skip "Full integration test - enable for comprehensive testing"
    
    # Install
    run_manage --action install --yes yes --workers 1
    assert_success
    
    # Check status
    run_manage --action status
    assert_success
    assert_output --partial "running"
    
    # List languages
    run_manage --action languages
    assert_success
    assert_output --partial "JavaScript"
    
    # Submit code
    run_manage --action submit --code 'console.log("Hello, Judge0!");' --language javascript
    assert_success
    assert_output --partial "Hello, Judge0!"
    
    # Uninstall
    run_manage --action uninstall --yes yes --force yes
    assert_success
}

# =============================================================================
# Cleanup Tests
# =============================================================================

@test "Judge0: Cleanup removes all data" {
    # Create test data
    mkdir -p "${JUDGE0_DATA_DIR}/submissions"
    touch "${JUDGE0_DATA_DIR}/submissions/test.json"
    
    # Run cleanup
    run bash -c "source ${BATS_TEST_DIRNAME}/lib/common.sh && judge0::cleanup_data yes"
    assert_success
    
    # Verify removal
    [ ! -d "$JUDGE0_DATA_DIR" ]
}

# =============================================================================
# Docker-specific Tests (when Docker available)
# =============================================================================

@test "Judge0: Docker network creation" {
    if ! command -v docker >/dev/null; then
        skip "Docker not available"
    fi
    
    run docker network create "$JUDGE0_NETWORK_NAME"
    assert_success
    
    # Cleanup
    docker network rm "$JUDGE0_NETWORK_NAME"
}

@test "Judge0: Docker volume creation" {
    if ! command -v docker >/dev/null; then
        skip "Docker not available"
    fi
    
    run docker volume create "$JUDGE0_VOLUME_NAME"
    assert_success
    
    # Cleanup
    docker volume rm "$JUDGE0_VOLUME_NAME"
}