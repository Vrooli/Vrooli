#!/usr/bin/env bats
# Template: Advanced BATS Test
# Use this template for complex testing scenarios with advanced features
#
# This template demonstrates:
# - Custom mock responses and sequences
# - Error injection and recovery testing  
# - Complex assertion patterns
# - Test data management
# - Advanced setup/teardown patterns

bats_require_minimum_version 1.5.0

# ================================
# CONFIGURATION
# ================================
# Set your configuration here
TEST_SCENARIO="advanced_workflow"  # CHANGE THIS
RESOURCES=("ollama" "n8n")          # CHANGE THIS
ENABLE_ERROR_INJECTION=true        # Set to false to disable error testing

# Load the unified testing infrastructure  
if [[ -n "${VROOLI_TEST_FIXTURES_DIR:-}" ]]; then
    source "${VROOLI_TEST_FIXTURES_DIR}/core/common_setup.bash"
else
    # Adjust the path based on where you place this test file
    TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    source "${TEST_DIR}/../core/common_setup.bash"
fi

# ================================
# ADVANCED SETUP & HELPERS
# ================================

setup() {
    # Setup integration environment
    setup_integration_test "${RESOURCES[@]}"
    
    # Create test data directory
    TEST_DATA_DIR="$TEST_TMPDIR/test_data"
    mkdir -p "$TEST_DATA_DIR"
    
    # Generate test data files
    create_test_data
    
    # Setup custom mock responses
    setup_custom_mocks
    
    # Set advanced test environment
    export TEST_SCENARIO_ID="${TEST_SCENARIO}_$$_$(date +%s)"
    export TEST_ADVANCED_MODE="true"
}

teardown() {
    # Advanced cleanup
    cleanup_test_scenario
    
    # Standard cleanup
    cleanup_mocks
}

create_test_data() {
    # Create various test data files
    
    # JSON test data
    cat > "$TEST_DATA_DIR/config.json" << 'EOF'
{
    "version": "1.0.0",
    "environment": "test",
    "resources": {
        "ollama": {
            "models": ["llama3.1:8b", "deepseek-r1:8b"],
            "max_concurrent": 5
        },
        "n8n": {
            "workflows": ["ai_pipeline", "data_processing"],
            "webhook_timeout": 30
        }
    },
    "test_settings": {
        "timeout": 60,
        "retry_count": 3,
        "log_level": "debug"
    }
}
EOF

    # Sample workflow definition
    cat > "$TEST_DATA_DIR/workflow.json" << 'EOF'
{
    "name": "AI Processing Workflow",
    "version": "1.0",
    "nodes": [
        {
            "id": "1",
            "type": "trigger",
            "name": "Webhook Trigger",
            "parameters": {
                "path": "/webhook/ai-process",
                "method": "POST"
            }
        },
        {
            "id": "2", 
            "type": "ollama",
            "name": "LLM Processing",
            "parameters": {
                "model": "llama3.1:8b",
                "prompt": "{{ $json.input }}"
            }
        },
        {
            "id": "3",
            "type": "response",
            "name": "Send Response", 
            "parameters": {
                "response": "{{ $node['LLM Processing'].json.response }}"
            }
        }
    ],
    "connections": {
        "Webhook Trigger": {"main": [["LLM Processing"]]},
        "LLM Processing": {"main": [["Send Response"]]}
    }
}
EOF

    # Test script
    cat > "$TEST_DATA_DIR/test_script.sh" << 'EOF'
#!/bin/bash
# Sample test script for advanced testing
echo "Running advanced test operations..."
curl -s "$OLLAMA_BASE_URL/api/tags" | jq -r '.models[].name'
curl -s "$N8N_BASE_URL/api/v1/workflows" | jq -r '.[].name'
echo "Test operations completed"
EOF
    chmod +x "$TEST_DATA_DIR/test_script.sh"
}

setup_custom_mocks() {
    # Setup custom HTTP response sequences for complex testing
    
    # Ollama API sequence: healthy -> processing -> completed
    mock::http::set_endpoint_sequence "$OLLAMA_BASE_URL/api/generate" \
        "200,202,200" \
        '{"status":"processing"},{"status":"processing","progress":50},{"response":"Generated response","done":true}'
    
    # N8N workflow execution sequence
    mock::http::set_endpoint_sequence "$N8N_BASE_URL/api/v1/workflows/execute" \
        "200,200,200" \
        '{"executionId":"exec_1","status":"running"},{"executionId":"exec_1","status":"running"},{"executionId":"exec_1","status":"success","data":{"result":"workflow completed"}}'
    
    # Setup error injection if enabled
    if [[ "$ENABLE_ERROR_INJECTION" == "true" ]]; then
        # Simulate temporary service failures
        mock::http::set_endpoint_delay "$OLLAMA_BASE_URL/api/generate" "0.1"  # 100ms delay
        mock::http::set_endpoint_delay "$N8N_BASE_URL/api/v1/workflows" "0.05" # 50ms delay
    fi
}

