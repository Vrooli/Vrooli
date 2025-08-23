#!/usr/bin/env bats
# Tests for Vault lib/common.sh functions

# Load Vrooli test infrastructure
# shellcheck disable=SC1091
source "${BATS_TEST_DIRNAME}/../../../../__test/setup.bash"

# Expensive setup operations run once per file
setup_file() {
    # Load var.sh to get directory variables
    # shellcheck disable=SC1091
    source "${BATS_TEST_DIRNAME}/../../../../lib/utils/var.sh"
    
    # Use Vrooli service test setup for vault
    vrooli_setup_service_test "vault"
    
    # Load Vault specific configuration once per file
    # shellcheck disable=SC1091
    source "$var_SCRIPTS_RESOURCES_DIR/storage/vault/config/defaults.sh"
    # shellcheck disable=SC1091
    source "$var_SCRIPTS_RESOURCES_DIR/storage/vault/config/messages.sh"
    # shellcheck disable=SC1091
    source "$var_SCRIPTS_RESOURCES_DIR/storage/vault/lib/common.sh"
}

# Lightweight per-test setup
setup() {
    # Setup standard Vrooli mocks
    vrooli_auto_setup
    
    # Set test environment variables
    export VAULT_PORT="8200"
    export VAULT_CONTAINER_NAME="vault-test"
    export VAULT_BASE_URL="http://localhost:8200"
    export VAULT_MODE="dev"
    export VAULT_DATA_DIR="/tmp/vault-test/data"
    export VAULT_CONFIG_DIR="/tmp/vault-test/config"
    export VAULT_LOGS_DIR="/tmp/vault-test/logs"
    export VAULT_DEV_ROOT_TOKEN_ID="myroot"
    export VAULT_STARTUP_MAX_WAIT="30"
    export VAULT_STARTUP_WAIT_INTERVAL="1"
    export VAULT_TOKEN_FILE="/tmp/vault-test/config/root-token"
    export VAULT_NAMESPACE_PREFIX="vrooli"
    export VAULT_SECRET_ENGINE="secret"
    
    # Export config functions
    vault::export_config
    vault::export_messages
}

# BATS teardown function - runs after each test
teardown() {
    vrooli_cleanup_test
}

# ============================================================================
# Container Status Tests
# ============================================================================

@test "vault::is_installed returns 0 when container exists" {
    mock::docker::set_container_state "$VAULT_CONTAINER_NAME" "running" "hashicorp/vault:1.17"
    
    run vault::is_installed
    [ "$status" -eq 0 ]
}

@test "vault::is_installed returns 1 when container does not exist" {
    # No container state set, so it won't exist
    
    run vault::is_installed
    [ "$status" -eq 1 ]
}

@test "vault::is_running returns 0 when container is running" {
    mock::docker::set_container_state "$VAULT_CONTAINER_NAME" "running" "hashicorp/vault:1.17"
    
    run vault::is_running
    [ "$status" -eq 0 ]
}

@test "vault::is_running returns 1 when container is stopped" {
    mock::docker::set_container_state "$VAULT_CONTAINER_NAME" "stopped" "hashicorp/vault:1.17"
    
    run vault::is_running
    [ "$status" -eq 1 ]
}

@test "vault::is_running returns 1 when container does not exist" {
    # No container state set
    
    run vault::is_running
    [ "$status" -eq 1 ]
}

# ============================================================================
# Health Check Tests
# ============================================================================

@test "vault::is_healthy returns 0 when vault is running and responding" {
    mock::docker::set_container_state "$VAULT_CONTAINER_NAME" "running" "hashicorp/vault:1.17"
    mock::http::set_response "GET http://localhost:8200/v1/sys/health" "200" '{"initialized": true, "sealed": false}'
    
    run vault::is_healthy
    [ "$status" -eq 0 ]
}

@test "vault::is_healthy returns 1 when vault is not running" {
    mock::docker::set_container_state "$VAULT_CONTAINER_NAME" "stopped" "hashicorp/vault:1.17"
    
    run vault::is_healthy
    [ "$status" -eq 1 ]
}

@test "vault::is_healthy returns 1 when vault is running but not responding" {
    mock::docker::set_container_state "$VAULT_CONTAINER_NAME" "running" "hashicorp/vault:1.17"
    mock::http::set_response "GET http://localhost:8200/v1/sys/health" "500" '{"errors": ["internal error"]}'
    
    run vault::is_healthy
    [ "$status" -eq 1 ]
}

# ============================================================================
# Initialization Tests
# ============================================================================

