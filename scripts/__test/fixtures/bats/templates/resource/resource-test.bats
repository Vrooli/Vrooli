#!/usr/bin/env bats
# Template: Resource-Specific BATS Test
# Use this template for testing integration with a specific Vrooli resource
#
# Copy this file and customize:
# 1. Change RESOURCE_NAME to your target resource
# 2. Update the test description
# 3. Add your specific test cases
# 4. Customize the setup/teardown if needed

bats_require_minimum_version 1.5.0

# ================================
# CONFIGURATION - CHANGE THIS
# ================================
# Set your target resource here (e.g., "ollama", "whisper", "n8n", "qdrant")
RESOURCE_NAME="ollama"  # CHANGE THIS

# Load the unified testing infrastructure  
if [[ -n "${VROOLI_TEST_FIXTURES_DIR:-}" ]]; then
    source "${VROOLI_TEST_FIXTURES_DIR}/core/common_setup.bash"
else
    # Adjust the path based on where you place this test file
    TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    source "${TEST_DIR}/../core/common_setup.bash"
fi

setup() {
    # Load resource-specific test environment
    # This automatically configures mocks and environment for your resource
    setup_resource_test "$RESOURCE_NAME"
    
    # Add any custom setup here
    # export MY_TEST_VAR="value"
}

teardown() {
    # Always clean up test environment
    cleanup_mocks
    
    # Add any custom cleanup here
}

# ================================
# EXAMPLE TESTS - Customize these
# ================================

@test "resource environment is configured" {
    # Test that resource-specific environment is set up
    # These variable names depend on your resource
    case "$RESOURCE_NAME" in
        "ollama")
            assert_env_set "OLLAMA_PORT"
            assert_env_set "OLLAMA_BASE_URL" 
            assert_env_equals "OLLAMA_PORT" "11434"
            ;;
        "whisper")
            assert_env_set "WHISPER_PORT"
            assert_env_set "WHISPER_BASE_URL"
            assert_env_equals "WHISPER_PORT" "8090"
            ;;
        "n8n")
            assert_env_set "N8N_PORT"
            assert_env_set "N8N_BASE_URL"
            assert_env_equals "N8N_PORT" "5678"
            ;;
        "qdrant")
            assert_env_set "QDRANT_PORT"
            assert_env_set "QDRANT_BASE_URL"
            assert_env_equals "QDRANT_PORT" "6333"
            ;;
        *)
            # Generic resource variables
            assert_env_set "RESOURCE_PORT"
            assert_env_set "RESOURCE_BASE_URL"
            ;;
    esac
    
    # Universal resource variables
    assert_env_set "RESOURCE_NAME"
    assert_env_set "TEST_NAMESPACE"
    assert_env_equals "RESOURCE_NAME" "$RESOURCE_NAME"
}

@test "resource service is healthy" {
    # Test that the resource appears healthy in the mock environment
    assert_resource_healthy "$RESOURCE_NAME"
}

@test "resource container is running" {
    # Test that the resource container is in running state
    local container_name
    case "$RESOURCE_NAME" in
        "ollama") container_name="$OLLAMA_CONTAINER_NAME" ;;
        "whisper") container_name="$WHISPER_CONTAINER_NAME" ;;
        "n8n") container_name="$N8N_CONTAINER_NAME" ;;
        "qdrant") container_name="$QDRANT_CONTAINER_NAME" ;;
        *) container_name="$RESOURCE_CONTAINER_NAME" ;;
    esac
    
    assert_docker_container_running "$container_name"
}

@test "resource API responds correctly" {
    # Test that the resource API returns valid responses
    local base_url
    case "$RESOURCE_NAME" in
        "ollama") base_url="$OLLAMA_BASE_URL" ;;
        "whisper") base_url="$WHISPER_BASE_URL" ;;
        "n8n") base_url="$N8N_BASE_URL" ;;
        "qdrant") base_url="$QDRANT_BASE_URL" ;;
        *) base_url="$RESOURCE_BASE_URL" ;;
    esac
    
    # Test health endpoint
    run curl -s "$base_url/health"
    assert_success
    assert_json_valid "$output"
}

@test "resource-specific API endpoints work" {
    # Add resource-specific API tests here
    case "$RESOURCE_NAME" in
        "ollama")
            # Test Ollama-specific endpoints
            run curl -s "$OLLAMA_BASE_URL/api/tags"
            assert_success
            assert_json_valid "$output"
            assert_json_field_exists "$output" ".models"
            ;;
        "whisper")
            # Test Whisper-specific endpoints
            run curl -s "$WHISPER_BASE_URL/transcribe" -X POST -d '{"audio": "test"}'
            assert_success
            assert_json_valid "$output"
            ;;
        "n8n")
            # Test N8N-specific endpoints
            run curl -s "$N8N_BASE_URL/api/v1/workflows"
            assert_success
            assert_json_valid "$output"
            ;;
        "qdrant")
            # Test Qdrant-specific endpoints
            run curl -s "$QDRANT_BASE_URL/collections"
            assert_success
            assert_json_valid "$output"
            ;;
        *)
            # Generic API test
            run curl -s "$RESOURCE_BASE_URL/api"
            assert_success
            ;;
    esac
}

# ================================
# ADD YOUR CUSTOM TESTS HERE
# ================================

# @test "my resource integration test" {
#     # Your resource-specific test logic here
#     run your_resource_command_here
#     assert_success
#     assert_output_contains "expected output"
# }

# @test "resource performance test" {
#     # Test resource performance characteristics
#     local start_time end_time duration
#     start_time=$(date +%s%N)
#     
#     # Your performance test here
#     run curl -s "$RESOURCE_BASE_URL/heavy_operation"
#     
#     end_time=$(date +%s%N)
#     duration=$(( (end_time - start_time) / 1000000 ))  # Convert to milliseconds
#     
#     assert_success
#     assert_less_than "$duration" "5000"  # Should complete in under 5 seconds
# }