cleanup_test_scenario() {
    # Clean up test scenario data
    if [[ -d "$TEST_DATA_DIR" ]]; then
        rm -rf "$TEST_DATA_DIR"
    fi
    
    # Clean up any background processes
    jobs -p | xargs -r kill 2>/dev/null || true
}

# ================================
# ADVANCED TEST PATTERNS
# ================================

@test "test data is properly created" {
    # Verify test data setup
    assert_dir_exists "$TEST_DATA_DIR"
    assert_file_exists "$TEST_DATA_DIR/config.json"
    assert_file_exists "$TEST_DATA_DIR/workflow.json"
    assert_file_exists "$TEST_DATA_DIR/test_script.sh"
    
    # Validate JSON files
    assert_json_valid "$(cat "$TEST_DATA_DIR/config.json")"
    assert_json_valid "$(cat "$TEST_DATA_DIR/workflow.json")"
    
    # Test script should be executable
    assert_file_permissions "$TEST_DATA_DIR/test_script.sh" "755"
}

@test "configuration parsing and validation" {
    # Test complex JSON configuration parsing
    local config
    config=$(cat "$TEST_DATA_DIR/config.json")
    
    # Test nested JSON field access
    assert_json_field_equals "$config" ".version" "1.0.0"
    assert_json_field_equals "$config" ".environment" "test"
    assert_json_field_exists "$config" ".resources.ollama"
    assert_json_field_exists "$config" ".resources.n8n"
    
    # Test array handling
    local models
    models=$(echo "$config" | jq -r '.resources.ollama.models[]')
    assert_string_contains "$models" "llama3.1:8b"
    assert_string_contains "$models" "deepseek-r1:8b"
    
    # Test numeric values
    local timeout
    timeout=$(echo "$config" | jq -r '.test_settings.timeout')
    assert_equals "$timeout" "60"
}

@test "workflow definition validation" {
    # Test complex workflow structure validation
    local workflow
    workflow=$(cat "$TEST_DATA_DIR/workflow.json")
    
    # Basic structure validation
    assert_json_field_exists "$workflow" ".name"
    assert_json_field_exists "$workflow" ".nodes"
    assert_json_field_exists "$workflow" ".connections"
    
    # Node validation
    local node_count
    node_count=$(echo "$workflow" | jq '.nodes | length')
    assert_equals "$node_count" "3"
    
    # Connection validation
    local trigger_connections
    trigger_connections=$(echo "$workflow" | jq -r '.connections."Webhook Trigger".main[0][0]')
    assert_equals "$trigger_connections" "LLM Processing"
}

@test "mock response sequences work correctly" {
    # Test that mock response sequences return different responses on subsequent calls
    
    # First call should return "processing"
    local response1
    response1=$(curl -s "$OLLAMA_BASE_URL/api/generate" -d '{"prompt":"test"}')
    assert_json_field_equals "$response1" ".status" "processing"
    
    # Second call should return "processing" with progress
    local response2
    response2=$(curl -s "$OLLAMA_BASE_URL/api/generate" -d '{"prompt":"test"}')
    assert_json_field_equals "$response2" ".status" "processing"
    assert_json_field_exists "$response2" ".progress"
    
    # Third call should return completed response
    local response3
    response3=$(curl -s "$OLLAMA_BASE_URL/api/generate" -d '{"prompt":"test"}')
    assert_json_field_equals "$response3" ".done" "true"
    assert_json_field_exists "$response3" ".response"
}

@test "error injection and recovery works" {
    if [[ "$ENABLE_ERROR_INJECTION" != "true" ]]; then
        skip "Error injection disabled"
    fi
    
    # Test that services can handle temporary errors
    
    # Inject a temporary error
    mock::http::set_endpoint_response "$OLLAMA_BASE_URL/api/generate" \
        '{"error":"Service temporarily unavailable"}' 503
    
    # First call should fail
    run curl -s -w "%{http_code}" "$OLLAMA_BASE_URL/api/generate"
    assert_failure
    
    # Restore service
    mock::http::set_endpoint_response "$OLLAMA_BASE_URL/api/generate" \
        '{"response":"Service restored","done":true}' 200
    
    # Subsequent call should succeed
    run curl -s "$OLLAMA_BASE_URL/api/generate"
    assert_success
    assert_json_field_equals "$output" ".done" "true"
}

