#!/usr/bin/env bats
# Judge0 Resource Management Tests
# Tests for Judge0 installation, configuration, and management

# Test environment setup
setup() {
    # Load test helper
    load '../../../helpers/test/test_helper'
    
    # Set test environment
    export TEST_MODE="true"
    export JUDGE0_TEST_MODE="true"
    
    # Store original values
    export ORIGINAL_JUDGE0_PORT="${JUDGE0_PORT:-2358}"
    export ORIGINAL_JUDGE0_API_KEY="${JUDGE0_API_KEY:-}"
    
    # Use test port to avoid conflicts
    export JUDGE0_PORT="12358"
    export JUDGE0_CONTAINER_NAME="test-judge0-server"
    export JUDGE0_WORKERS_NAME="test-judge0-workers"
    export JUDGE0_NETWORK_NAME="test-judge0-network"
    export JUDGE0_VOLUME_NAME="test-judge0-data"
    export JUDGE0_DATA_DIR="${BATS_TEST_TMPDIR}/judge0"
    export JUDGE0_CONFIG_DIR="${JUDGE0_DATA_DIR}/config"
    
    # Create test directories
    mkdir -p "$JUDGE0_CONFIG_DIR"
}

# Test environment cleanup
teardown() {
    # Cleanup test containers
    docker stop "$JUDGE0_CONTAINER_NAME" >/dev/null 2>&1 || true
    docker rm "$JUDGE0_CONTAINER_NAME" >/dev/null 2>&1 || true
    docker stop "${JUDGE0_CONTAINER_NAME}-db" >/dev/null 2>&1 || true
    docker rm "${JUDGE0_CONTAINER_NAME}-db" >/dev/null 2>&1 || true
    docker stop "${JUDGE0_CONTAINER_NAME}-redis" >/dev/null 2>&1 || true
    docker rm "${JUDGE0_CONTAINER_NAME}-redis" >/dev/null 2>&1 || true
    
    # Cleanup workers
    for i in {1..5}; do
        docker stop "${JUDGE0_WORKERS_NAME}-${i}" >/dev/null 2>&1 || true
        docker rm "${JUDGE0_WORKERS_NAME}-${i}" >/dev/null 2>&1 || true
    done
    
    # Cleanup network and volume
    docker network rm "$JUDGE0_NETWORK_NAME" >/dev/null 2>&1 || true
    docker volume rm "$JUDGE0_VOLUME_NAME" >/dev/null 2>&1 || true
    
    # Restore original values
    export JUDGE0_PORT="$ORIGINAL_JUDGE0_PORT"
    export JUDGE0_API_KEY="$ORIGINAL_JUDGE0_API_KEY"
    
    # Cleanup test directories
    rm -rf "$JUDGE0_DATA_DIR"
}

# Helper function to run manage.sh
run_manage() {
    run "${BATS_TEST_DIRNAME}/manage.sh" "$@"
}

# =============================================================================
# Basic Functionality Tests
# =============================================================================

@test "Judge0: Script exists and is executable" {
    [ -f "${BATS_TEST_DIRNAME}/manage.sh" ]
    [ -x "${BATS_TEST_DIRNAME}/manage.sh" ]
}

@test "Judge0: Help command displays usage" {
    run_manage --help
    assert_success
    assert_output --partial "Judge0 Code Execution"
    assert_output --partial "USAGE:"
    assert_output --partial "OPTIONS:"
}

@test "Judge0: Version information available" {
    run_manage --action info --yes yes
    # Should fail if not installed, but show proper error
    assert_output --partial "Judge0"
}

# =============================================================================
# Installation Tests
# =============================================================================

@test "Judge0: Installation checks requirements" {
    skip "Full installation test - requires Docker"
    
    # Mock Docker check failure
    docker() { return 1; }
    export -f docker
    
    run_manage --action install --yes yes
    assert_failure
    assert_output --partial "Docker is not running"
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
    assert_output "5"
    
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
    run_manage --action status --yes yes
    assert_failure
    assert_output --partial "not installed"
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
    run bash -c "source ${BATS_TEST_DIRNAME}/lib/common.sh && source ${BATS_TEST_DIRNAME}/config/defaults.sh && judge0::validate_language 'javascript'"
    assert_success
}

@test "Judge0: Code submission parameter validation" {
    skip "Requires API mock"
    
    # Test that submission requires code
    run_manage --action submit --language javascript --yes yes
    assert_failure
    assert_output --partial "Code is required"
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
    run_manage --action usage --yes yes
    assert_failure  # Not running
    assert_output --partial "not running"
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
    run_manage --action invalid-action --yes yes
    assert_failure
    assert_output --partial "Unknown action"
}

@test "Judge0: Handles missing required parameters" {
    run_manage --action submit --yes yes
    assert_failure
    assert_output --partial "Code is required"
}

@test "Judge0: Handles invalid language gracefully" {
    skip "Requires running instance"
    
    run_manage --action submit --code "print('test')" --language "invalid-lang" --yes yes
    assert_failure
    assert_output --partial "Language not supported"
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