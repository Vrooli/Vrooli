#!/usr/bin/env bats
# Tests for secrets.sh - Secret management utilities

bats_require_minimum_version 1.5.0

# Load test infrastructure
source "${BATS_TEST_DIRNAME}/../../__test/fixtures/setup.bash"

# Load BATS helpers
load "${BATS_TEST_DIRNAME}/../../__test/helpers/bats-support/load"
load "${BATS_TEST_DIRNAME}/../../__test/helpers/bats-assert/load"

setup() {
    vrooli_setup_unit_test
    
    # Source the secrets utilities
    source "${BATS_TEST_DIRNAME}/secrets.sh"
    
    # Create mock project structure
    export MOCK_PROJECT_ROOT="${MOCK_TMP_DIR}/project"
    mkdir -p "${MOCK_PROJECT_ROOT}/.vrooli"
    
    # Override var variables for testing
    export var_VROOLI_CONFIG_DIR="${MOCK_PROJECT_ROOT}/.vrooli"
    export var_SERVICE_JSON_FILE="${MOCK_PROJECT_ROOT}/.vrooli/service.json"
}

teardown() {
    vrooli_cleanup_test
}

@test "secrets::get_project_root finds project root" {
    run secrets::get_project_root
    assert_success
    assert_output --partial "project"
}

@test "secrets::get_secrets_file returns correct path" {
    run secrets::get_secrets_file
    assert_success
    assert_output "${MOCK_PROJECT_ROOT}/.vrooli/secrets.json"
}

@test "secrets::get_service_file returns correct path" {
    run secrets::get_service_file
    assert_success
    assert_output "${MOCK_PROJECT_ROOT}/.vrooli/service.json"
}

@test "secrets::resolve tries environment variables as fallback" {
    export TEST_SECRET="test_value"
    
    run secrets::resolve "TEST_SECRET"
    assert_success
    assert_output "test_value"
    
    unset TEST_SECRET
}

@test "secrets::resolve returns failure for missing secrets" {
    run secrets::resolve "NONEXISTENT_SECRET"
    assert_failure
}

@test "secrets::resolve uses project secrets.json" {
    # Create mock secrets file
    cat > "${MOCK_PROJECT_ROOT}/.vrooli/secrets.json" <<EOF
{
  "API_KEY": "secret_api_key"
}
EOF
    
    run secrets::resolve "API_KEY"
    assert_success
    assert_output "secret_api_key"
}

@test "secrets::save_key creates secrets file" {
    run secrets::save_key "NEW_SECRET" "new_value"
    assert_success
    
    # Verify file was created with correct content
    assert [ -f "${MOCK_PROJECT_ROOT}/.vrooli/secrets.json" ]
    
    # Check that the secret was saved
    run jq -r '.NEW_SECRET' "${MOCK_PROJECT_ROOT}/.vrooli/secrets.json"
    assert_success
    assert_output "new_value"
}

@test "secrets::save_key validates required parameters" {
    run secrets::save_key "" "value"
    assert_failure
    assert_output --partial "Secret key name is required"
    
    run secrets::save_key "KEY" ""
    assert_failure
    assert_output --partial "Secret value is required"
}

@test "secrets::substitute_json replaces secret placeholders" {
    export TEST_SECRET="replaced_value"
    
    local json_input='{"config": "{{TEST_SECRET}}"}'
    run secrets::substitute_json "$json_input"
    assert_success
    assert_output --partial "replaced_value"
    
    unset TEST_SECRET
}

@test "secrets::substitute_service_references handles service URLs" {
    # Mock RESOURCE_PORTS array
    declare -gA RESOURCE_PORTS=(["ollama"]="11434")
    
    local json_input='{"url": "${service.ollama.url}"}'
    run secrets::substitute_service_references "$json_input"
    assert_success
    assert_output --partial "http://localhost:11434"
}

@test "secrets::substitute_all_templates processes both patterns" {
    export TEST_SECRET="test_value"
    declare -gA RESOURCE_PORTS=(["ollama"]="11434")
    
    local json_input='{"secret": "{{TEST_SECRET}}", "url": "${service.ollama.url}"}'
    run secrets::substitute_all_templates "$json_input"
    assert_success
    assert_output --partial "test_value"
    assert_output --partial "http://localhost:11434"
    
    unset TEST_SECRET
}

@test "secrets::validate_storage checks file permissions" {
    # Create secrets file with wrong permissions
    cat > "${MOCK_PROJECT_ROOT}/.vrooli/secrets.json" <<EOF
{"test": "value"}
EOF
    chmod 644 "${MOCK_PROJECT_ROOT}/.vrooli/secrets.json"
    
    run secrets::validate_storage
    assert_failure
    assert_output --partial "incorrect permissions"
}

@test "secrets::validate_all_configs checks configuration locations" {
    run secrets::validate_all_configs
    assert_success
}