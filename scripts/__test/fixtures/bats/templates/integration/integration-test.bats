#!/usr/bin/env bats
# Template: Integration BATS Test
# Use this template for testing workflows involving multiple resources
#
# Copy this file and customize:
# 1. Change RESOURCES array to your target resources
# 2. Update the test description
# 3. Add your workflow test cases
# 4. Customize the setup/teardown if needed

bats_require_minimum_version 1.5.0

# ================================
# CONFIGURATION - CHANGE THIS
# ================================
# Set your target resources here
# Examples: "ollama whisper n8n", "qdrant minio postgres", "searxng browserless agent-s2"
RESOURCES=("ollama" "whisper" "n8n")  # CHANGE THIS

# Load the unified testing infrastructure  
if [[ -n "${VROOLI_TEST_FIXTURES_DIR:-}" ]]; then
    source "${VROOLI_TEST_FIXTURES_DIR}/core/common_setup.bash"
else
    # Adjust the path based on where you place this test file
    TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    source "${TEST_DIR}/../core/common_setup.bash"
fi

setup() {
    # Load integration test environment for multiple resources
    # This automatically configures mocks and environment for all your resources
    setup_integration_test "${RESOURCES[@]}"
    
    # Add any custom setup here
    # export MY_WORKFLOW_CONFIG="value"
}

teardown() {
    # Always clean up test environment
    cleanup_mocks
    
    # Add any custom cleanup here
}

# ================================
# EXAMPLE TESTS - Customize these
# ================================

@test "all resources are configured" {
    # Test that all resource environments are set up
    for resource in "${RESOURCES[@]}"; do
        case "$resource" in
            "ollama")
                assert_env_set "OLLAMA_PORT"
                assert_env_set "OLLAMA_BASE_URL"
                ;;
            "whisper")
                assert_env_set "WHISPER_PORT" 
                assert_env_set "WHISPER_BASE_URL"
                ;;
            "n8n")
                assert_env_set "N8N_PORT"
                assert_env_set "N8N_BASE_URL"
                ;;
            "qdrant")
                assert_env_set "QDRANT_PORT"
                assert_env_set "QDRANT_BASE_URL"
                ;;
            *)
                # For other resources, check generic variables
                assert_env_set "RESOURCE_NAME"
                ;;
        esac
    done
    
    # All resources should share the same test namespace
    assert_env_set "TEST_NAMESPACE"
}

@test "all resources are healthy" {
    # Test that all resources appear healthy in the mock environment
    for resource in "${RESOURCES[@]}"; do
        assert_resource_healthy "$resource"
    done
}

@test "containers are isolated by namespace" {
    # All containers should use the same test namespace for isolation
    for resource in "${RESOURCES[@]}"; do
        local container_name="${TEST_NAMESPACE}_${resource}"
        assert_docker_container_running "$container_name"
    done
}

@test "cross-resource communication works" {
    # Test that resources can communicate with each other
    # This is a mock test - in real scenarios this would test actual workflows
    
    # Example: AI pipeline workflow (Whisper -> Ollama -> N8N)
    if [[ " ${RESOURCES[*]} " =~ " ollama " ]] && [[ " ${RESOURCES[*]} " =~ " whisper " ]] && [[ " ${RESOURCES[*]} " =~ " n8n " ]]; then
        local workflow_json
        workflow_json='{
            "name": "AI Processing Pipeline",
            "nodes": [
                {
                    "type": "whisper",
                    "name": "Audio Transcription",
                    "url": "'$WHISPER_BASE_URL'",
                    "endpoint": "/transcribe"
                },
                {
                    "type": "ollama", 
                    "name": "Text Processing",
                    "url": "'$OLLAMA_BASE_URL'",
                    "endpoint": "/api/generate"
                },
                {
                    "type": "n8n",
                    "name": "Workflow Automation",
                    "url": "'$N8N_BASE_URL'", 
                    "endpoint": "/api/v1/workflows"
                }
            ]
        }'
        
        assert_json_valid "$workflow_json"
        
        # Test that N8N can accept this workflow configuration
        run curl -X POST "$N8N_BASE_URL/api/v1/workflows" \
            -H "Content-Type: application/json" \
            -d "$workflow_json"
        assert_success
    fi
    
    # Example: Data pipeline workflow (Database -> Vector DB -> Search)
    if [[ " ${RESOURCES[*]} " =~ " postgres " ]] && [[ " ${RESOURCES[*]} " =~ " qdrant " ]] && [[ " ${RESOURCES[*]} " =~ " searxng " ]]; then
        # Test data flow: Postgres -> Qdrant -> SearXNG
        local data_pipeline='{
            "pipeline": "data_processing",
            "steps": [
                {"source": "postgres", "url": "'$POSTGRES_BASE_URL'"},
                {"processor": "qdrant", "url": "'$QDRANT_BASE_URL'"},
                {"search": "searxng", "url": "'$SEARXNG_BASE_URL'"}
            ]
        }'
        
        assert_json_valid "$data_pipeline"
    fi
}

