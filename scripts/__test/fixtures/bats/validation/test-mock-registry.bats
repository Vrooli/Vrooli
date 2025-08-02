#!/usr/bin/env bats
# Validation Test Suite for Mock Registry System
# This test validates that the mock registry and mock loading works correctly

bats_require_minimum_version 1.5.0

# Load BATS libraries
load "${BATS_TEST_DIRNAME}/../../../helpers/bats-support/load"
load "${BATS_TEST_DIRNAME}/../../../helpers/bats-assert/load"

# Load the infrastructure we're testing
source "${BATS_TEST_DIRNAME}/../core/common_setup.bash"

setup() {
    # Clean environment for each test
    unset MOCK_REGISTRY_LOADED
    # Re-source to test loading fresh
    source "${BATS_TEST_DIRNAME}/../mocks/mock_registry.bash"
}

teardown() {
    cleanup_mocks 2>/dev/null || true
}

#######################################
# Mock Registry Core Function Tests
#######################################

@test "mock registry loads without errors" {
    run bash -c "source '${BATS_TEST_DIRNAME}/../mocks/mock_registry.bash' && echo 'loaded'"
    assert_success
    assert_output_contains "Mock registry system loaded"
    assert_output_contains "loaded"
}

@test "mock::load function exists and works" {
    assert_function_exists "mock::load"
    
    run mock::load system docker
    assert_success
    assert_output_contains "Loaded system:docker"
}

@test "mock::detect_resource_category function works correctly" {
    assert_function_exists "mock::detect_resource_category"
    
    # Test AI resources
    run mock::detect_resource_category "ollama"
    assert_success
    assert_output_equals "ai"
    
    run mock::detect_resource_category "whisper"
    assert_success
    assert_output_equals "ai"
    
    # Test automation resources
    run mock::detect_resource_category "n8n"
    assert_success
    assert_output_equals "automation"
    
    run mock::detect_resource_category "node-red"
    assert_success
    assert_output_equals "automation"
    
    # Test storage resources
    run mock::detect_resource_category "redis"
    assert_success
    assert_output_equals "storage"
    
    run mock::detect_resource_category "postgres"
    assert_success
    assert_output_equals "storage"
    
    # Test agents
    run mock::detect_resource_category "agent-s2"
    assert_success
    assert_output_equals "agents"
    
    # Test search
    run mock::detect_resource_category "searxng"
    assert_success
    assert_output_equals "search"
    
    # Test execution
    run mock::detect_resource_category "judge0"
    assert_success
    assert_output_equals "execution"
    
    # Test unknown resource
    run mock::detect_resource_category "unknown_resource"
    assert_success
    assert_output_equals "unknown"
}

@test "mock::setup_minimal function works" {
    assert_function_exists "mock::setup_minimal"
    
    run mock::setup_minimal
    assert_success
    assert_output_contains "Setting up minimal test environment"
    assert_output_contains "Loaded system:docker"
    assert_output_contains "Loaded system:http"
    assert_output_contains "Loaded system:commands"
}

@test "mock::setup_resource function works" {
    assert_function_exists "mock::setup_resource"
    
    run mock::setup_resource "ollama"
    assert_success
    assert_output_contains "Setting up test environment for resource: ollama"
    assert_output_contains "Loaded resource:ollama"
}

@test "mock::setup_integration function works" {
    assert_function_exists "mock::setup_integration"
    
    run mock::setup_integration "ollama" "n8n"
    assert_success
    assert_output_contains "Setting up integration test environment for: ollama n8n"
    assert_output_contains "Loaded resource:ollama"
    assert_output_contains "Loaded resource:n8n"
}

@test "mock::configure_resource_environment function works" {
    assert_function_exists "mock::configure_resource_environment"
    
    # Test Ollama configuration
    mock::configure_resource_environment "ollama"
    assert_env_equals "RESOURCE_NAME" "ollama"
    assert_env_equals "OLLAMA_PORT" "11434"
    assert_env_equals "OLLAMA_BASE_URL" "http://localhost:11434"
    assert_env_set "OLLAMA_CONTAINER_NAME"
    
    # Test N8N configuration
    mock::configure_resource_environment "n8n"
    assert_env_equals "N8N_PORT" "5678"
    assert_env_equals "N8N_BASE_URL" "http://localhost:5678"
    assert_env_set "N8N_CONTAINER_NAME"
    
    # Test generic resource configuration
    mock::configure_resource_environment "generic_resource"
    assert_env_set "RESOURCE_PORT"
    assert_env_set "RESOURCE_BASE_URL"
    assert_env_set "RESOURCE_CONTAINER_NAME"
}

@test "mock::cleanup function works" {
    assert_function_exists "mock::cleanup"
    
    # Setup some environment first
    mock::setup_minimal
    assert_env_set "TEST_TMPDIR"
    
    # Now cleanup
    run mock::cleanup
    assert_success
    assert_output_contains "Cleaning up test environment"
}

@test "mock::list_loaded function works" {
    assert_function_exists "mock::list_loaded"
    
    # Load some mocks first
    mock::load system docker
    mock::load system http
    
    run mock::list_loaded
    assert_success
    assert_output_contains "Currently loaded mocks:"
    assert_output_contains "system_docker"
    assert_output_contains "system_http"
}

@test "mock::is_loaded function works" {
    assert_function_exists "mock::is_loaded"
    
    # Load a mock
    mock::load system docker
    
    # Check if it's loaded
    run mock::is_loaded system docker
    assert_success
    
    # Check unloaded mock
    run mock::is_loaded system nonexistent
    assert_failure
}

#######################################
# System Mock Loading Tests
#######################################