@test "complex workflow execution simulation" {
    # Test a complete workflow execution with multiple steps and error handling
    
    local workflow_id="test_workflow_${TEST_SCENARIO_ID}"
    
    # Step 1: Submit workflow
    local submit_response
    submit_response=$(curl -s "$N8N_BASE_URL/api/v1/workflows" \
        -X POST \
        -H "Content-Type: application/json" \
        -d @"$TEST_DATA_DIR/workflow.json")
    assert_json_valid "$submit_response"
    
    # Step 2: Execute workflow with test data
    local execution_response
    execution_response=$(curl -s "$N8N_BASE_URL/api/v1/workflows/execute" \
        -X POST \
        -H "Content-Type: application/json" \
        -d '{"workflowId":"'$workflow_id'","input":{"text":"Process this text"}}')
    assert_json_valid "$execution_response"
    
    # Step 3: Check execution status (using sequence)
    local status_response
    status_response=$(curl -s "$N8N_BASE_URL/api/v1/workflows/execute")
    assert_json_field_exists "$status_response" ".executionId"
    
    # Step 4: Process through LLM
    local llm_response  
    llm_response=$(curl -s "$OLLAMA_BASE_URL/api/generate" \
        -d '{"prompt":"Process this text","model":"llama3.1:8b"}')
    assert_json_valid "$llm_response"
    
    # Step 5: Verify final result
    local final_status
    final_status=$(curl -s "$N8N_BASE_URL/api/v1/workflows/execute")
    assert_json_field_equals "$final_status" ".status" "success"
}

@test "concurrent operations with complex state" {
    # Test concurrent operations while maintaining state consistency
    
    local pids=()
    local results_dir="$TEST_TMPDIR/concurrent_results"
    mkdir -p "$results_dir"
    
    # Start multiple concurrent operations
    for i in {1..5}; do
        (
            local result_file="$results_dir/result_$i.json"
            local response
            response=$(curl -s "$OLLAMA_BASE_URL/api/generate" \
                -d '{"prompt":"Concurrent request '$i'","model":"test"}')
            echo "$response" > "$result_file"
        ) &
        pids+=($!)
    done
    
    # Wait for all operations to complete
    for pid in "${pids[@]}"; do
        wait "$pid"
    done
    
    # Verify all results
    for i in {1..5}; do
        local result_file="$results_dir/result_$i.json"
        assert_file_exists "$result_file"
        assert_json_valid "$(cat "$result_file")"
    done
    
    # Verify no conflicts occurred
    local unique_responses
    unique_responses=$(cat "$results_dir"/*.json | jq -r '.response // .status' | sort -u | wc -l)
    assert_greater_than "$unique_responses" "0"
}

@test "resource health monitoring with detailed checks" {
    # Advanced health monitoring that checks multiple aspects
    
    for resource in "${RESOURCES[@]}"; do
        case "$resource" in
            "ollama")
                # Check Ollama health comprehensively
                assert_resource_healthy "ollama"
                
                # Check API endpoints
                run curl -s "$OLLAMA_BASE_URL/api/tags"
                assert_success
                assert_json_valid "$output"
                
                # Check model availability
                local models
                models=$(curl -s "$OLLAMA_BASE_URL/api/tags" | jq -r '.models[].name')
                assert_not_empty "$models"
                
                # Check container health
                assert_docker_container_running "$OLLAMA_CONTAINER_NAME"
                ;;
                
            "n8n")
                # Check N8N health comprehensively 
                assert_resource_healthy "n8n"
                
                # Check workflow API
                run curl -s "$N8N_BASE_URL/api/v1/workflows"
                assert_success
                assert_json_valid "$output"
                
                # Check container health
                assert_docker_container_running "$N8N_CONTAINER_NAME"
                ;;
        esac
    done
}

@test "test script execution and validation" {
    # Test that the generated test script works correctly
    local script_output
    
    # Execute the test script
    script_output=$("$TEST_DATA_DIR/test_script.sh" 2>&1)
    local exit_code=$?
    
    # Verify execution success
    assert_equals "$exit_code" "0"
    assert_not_empty "$script_output"
    assert_string_contains "$script_output" "Running advanced test operations"
    assert_string_contains "$script_output" "Test operations completed"
}

# ================================
# ADD YOUR CUSTOM ADVANCED TESTS HERE
# ================================

# @test "my custom complex scenario" {
#     # Your complex test scenario here
#     
#     # Setup complex test state
#     local test_state='{"phase":"initialization","data":{}}'
#     
#     # Execute multi-step operations
#     for step in {1..5}; do
#         # Update test state
#         test_state=$(echo "$test_state" | jq --arg step "$step" '.phase = "step_" + $step')
#         
#         # Perform step operations
#         # ... your logic here ...
#     done
#     
#     # Verify final state
#     assert_json_field_equals "$test_state" ".phase" "step_5"
# }