@test "workflow coordination works" {
    # Test a complete workflow that involves all resources
    # This is where you would test your actual business logic
    
    # Example workflow test
    local workflow_id="test_workflow_$$"
    local workflow_result
    
    # Step 1: Initialize workflow
    # (In a real test, this would call your actual workflow management system)
    echo "Starting workflow: $workflow_id"
    
    # Step 2: Process through each resource
    for resource in "${RESOURCES[@]}"; do
        case "$resource" in
            "ollama")
                # Simulate LLM processing
                workflow_result=$(curl -s "$OLLAMA_BASE_URL/api/generate" \
                    -d '{"prompt": "Process this data", "model": "test"}')
                assert_json_valid "$workflow_result"
                ;;
            "whisper")
                # Simulate audio transcription
                workflow_result=$(curl -s "$WHISPER_BASE_URL/transcribe" \
                    -d '{"audio": "test_audio_data"}')
                assert_json_valid "$workflow_result"
                ;;
            "n8n")
                # Simulate workflow execution
                workflow_result=$(curl -s "$N8N_BASE_URL/api/v1/workflows/execute" \
                    -d '{"workflowId": "'$workflow_id'"}')
                assert_json_valid "$workflow_result"
                ;;
        esac
    done
    
    # Step 3: Verify workflow completion
    assert_not_empty "$workflow_result"
    echo "Workflow completed: $workflow_id"
}

@test "error handling works across resources" {
    # Test that errors in one resource don't break the entire workflow
    
    # Simulate an error condition in the first resource
    local first_resource="${RESOURCES[0]}"
    
    # Mock an error response for the first resource
    case "$first_resource" in
        "ollama")
            mock::http::set_endpoint_response "$OLLAMA_BASE_URL/api/generate" \
                '{"error": "Service temporarily unavailable"}' 503
            ;;
        "whisper")
            mock::http::set_endpoint_response "$WHISPER_BASE_URL/transcribe" \
                '{"error": "Processing failed"}' 503
            ;;
        *)
            # Generic error setup
            local base_url="${first_resource^^}_BASE_URL"  # Convert to uppercase
            mock::http::set_endpoint_response "${!base_url}/api" \
                '{"error": "Service error"}' 503
            ;;
    esac
    
    # Test that other resources still work
    for resource in "${RESOURCES[@]:1}"; do  # Skip the first resource
        assert_resource_healthy "$resource"
    done
}

# ================================
# ADD YOUR CUSTOM TESTS HERE
# ================================

# @test "my custom workflow test" {
#     # Your integration workflow test logic here
#     
#     # Step 1: Prepare data
#     local test_data='{"input": "test"}'
#     assert_json_valid "$test_data"
#     
#     # Step 2: Process through workflow
#     # ... your workflow steps ...
#     
#     # Step 3: Verify results
#     # assert_output_contains "expected result"
# }

# @test "performance integration test" {
#     # Test the performance of the entire workflow
#     local start_time end_time duration
#     start_time=$(date +%s%N)
#     
#     # Run your complete workflow here
#     for resource in "${RESOURCES[@]}"; do
#         curl -s "${resource^^}_BASE_URL/health" >/dev/null
#     done
#     
#     end_time=$(date +%s%N)
#     duration=$(( (end_time - start_time) / 1000000 ))  # Convert to milliseconds
#     
#     # Workflow should complete within reasonable time
#     assert_less_than "$duration" "10000"  # Should complete in under 10 seconds
# }