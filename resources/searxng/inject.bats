#!/usr/bin/env bats
# Tests for SearXNG inject.sh script

# Source var.sh to get test paths
source "${BATS_TEST_DIRNAME}/../../../lib/utils/var.sh"

# Source trash module for safe test cleanup
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Load Vrooli test infrastructure
source "${var_SCRIPTS_TEST_DIR}/fixtures/setup.bash"

# Setup for each test
setup() {
    # Setup standard mocks
    vrooli_auto_setup
    
    # Set SearXNG-specific test environment
    export SEARXNG_HOST="http://localhost:8080"
    export SEARXNG_DATA_DIR="${BATS_TMPDIR}/searxng_test"
    export SEARXNG_SETTINGS_DIR="${SEARXNG_DATA_DIR}/settings"
    export SEARXNG_ENGINES_DIR="${SEARXNG_DATA_DIR}/engines"
    
    # Create test directories
    mkdir -p "${SEARXNG_SETTINGS_DIR}"
    mkdir -p "${SEARXNG_ENGINES_DIR}"
    
    # Load the script without executing main
    SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    source "${SCRIPT_DIR}/inject.sh" || true
}

# BATS teardown function - runs after each test
teardown() {
    vrooli_cleanup_test
    
    # Clean up test directories
    trash::safe_remove "${BATS_TMPDIR}/searxng_test" --test-cleanup
}

# ============================================================================
# Script Loading Tests
# ============================================================================

@test "inject.sh script loads without errors" {
    # Script loading happens in setup, this verifies it worked
    declare -f searxng_inject::usage > /dev/null
    [ "$?" -eq 0 ]
}

@test "inject.sh defines all required functions" {
    declare -f searxng_inject::usage > /dev/null
    [ "$?" -eq 0 ]
    declare -f searxng_inject::validate_config > /dev/null
    [ "$?" -eq 0 ]
    declare -f searxng_inject::inject_data > /dev/null
    [ "$?" -eq 0 ]
    declare -f searxng_inject::check_status > /dev/null
    [ "$?" -eq 0 ]
}

# ============================================================================
# Configuration Validation Tests
# ============================================================================

@test "validate_config accepts valid engine configuration" {
    local config='{"engines": [{"name": "google", "enabled": true}]}'
    
    run searxng_inject::validate_config "$config"
    [ "$status" -eq 0 ]
}

@test "validate_config rejects invalid JSON" {
    local invalid_config='{"engines": [invalid json}'
    
    run searxng_inject::validate_config "$invalid_config"
    [ "$status" -eq 1 ]
}

@test "validate_config rejects empty configuration" {
    local empty_config='{}'
    
    run searxng_inject::validate_config "$empty_config"
    [ "$status" -eq 1 ]
}

@test "validate_config accepts preferences configuration" {
    local config='{"preferences": {"language": "en", "theme": "simple"}}'
    
    run searxng_inject::validate_config "$config"
    [ "$status" -eq 0 ]
}

@test "validate_config accepts mixed configuration" {
    local config='{
        "engines": [{"name": "google", "enabled": true}],
        "preferences": {"language": "en"},
        "server": {"secret_key": "test-key"}
    }'
    
    run searxng_inject::validate_config "$config"
    [ "$status" -eq 0 ]
}

# ============================================================================
# Engine Configuration Tests
# ============================================================================

@test "configure_engine creates engine file" {
    local engine_config='{"name": "test-engine", "enabled": true, "weight": 1.5}'
    
    run searxng_inject::configure_engine "$engine_config"
    [ "$status" -eq 0 ]
    [ -f "${SEARXNG_ENGINES_DIR}/test-engine.yml" ]
}

@test "configure_engine handles missing optional fields" {
    local engine_config='{"name": "minimal-engine"}'
    
    run searxng_inject::configure_engine "$engine_config"
    [ "$status" -eq 0 ]
    [ -f "${SEARXNG_ENGINES_DIR}/minimal-engine.yml" ]
    
    # Check that defaults are applied
    grep -q "enabled: true" "${SEARXNG_ENGINES_DIR}/minimal-engine.yml"
    grep -q "weight: 1.0" "${SEARXNG_ENGINES_DIR}/minimal-engine.yml"
}

