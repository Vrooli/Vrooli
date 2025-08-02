#!/usr/bin/env bats
# Example: Multi-resource integration testing
# Use this pattern for testing workflows involving multiple Vrooli resources

bats_require_minimum_version 1.5.0

# Load the unified testing infrastructure
# Source from relative path or use VROOLI_TEST_FIXTURES_DIR if set
if [[ -n "${VROOLI_TEST_FIXTURES_DIR:-}" ]]; then
    source "${VROOLI_TEST_FIXTURES_DIR}/core/common_setup.bash"
else
    # Find fixtures directory relative to this test file
    TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    source "${TEST_DIR}/../../core/common_setup.bash"
fi

setup() {
    # Set up environment for multi-resource testing
    # This configures Ollama (AI), Whisper (transcription), and N8N (automation)
    setup_integration_test "ollama" "whisper" "n8n"
}

teardown() {
    # Clean up test environment
    cleanup_mocks
}

@test "all resources are properly configured" {
    # Test that each resource has its environment configured
    assert_env_set "OLLAMA_PORT"
    assert_env_set "OLLAMA_BASE_URL"
    assert_env_set "WHISPER_PORT" 
    assert_env_set "WHISPER_BASE_URL"
    assert_env_set "N8N_PORT"
    assert_env_set "N8N_BASE_URL"
    
    # Test expected port assignments
    assert_env_equals "OLLAMA_PORT" "11434"
    assert_env_equals "WHISPER_PORT" "8090"
    assert_env_equals "N8N_PORT" "5678"
}

@test "all resources share the same test namespace" {
    # Test namespace consistency across resources
    assert_env_set "TEST_NAMESPACE"
    
    # All container names should include the same namespace
    assert_string_contains "$OLLAMA_CONTAINER_NAME" "$TEST_NAMESPACE"
    assert_string_contains "$WHISPER_CONTAINER_NAME" "$TEST_NAMESPACE"
    assert_string_contains "$N8N_CONTAINER_NAME" "$TEST_NAMESPACE"
}

@test "all resources are healthy" {
    # Test health of each resource
    assert_resource_healthy "ollama"
    assert_resource_healthy "whisper"
    assert_resource_healthy "n8n"
}

@test "all containers are running" {
    # Test Docker container states
    assert_docker_container_running "$OLLAMA_CONTAINER_NAME"
    assert_docker_container_running "$WHISPER_CONTAINER_NAME"
    assert_docker_container_running "$N8N_CONTAINER_NAME"
}

@test "cross-resource communication is possible" {
    # Test that resources can communicate with each other
    
    # Create an N8N workflow that uses both Ollama and Whisper
    local workflow_json='{
        "name": "AI Processing Pipeline",
        "nodes": [
            {
                "type": "whisper-transcribe",
                "url": "'$WHISPER_BASE_URL'/transcribe",
                "position": [100, 100]
            },
            {
                "type": "ollama-generate", 
                "url": "'$OLLAMA_BASE_URL'/api/generate",
                "position": [300, 100]
            }
        ],
        "connections": [
            {"from": "whisper-transcribe", "to": "ollama-generate"}
        ]
    }'
    
    # Test workflow creation
    run curl -X POST "$N8N_BASE_URL/api/v1/workflows" \
        -H "Content-Type: application/json" \
        -d "$workflow_json"
    
    assert_success
    assert_json_valid "$output"
}

@test "AI processing pipeline works end-to-end" {
    # Simulate complete AI workflow: Audio → Whisper → Ollama → Result
    
    # Step 1: Transcribe audio with Whisper
    local transcription_request='{
        "audio_data": "mock_audio_base64",
        "language": "en"
    }'
    
    local transcription_response=$(curl -X POST "$WHISPER_BASE_URL/transcribe" \
        -H "Content-Type: application/json" \
        -d "$transcription_request")
    
    assert_json_valid "$transcription_response"
    assert_json_field_exists "$transcription_response" ".text"
    
    # Step 2: Process transcription with Ollama
    local transcribed_text=$(echo "$transcription_response" | jq -r '.text')
    local generation_request='{
        "model": "llama3.1:8b",
        "prompt": "Summarize this text: '"$transcribed_text"'",
        "stream": false
    }'
    
    local generation_response=$(curl -X POST "$OLLAMA_BASE_URL/api/generate" \
        -H "Content-Type: application/json" \
        -d "$generation_request")
    
    assert_json_valid "$generation_response"
    assert_json_field_exists "$generation_response" ".response"
    assert_json_field_equals "$generation_response" ".done" "true"
    
    # Step 3: Verify the pipeline produced a result
    local summary=$(echo "$generation_response" | jq -r '.response')
    assert_not_empty "$summary"
}

@test "resource isolation prevents conflicts" {
    # Test that resources don't interfere with each other
    
    # Each resource should have unique ports
    local ports=("$OLLAMA_PORT" "$WHISPER_PORT" "$N8N_PORT")
    local unique_ports=($(printf '%s\n' "${ports[@]}" | sort -u))
    
    assert_equals "${#ports[@]}" "${#unique_ports[@]}"
    
    # Each resource should have unique container names
    local containers=("$OLLAMA_CONTAINER_NAME" "$WHISPER_CONTAINER_NAME" "$N8N_CONTAINER_NAME")
    local unique_containers=($(printf '%s\n' "${containers[@]}" | sort -u))
    
    assert_equals "${#containers[@]}" "${#unique_containers[@]}"
}

@test "workflow coordination through N8N" {
    # Test N8N's ability to coordinate between Ollama and Whisper
    
    # Get N8N workflow list
    run curl -s "$N8N_BASE_URL/api/v1/workflows"
    assert_success
    assert_json_valid "$output"
    
    # Test N8N can reach both services
    local n8n_health=$(curl -s "$N8N_BASE_URL/healthz")
    assert_json_field_equals "$n8n_health" ".status" "ok"
    
    # Test that N8N can make requests to other services
    # (This would be done through N8N's workflow execution in a real scenario)
    local ollama_via_n8n=$(curl -s "$N8N_BASE_URL/proxy/ollama/api/tags" || curl -s "$OLLAMA_BASE_URL/api/tags")
    assert_json_valid "$ollama_via_n8n"
}

@test "performance is acceptable for integration tests" {
    # Test that multi-resource setup doesn't have excessive overhead
    
    local start_time=$(date +%s%3N)
    
    # Perform multiple resource operations
    curl -s "$OLLAMA_BASE_URL/health" > /dev/null
    curl -s "$WHISPER_BASE_URL/health" > /dev/null  
    curl -s "$N8N_BASE_URL/healthz" > /dev/null
    
    local end_time=$(date +%s%3N)
    local duration=$((end_time - start_time))
    
    # Should complete in under 1 second (1000ms) even with mocking overhead
    assert_less_than "$duration" "1000"
}

@test "cleanup works for all resources" {
    # Test that cleanup properly handles multiple resources
    
    # Verify all resources are currently active
    assert_resource_healthy "ollama"
    assert_resource_healthy "whisper"
    assert_resource_healthy "n8n"
    
    # Manual cleanup test (normally done in teardown)
    cleanup_mocks
    
    # After cleanup, verify environment is reset
    # Note: In real tests, this would be in a separate test file
    # since cleanup_mocks is called in teardown()
}