@test "vault::is_initialized returns 0 when vault is initialized" {
    mock::docker::set_container_state "$VAULT_CONTAINER_NAME" "running" "hashicorp/vault:1.17"
    mock::http::set_response "GET http://localhost:8200/v1/sys/health" "200" '{"initialized": true, "sealed": false}'
    mock::http::set_response "GET http://localhost:8200/v1/sys/init" "200" '{"initialized": true}'
    
    run vault::is_initialized
    [ "$status" -eq 0 ]
}

@test "vault::is_initialized returns 1 when vault is not initialized" {
    mock::docker::set_container_state "$VAULT_CONTAINER_NAME" "running" "hashicorp/vault:1.17"
    mock::http::set_response "GET http://localhost:8200/v1/sys/health" "200" '{"initialized": false, "sealed": true}'
    mock::http::set_response "GET http://localhost:8200/v1/sys/init" "200" '{"initialized": false}'
    
    run vault::is_initialized
    [ "$status" -eq 1 ]
}

@test "vault::is_initialized returns 1 when vault is not healthy" {
    mock::docker::set_container_state "$VAULT_CONTAINER_NAME" "stopped" "hashicorp/vault:1.17"
    
    run vault::is_initialized
    [ "$status" -eq 1 ]
}

# ============================================================================
# Seal Status Tests
# ============================================================================

@test "vault::is_sealed returns 0 when vault is sealed" {
    mock::docker::set_container_state "$VAULT_CONTAINER_NAME" "running" "hashicorp/vault:1.17"
    mock::http::set_response "GET http://localhost:8200/v1/sys/health" "200" '{"initialized": true, "sealed": true}'
    mock::http::set_response "GET http://localhost:8200/v1/sys/seal-status" "200" '{"sealed": true}'
    
    run vault::is_sealed
    [ "$status" -eq 0 ]
}

@test "vault::is_sealed returns 1 when vault is unsealed" {
    mock::docker::set_container_state "$VAULT_CONTAINER_NAME" "running" "hashicorp/vault:1.17"
    mock::http::set_response "GET http://localhost:8200/v1/sys/health" "200" '{"initialized": true, "sealed": false}'
    mock::http::set_response "GET http://localhost:8200/v1/sys/seal-status" "200" '{"sealed": false}'
    
    run vault::is_sealed
    [ "$status" -eq 1 ]
}

@test "vault::is_sealed returns 0 when vault is not healthy" {
    mock::docker::set_container_state "$VAULT_CONTAINER_NAME" "stopped" "hashicorp/vault:1.17"
    
    run vault::is_sealed
    [ "$status" -eq 0 ]  # Assume sealed for safety when can't determine status
}

# ============================================================================
# Status String Tests
# ============================================================================

@test "vault::get_status returns 'not_installed' when container does not exist" {
    # No container state set
    
    run vault::get_status
    [ "$status" -eq 0 ]
    [ "$output" = "not_installed" ]
}

@test "vault::get_status returns 'stopped' when container exists but is stopped" {
    mock::docker::set_container_state "$VAULT_CONTAINER_NAME" "stopped" "hashicorp/vault:1.17"
    
    run vault::get_status
    [ "$status" -eq 0 ]
    [ "$output" = "stopped" ]
}

@test "vault::get_status returns 'unhealthy' when container is running but not responding" {
    mock::docker::set_container_state "$VAULT_CONTAINER_NAME" "running" "hashicorp/vault:1.17"
    mock::http::set_response "GET http://localhost:8200/v1/sys/health" "500" '{"errors": ["internal error"]}'
    
    run vault::get_status
    [ "$status" -eq 0 ]
    [ "$output" = "unhealthy" ]
}

@test "vault::get_status returns 'sealed' when vault is healthy but sealed" {
    mock::docker::set_container_state "$VAULT_CONTAINER_NAME" "running" "hashicorp/vault:1.17"
    mock::http::set_response "GET http://localhost:8200/v1/sys/health" "200" '{"initialized": true, "sealed": true}'
    mock::http::set_response "GET http://localhost:8200/v1/sys/seal-status" "200" '{"sealed": true}'
    
    run vault::get_status
    [ "$status" -eq 0 ]
    [ "$output" = "sealed" ]
}

@test "vault::get_status returns 'healthy' when vault is fully operational" {
    mock::docker::set_container_state "$VAULT_CONTAINER_NAME" "running" "hashicorp/vault:1.17"
    mock::http::set_response "GET http://localhost:8200/v1/sys/health" "200" '{"initialized": true, "sealed": false}'
    mock::http::set_response "GET http://localhost:8200/v1/sys/seal-status" "200" '{"sealed": false}'
    
    run vault::get_status
    [ "$status" -eq 0 ]
    [ "$output" = "healthy" ]
}

# ============================================================================
# Directory Management Tests
# ============================================================================