# ============================================================================
# Preferences Configuration Tests
# ============================================================================

@test "apply_preferences creates preference files" {
    local preferences='{"language": "en", "theme": "simple", "safe_search": 1}'
    
    run searxng_inject::apply_preferences "$preferences"
    [ "$status" -eq 0 ]
    [ -f "${SEARXNG_SETTINGS_DIR}/preferences.json" ]
    [ -f "${SEARXNG_SETTINGS_DIR}/preferences.yml" ]
}

@test "apply_preferences handles empty preferences" {
    local preferences='{}'
    
    run searxng_inject::apply_preferences "$preferences"
    [ "$status" -eq 0 ]
    [ -f "${SEARXNG_SETTINGS_DIR}/preferences.json" ]
}

# ============================================================================
# Server Settings Tests
# ============================================================================

@test "apply_server creates server configuration" {
    local server='{"secret_key": "test-secret", "limiter": true, "public_instance": false}'
    
    run searxng_inject::apply_server "$server"
    [ "$status" -eq 0 ]
    [ -f "${SEARXNG_SETTINGS_DIR}/server.yml" ]
    
    # Check that secret key is properly set
    grep -q "secret_key: \"test-secret\"" "${SEARXNG_SETTINGS_DIR}/server.yml"
}

# ============================================================================
# Integration Tests
# ============================================================================

@test "inject_data processes complete configuration" {
    # Mock curl to avoid external dependencies
    curl() {
        case "$*" in
            *"healthz"*) echo "OK"; return 0;;
            *) return 1;;
        esac
    }
    export -f curl
    
    local config='{
        "engines": [
            {"name": "google", "enabled": true},
            {"name": "duckduckgo", "enabled": false}
        ],
        "preferences": {"language": "en", "theme": "simple"},
        "server": {"secret_key": "integration-test-key"}
    }'
    
    run searxng_inject::inject_data "$config"
    [ "$status" -eq 0 ]
    
    # Check that files were created
    [ -f "${SEARXNG_ENGINES_DIR}/google.yml" ]
    [ -f "${SEARXNG_ENGINES_DIR}/duckduckgo.yml" ]
    [ -f "${SEARXNG_SETTINGS_DIR}/preferences.json" ]
    [ -f "${SEARXNG_SETTINGS_DIR}/server.yml" ]
}

@test "inject_data handles rollback on failure" {
    # Mock a function to fail
    searxng_inject::configure_engine() {
        return 1
    }
    export -f searxng_inject::configure_engine
    
    local config='{"engines": [{"name": "fail-engine", "enabled": true}]}'
    
    run searxng_inject::inject_data "$config"
    [ "$status" -eq 1 ]
}

# ============================================================================
# Status Check Tests  
# ============================================================================

@test "check_status reports configuration status" {
    # Create some test files
    echo '{"language": "en"}' > "${SEARXNG_SETTINGS_DIR}/preferences.json"
    echo "enabled: true" > "${SEARXNG_ENGINES_DIR}/test.yml"
    
    local config='{"engines": [{"name": "test", "enabled": true}], "preferences": {"language": "en"}}'
    
    run searxng_inject::check_status "$config"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Found 1 engine configurations" ]]
    [[ "$output" =~ "Preferences configured" ]]
}

# ============================================================================
# Error Handling Tests
# ============================================================================

@test "main function handles missing configuration" {
    run searxng_inject::main "--validate"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Configuration JSON required" ]]
}

@test "main function handles unknown action" {
    local config='{"engines": []}'
    
    run searxng_inject::main "--unknown" "$config"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Unknown action" ]]
}

@test "usage function displays help" {
    run searxng_inject::usage
    [ "$status" -eq 0 ]
    [[ "$output" =~ "SearXNG Data Injection Adapter" ]]
    [[ "$output" =~ "USAGE:" ]]
}