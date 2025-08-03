#!/usr/bin/env bats
# Ollama Service Test
# Migrated from old docs/examples/resource-test.bats

bats_require_minimum_version 1.5.0

# Load new unified test infrastructure
load "../../setup"

setup() {
    # Use new service-specific setup
    vrooli_setup_service_test "ollama"
}

teardown() {
    vrooli_cleanup_test
}

@test "ollama environment is properly configured" {
    # Test that service-specific environment variables are set
    assert_env_set "VROOLI_OLLAMA_PORT"
    assert_env_set "VROOLI_OLLAMA_BASE_URL"
    assert_env_set "VROOLI_OLLAMA_CONTAINER_NAME"
    
    # Test expected values
    assert_env_equals "VROOLI_OLLAMA_PORT" "11434"
    assert_env_equals "VROOLI_OLLAMA_BASE_URL" "http://localhost:11434"
    
    # Test namespace isolation
    assert_env_set "TEST_NAMESPACE"
}

@test "ollama service health check" {
    # Test health endpoint (mocked)
    run curl -s "$VROOLI_OLLAMA_BASE_URL/health"
    assert_success
    assert_output_contains "healthy"
}

@test "ollama API version check" {
    # Test version endpoint
    run curl -s "$VROOLI_OLLAMA_BASE_URL/api/version"
    assert_success
    assert_json_valid "$output"
}

@test "ollama model operations" {
    # Test listing models (mocked)
    run ollama list
    assert_success
    assert_output_contains "models"
}

@test "ollama generate endpoint" {
    # Test generation API (mocked)
    local prompt='{"model":"llama2","prompt":"Hello"}'
    run curl -X POST "$VROOLI_OLLAMA_BASE_URL/api/generate" -d "$prompt"
    assert_success
    assert_output_contains "response"
}

@test "ollama container status" {
    # Test Docker container (mocked)
    run docker ps --filter "name=$VROOLI_OLLAMA_CONTAINER_NAME"
    assert_success
    assert_output_contains "CONTAINER"
}

# Migrated: Service-specific test for Ollama