@test "all system mocks can be loaded" {
    local system_mocks=("docker" "http" "commands" "filesystem")
    
    for mock in "${system_mocks[@]}"; do
        run mock::load system "$mock"
        assert_success
        assert_output_contains "Loaded system:$mock"
    done
}

@test "system mock files exist" {
    local system_mocks=("docker" "http" "commands" "filesystem")
    
    for mock in "${system_mocks[@]}"; do
        local mock_file="${VROOLI_TEST_FIXTURES_DIR}/mocks/system/${mock}.bash"
        assert_file_exists "$mock_file"
    done
}

#######################################
# Resource Mock Loading Tests
#######################################

@test "AI resource mocks can be loaded" {
    local ai_resources=("ollama" "whisper" "comfyui" "unstructured-io")
    
    for resource in "${ai_resources[@]}"; do
        run mock::load resource "$resource"
        assert_success
        assert_output_contains "Loaded resource:$resource"
    done
}

@test "automation resource mocks can be loaded" {
    local automation_resources=("n8n" "node-red" "huginn" "windmill")
    
    for resource in "${automation_resources[@]}"; do
        run mock::load resource "$resource"
        assert_success
        assert_output_contains "Loaded resource:$resource"
    done
}

@test "storage resource mocks can be loaded" {
    local storage_resources=("redis" "postgres" "qdrant" "minio" "vault")
    
    for resource in "${storage_resources[@]}"; do
        run mock::load resource "$resource"
        assert_success
        assert_output_contains "Loaded resource:$resource"
    done
}

@test "agent resource mocks can be loaded" {
    local agent_resources=("agent-s2" "browserless" "claude-code")
    
    for resource in "${agent_resources[@]}"; do
        run mock::load resource "$resource"
        assert_success
        assert_output_contains "Loaded resource:$resource"
    done
}

@test "other resource mocks can be loaded" {
    local other_resources=("searxng" "judge0")
    
    for resource in "${other_resources[@]}"; do
        run mock::load resource "$resource"
        assert_success
        assert_output_contains "Loaded resource:$resource"
    done
}

#######################################
# Resource Mock File Existence Tests
#######################################

@test "all AI resource mock files exist" {
    local ai_resources=("ollama" "whisper" "comfyui" "unstructured-io")
    
    for resource in "${ai_resources[@]}"; do
        local mock_file="${VROOLI_TEST_FIXTURES_DIR}/mocks/resources/ai/${resource}.bash"
        assert_file_exists "$mock_file"
    done
}

@test "all automation resource mock files exist" {
    local automation_resources=("n8n" "node-red" "huginn" "windmill")
    
    for resource in "${automation_resources[@]}"; do
        local mock_file="${VROOLI_TEST_FIXTURES_DIR}/mocks/resources/automation/${resource}.bash"
        assert_file_exists "$mock_file"
    done
}

@test "all storage resource mock files exist" {
    local storage_resources=("redis" "postgres" "qdrant" "minio" "vault")
    
    for resource in "${storage_resources[@]}"; do
        local mock_file="${VROOLI_TEST_FIXTURES_DIR}/mocks/resources/storage/${resource}.bash"
        assert_file_exists "$mock_file"
    done
}

#######################################
# Error Handling Tests
#######################################

@test "mock::load handles invalid category gracefully" {
    run mock::load invalid_category test_resource
    assert_failure
    assert_output_contains "ERROR: Unknown mock category: invalid_category"
}

@test "mock::load handles nonexistent mock files gracefully" {
    run mock::load system nonexistent_mock
    # Should try legacy fallback
    assert_output_contains "WARNING: Mock not found"
}

@test "mock::setup_resource handles invalid resource gracefully" {
    run mock::setup_resource "nonexistent_resource"
    # Should not crash catastrophically
    assert_success
}

#######################################
# Performance and State Tests
#######################################

@test "duplicate mock loading is prevented" {
    # Load same mock twice
    mock::load system docker
    local first_load_output="$output"
    
    run mock::load system docker
    assert_success
    # Second load should be skipped (no output about loading)
    assert_output_equals ""
}

@test "environment variables are preserved across mock loads" {
    export TEST_PRESERVATION="preserved_value"
    
    mock::load system docker
    mock::load system http
    
    assert_env_equals "TEST_PRESERVATION" "preserved_value"
}

@test "temporary directory creation works correctly" {
    mock::setup_minimal
    
    assert_env_set "TEST_TMPDIR"
    assert_dir_exists "$TEST_TMPDIR"
}

#######################################
# Integration with Common Setup Tests
#######################################

@test "mock registry integrates correctly with common setup" {
    run bash -c "
        source '${BATS_TEST_DIRNAME}/../core/common_setup.bash'
        setup_standard_mocks
        mock::list_loaded
    "
    assert_success
    assert_output_contains "Currently loaded mocks:"
    assert_output_contains "system_docker"
    assert_output_contains "system_http"
    assert_output_contains "system_commands"
}

@test "resource environment variables are set correctly in integration" {
    run bash -c "
        source '${BATS_TEST_DIRNAME}/../core/common_setup.bash'
        setup_resource_test ollama
        echo \"OLLAMA_PORT=\${OLLAMA_PORT}\"
        echo \"OLLAMA_BASE_URL=\${OLLAMA_BASE_URL}\"
        echo \"OLLAMA_CONTAINER_NAME=\${OLLAMA_CONTAINER_NAME}\"
    "
    assert_success
    assert_output_contains "OLLAMA_PORT=11434"
    assert_output_contains "OLLAMA_BASE_URL=http://localhost:11434"
    assert_output_contains "OLLAMA_CONTAINER_NAME="
}