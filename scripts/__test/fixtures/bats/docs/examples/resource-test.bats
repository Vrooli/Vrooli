#!/usr/bin/env bats
# Example: Single resource testing with Ollama
# Use this pattern for testing integration with specific Vrooli resources

bats_require_minimum_version 1.5.0

# Load the unified testing infrastructure
source "/home/matthalloran8/Vrooli/scripts/__test/fixtures/bats/core/common_setup.bash"

setup() {
    # Set up environment for Ollama resource testing
    # This automatically configures ports, URLs, container names, and loads appropriate mocks
    setup_resource_test "ollama"
}

teardown() {
    # Clean up test environment
    cleanup_mocks
}

@test "ollama environment is properly configured" {
    # Test that resource-specific environment variables are set
    assert_env_set "OLLAMA_PORT"
    assert_env_set "OLLAMA_BASE_URL"
    assert_env_set "OLLAMA_CONTAINER_NAME"
    
    # Test expected values
    assert_env_equals "OLLAMA_PORT" "11434"
    assert_env_equals "OLLAMA_BASE_URL" "http://localhost:11434"
    
    # Test test isolation variables
    assert_env_set "TEST_NAMESPACE"
    assert_env_set "RESOURCE_NAME"
    assert_env_equals "RESOURCE_NAME" "ollama"
}

@test "ollama service health check" {
    # Use resource-specific health assertion
    assert_resource_healthy "ollama"
    
    # Test manual health check
    run curl -s "$OLLAMA_BASE_URL/health"
    assert_success
    assert_json_valid "$output"
}

@test "ollama container is running" {
    # Test Docker container state
    assert_docker_container_running "$OLLAMA_CONTAINER_NAME"
    
    # Test container inspect
    run docker inspect "$OLLAMA_CONTAINER_NAME" --format '{{.State.Status}}'
    assert_success
    assert_output_equals "running"
}

@test "ollama API endpoints respond correctly" {
    # Test models API endpoint
    run curl -s "$OLLAMA_BASE_URL/api/tags"
    assert_success
    assert_json_valid "$output"
    assert_json_field_exists "$output" ".models"
    
    # Test that models are available
    local models_response=$(curl -s "$OLLAMA_BASE_URL/api/tags")
    local model_count=$(echo "$models_response" | jq '.models | length')
    assert_greater_than "$model_count" "0"
}

@test "ollama model generation works" {
    # Test text generation endpoint
    local generation_request='{
        "model": "llama3.1:8b",
        "prompt": "Hello, how are you?",
        "stream": false
    }'
    
    run curl -X POST "$OLLAMA_BASE_URL/api/generate" \
        -H "Content-Type: application/json" \
        -d "$generation_request"
    
    assert_success
    assert_json_valid "$output"
    assert_json_field_exists "$output" ".response"
    assert_json_field_equals "$output" ".done" "true"
}

@test "ollama container logs are accessible" {
    # Test Docker logs access
    run docker logs "$OLLAMA_CONTAINER_NAME"
    assert_success
    assert_output_contains "Container $OLLAMA_CONTAINER_NAME started"
}

@test "ollama port is accessible" {
    # Test port connectivity
    run nc -z localhost "$OLLAMA_PORT"
    assert_success
    
    # Test with custom port check
    assert_port_open "localhost" "$OLLAMA_PORT"
}

@test "ollama configuration is valid" {
    # Test that all required configuration is present
    local required_vars=("OLLAMA_PORT" "OLLAMA_BASE_URL" "OLLAMA_CONTAINER_NAME")
    
    for var in "${required_vars[@]}"; do
        assert_env_set "$var"
        assert_env_not_empty "$var"
    done
}

@test "ollama isolation works correctly" {
    # Test that namespace isolation is working
    assert_string_contains "$OLLAMA_CONTAINER_NAME" "$TEST_NAMESPACE"
    
    # Test that temporary directories are namespaced
    assert_string_contains "$TEST_TMPDIR" "$$"  # Process ID in path
}

@test "ollama mock responses are realistic" {
    # Test that mock responses match real Ollama API format
    local tags_response=$(curl -s "$OLLAMA_BASE_URL/api/tags")
    
    # Verify response structure matches Ollama API
    assert_json_field_exists "$tags_response" ".models"
    assert_json_field_exists "$tags_response" ".models[0].name"
    assert_json_field_exists "$tags_response" ".models[0].size"
    
    # Verify realistic model names
    local first_model_name=$(echo "$tags_response" | jq -r '.models[0].name')
    assert_string_contains "$first_model_name" ":"  # Models have format like "llama3.1:8b"
}