@test "vault::ensure_directories creates required directories" {
    # Test directories will be cleaned up by teardown
    
    run vault::ensure_directories
    [ "$status" -eq 0 ]
    
    # Check that directories were created
    [ -d "$VAULT_DATA_DIR" ]
    [ -d "$VAULT_CONFIG_DIR" ]
    [ -d "$VAULT_LOGS_DIR" ]
}

# ============================================================================
# Configuration Tests
# ============================================================================

@test "vault::create_config creates dev configuration" {
    run vault::create_config "dev"
    [ "$status" -eq 0 ]
    
    # Check that config file was created
    [ -f "${VAULT_CONFIG_DIR}/vault.hcl" ]
    
    # Check that it contains dev mode comment
    run cat "${VAULT_CONFIG_DIR}/vault.hcl"
    [[ "$output" =~ "Development Configuration" ]]
}

@test "vault::create_config creates prod configuration" {
    run vault::create_config "prod"
    [ "$status" -eq 0 ]
    
    # Check that config file was created
    [ -f "${VAULT_CONFIG_DIR}/vault.hcl" ]
    
    # Check that it contains production settings
    run cat "${VAULT_CONFIG_DIR}/vault.hcl"
    [[ "$output" =~ "Production Configuration" ]]
    [[ "$output" =~ "storage \"file\"" ]]
    [[ "$output" =~ "listener \"tcp\"" ]]
}

@test "vault::create_config fails with unknown mode" {
    run vault::create_config "unknown"
    [ "$status" -eq 1 ]
}

# ============================================================================
# Secret Path Validation Tests
# ============================================================================

@test "vault::validate_secret_path accepts valid path" {
    run vault::validate_secret_path "myapp/config/database"
    [ "$status" -eq 0 ]
}

@test "vault::validate_secret_path rejects empty path" {
    run vault::validate_secret_path ""
    [ "$status" -eq 1 ]
}

@test "vault::validate_secret_path rejects path with spaces" {
    run vault::validate_secret_path "my app/config"
    [ "$status" -eq 1 ]
}

@test "vault::validate_secret_path rejects path starting with slash" {
    run vault::validate_secret_path "/myapp/config"
    [ "$status" -eq 1 ]
}

# ============================================================================
# Secret Path Construction Tests
# ============================================================================

@test "vault::construct_secret_path builds correct full path" {
    run vault::construct_secret_path "myapp/config"
    [ "$status" -eq 0 ]
    [ "$output" = "secret/data/vrooli/myapp/config" ]
}

@test "vault::construct_secret_path fails with invalid path" {
    run vault::construct_secret_path "/invalid/path"
    [ "$status" -eq 1 ]
}

# ============================================================================
# Token Management Tests
# ============================================================================

@test "vault::get_root_token returns dev token in dev mode" {
    export VAULT_MODE="dev"
    export VAULT_DEV_ROOT_TOKEN_ID="test-root-token"
    
    run vault::get_root_token
    [ "$status" -eq 0 ]
    [ "$output" = "test-root-token" ]
}

@test "vault::get_root_token reads from file in prod mode" {
    export VAULT_MODE="prod"
    mkdir -p "$(dirname "$VAULT_TOKEN_FILE")"
    echo "prod-root-token" > "$VAULT_TOKEN_FILE"
    
    run vault::get_root_token
    [ "$status" -eq 0 ]
    [ "$output" = "prod-root-token" ]
}

@test "vault::get_root_token fails when token file does not exist in prod mode" {
    export VAULT_MODE="prod"
    export VAULT_TOKEN_FILE="/nonexistent/token-file"
    
    run vault::get_root_token
    [ "$status" -eq 1 ]
}

# ============================================================================
# Utility Function Tests
# ============================================================================

@test "vault::generate_random_string generates string of correct length" {
    run vault::generate_random_string 16
    [ "$status" -eq 0 ]
    [ "${#output}" -eq 16 ]
}

@test "vault::generate_random_string uses default length when no argument provided" {
    run vault::generate_random_string
    [ "$status" -eq 0 ]
    [ "${#output}" -eq 32 ]
}

# ============================================================================
# Wait Function Tests
# ============================================================================

@test "vault::wait_for_health returns 0 when vault becomes healthy quickly" {
    mock::docker::set_container_state "$VAULT_CONTAINER_NAME" "running" "hashicorp/vault:1.17"
    mock::http::set_response "GET http://localhost:8200/v1/sys/health" "200" '{"initialized": true, "sealed": false}'
    
    run vault::wait_for_health 5 1
    [ "$status" -eq 0 ]
}

@test "vault::wait_for_health returns 1 when timeout is reached" {
    # Don't set up any container state, so vault will never be healthy
    
    run vault::wait_for_health 2 1
    [ "$status" -eq 1 ]
}