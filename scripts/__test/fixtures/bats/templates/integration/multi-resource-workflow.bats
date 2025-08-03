#!/usr/bin/env bats
# Multi-Resource Workflow Integration Test Template
# Tests complex workflows involving multiple Vrooli resources working together
#
# Copy this template and customize it for your specific workflow needs.
# Example: AI content processing pipeline with multiple services

#######################################
# TEST CONFIGURATION
# Customize these variables for your workflow
#######################################

# Define the resources involved in your workflow
WORKFLOW_RESOURCES=("ollama" "whisper" "n8n" "questdb")
WORKFLOW_NAME="ai_content_pipeline"

# Define expected workflow stages
WORKFLOW_STAGES=(
    "audio_ingestion"
    "speech_to_text" 
    "ai_processing"
    "data_storage"
    "workflow_orchestration"
)

# Performance thresholds
MAX_WORKFLOW_DURATION=300  # 5 minutes
MAX_STAGE_DURATION=60      # 1 minute per stage

#######################################
# SETUP AND TEARDOWN
#######################################

# Load the test infrastructure
if [[ -n "${VROOLI_TEST_FIXTURES_DIR:-}" ]]; then
    source "${VROOLI_TEST_FIXTURES_DIR}/core/common_setup.bash"
else
    TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    source "${TEST_DIR}/../../core/common_setup.bash"
fi

setup() {
    # Setup integration test environment for multiple resources
    setup_integration_test "${WORKFLOW_RESOURCES[@]}"
    
    # Configure workflow-specific environment
    export WORKFLOW_ID="test_${WORKFLOW_NAME}_$$"
    export WORKFLOW_DATA_DIR="$TEST_TMPDIR/workflow_data"
    export WORKFLOW_CONFIG_FILE="$TEST_TMPDIR/workflow_config.json"
    
    # Create workflow directories
    mkdir -p "$WORKFLOW_DATA_DIR"/{input,output,temp}
    
    # Generate test workflow configuration
    cat > "$WORKFLOW_CONFIG_FILE" << EOF
{
    "workflow_id": "$WORKFLOW_ID",
    "name": "$WORKFLOW_NAME",
    "resources": $(printf '%s\n' "${WORKFLOW_RESOURCES[@]}" | jq -R . | jq -s .),
    "stages": $(printf '%s\n' "${WORKFLOW_STAGES[@]}" | jq -R . | jq -s .),
    "config": {
        "timeout": $MAX_WORKFLOW_DURATION,
        "retry_count": 3,
        "parallel_processing": true
    }
}
EOF

    # Setup mock verification expectations
    mock::verify::expect_calls "docker" "run.*ollama" 1
    mock::verify::expect_calls "docker" "run.*whisper" 1
    mock::verify::expect_calls "docker" "run.*n8n" 1
    mock::verify::expect_calls "docker" "run.*questdb" 1
    mock::verify::expect_calls "http" ".*health.*" 4  # Health checks for all resources
}

teardown() {
    # Validate all mock expectations were met
    mock::verify::validate_all
    
    # Clean up workflow data
    rm -rf "$WORKFLOW_DATA_DIR"
    rm -f "$WORKFLOW_CONFIG_FILE"
    
    # Standard teardown
    teardown_test_environment
}

#######################################
# WORKFLOW INFRASTRUCTURE TESTS
#######################################

@test "workflow: all required resources are available" {
    # Verify each resource can be contacted
    for resource in "${WORKFLOW_RESOURCES[@]}"; do
        assert_resource_healthy "$resource"
    done
}

@test "workflow: resource dependencies are satisfied" {
    # Test that resources can communicate with each other
    
    # Test Ollama API availability
    run curl -s "$OLLAMA_BASE_URL/api/tags"
    assert_success
    assert_json_valid "$output"
    assert_json_field_exists "$output" ".models"
    
    # Test Whisper API availability  
    run curl -s "$WHISPER_BASE_URL/health"
    assert_success
    assert_json_field_equals "$output" ".status" "ready"
    
    # Test N8N workflow API
    run curl -s "$N8N_BASE_URL/healthz"
    assert_success
    assert_json_field_equals "$output" ".status" "ok"
    
    # Test QuestDB connectivity
    run curl -s "$QUESTDB_BASE_URL/status"
    assert_success
    assert_json_field_equals "$output" ".status" "OK"
}

@test "workflow: environment configuration is valid" {
    # Verify workflow configuration file was created correctly
    assert_file_exists "$WORKFLOW_CONFIG_FILE"
    assert_file_contains "$WORKFLOW_CONFIG_FILE" "$WORKFLOW_ID"
    
    # Validate JSON structure
    run jq -r '.workflow_id' "$WORKFLOW_CONFIG_FILE"
    assert_success
    assert_output "$WORKFLOW_ID"
    
    # Check that all required resources are listed
    for resource in "${WORKFLOW_RESOURCES[@]}"; do
        run jq -r --arg resource "$resource" '.resources[] | select(. == $resource)' "$WORKFLOW_CONFIG_FILE"
        assert_success
        assert_output "$resource"
    done
}

#######################################
# WORKFLOW EXECUTION TESTS
#######################################

@test "workflow: stage 1 - audio ingestion" {
    local stage_start_time=$(date +%s)
    
    # Create test audio file
    local test_audio="$WORKFLOW_DATA_DIR/input/test_audio.wav"
    echo "mock audio data" > "$test_audio"
    
    # Verify file was created
    assert_file_exists "$test_audio"
    assert_file_contains "$test_audio" "mock audio data"
    
    # Validate stage completion time
    local stage_end_time=$(date +%s)
    local stage_duration=$((stage_end_time - stage_start_time))
    assert_less_than "$stage_duration" "$MAX_STAGE_DURATION"
    
    echo "Stage 1 completed in ${stage_duration}s"
}

@test "workflow: stage 2 - speech to text conversion" {
    local stage_start_time=$(date +%s)
    
    # Setup mock Whisper response
    mock::http::set_endpoint_response "$WHISPER_BASE_URL/transcribe" \
        '{"text":"This is a mock transcription result","language":"en","duration":5.2}' 200
    
    # Simulate audio transcription request
    local test_audio="$WORKFLOW_DATA_DIR/input/test_audio.wav"
    local transcription_result="$WORKFLOW_DATA_DIR/temp/transcription.json"
    
    run curl -s -X POST "$WHISPER_BASE_URL/transcribe" \
        -F "audio=@${test_audio}" \
        -o "$transcription_result"
    assert_success
    
    # Verify transcription output
    assert_file_exists "$transcription_result"
    
    run jq -r '.text' "$transcription_result"
    assert_success
    assert_output "This is a mock transcription result"
    
    # Validate stage completion time
    local stage_end_time=$(date +%s)
    local stage_duration=$((stage_end_time - stage_start_time))
    assert_less_than "$stage_duration" "$MAX_STAGE_DURATION"
    
    echo "Stage 2 completed in ${stage_duration}s"
}

@test "workflow: stage 3 - AI text processing" {
    local stage_start_time=$(date +%s)
    
    # Setup mock Ollama response
    mock::http::set_endpoint_response "$OLLAMA_BASE_URL/api/generate" \
        '{"response":"This is an AI-processed version of the transcribed text.","done":true}' 200
    
    # Simulate AI processing request
    local transcription="This is a mock transcription result"
    local ai_result="$WORKFLOW_DATA_DIR/temp/ai_processed.json"
    
    run curl -s -X POST "$OLLAMA_BASE_URL/api/generate" \
        -H "Content-Type: application/json" \
        -d '{"model":"llama3.1:8b","prompt":"Process this text: '"$transcription"'","stream":false}' \
        -o "$ai_result"
    assert_success
    
    # Verify AI processing output
    assert_file_exists "$ai_result"
    
    run jq -r '.response' "$ai_result"
    assert_success
    assert_output "This is an AI-processed version of the transcribed text."
    
    # Validate stage completion time
    local stage_end_time=$(date +%s)
    local stage_duration=$((stage_end_time - stage_start_time))
    assert_less_than "$stage_duration" "$MAX_STAGE_DURATION"
    
    echo "Stage 3 completed in ${stage_duration}s"
}

@test "workflow: stage 4 - data storage" {
    local stage_start_time=$(date +%s)
    
    # Setup mock QuestDB responses
    mock::http::set_endpoint_response "$QUESTDB_BASE_URL/exec?query=INSERT%20INTO%20workflow_metrics&fmt=json" \
        '{"query":"INSERT INTO workflow_metrics","ddl":"OK"}' 200
    
    # Simulate data storage
    local workflow_data='{"workflow_id":"'"$WORKFLOW_ID"'","stage":"data_storage","timestamp":"'"$(date -Iseconds)"'","status":"completed"}'
    local storage_result="$WORKFLOW_DATA_DIR/output/storage_result.json"
    
    # Insert data into QuestDB
    run curl -s -G "$QUESTDB_BASE_URL/exec" \
        --data-urlencode "query=INSERT INTO workflow_metrics VALUES ('$WORKFLOW_ID', 'data_storage', now(), 'completed')" \
        --data-urlencode "fmt=json" \
        -o "$storage_result"
    assert_success
    
    # Verify storage operation
    assert_file_exists "$storage_result"
    
    # Validate stage completion time
    local stage_end_time=$(date +%s)
    local stage_duration=$((stage_end_time - stage_start_time))
    assert_less_than "$stage_duration" "$MAX_STAGE_DURATION"
    
    echo "Stage 4 completed in ${stage_duration}s"
}

@test "workflow: stage 5 - workflow orchestration" {
    local stage_start_time=$(date +%s)
    
    # Setup mock N8N workflow execution
    mock::http::set_endpoint_response "$N8N_BASE_URL/api/v1/workflows/execute" \
        '{"executionId":"exec_123","status":"running"}' 200
    
    mock::http::set_endpoint_response "$N8N_BASE_URL/api/v1/executions/exec_123" \
        '{"id":"exec_123","status":"success","data":{"workflow_id":"'"$WORKFLOW_ID"'","stages_completed":4}}' 200
    
    # Trigger workflow execution
    local workflow_execution="$WORKFLOW_DATA_DIR/output/workflow_execution.json"
    
    run curl -s -X POST "$N8N_BASE_URL/api/v1/workflows/execute" \
        -H "Content-Type: application/json" \
        -d '{"workflowId":"ai_content_pipeline","data":{"workflow_id":"'"$WORKFLOW_ID"'"}}' \
        -o "$workflow_execution"
    assert_success
    
    # Check execution status
    local execution_id
    execution_id=$(jq -r '.executionId' "$workflow_execution")
    assert_output_not_empty "$execution_id"
    
    # Verify execution completion
    run curl -s "$N8N_BASE_URL/api/v1/executions/$execution_id"
    assert_success
    assert_json_field_equals "$output" ".status" "success"
    
    # Validate stage completion time
    local stage_end_time=$(date +%s)
    local stage_duration=$((stage_end_time - stage_start_time))
    assert_less_than "$stage_duration" "$MAX_STAGE_DURATION"
    
    echo "Stage 5 completed in ${stage_duration}s"
}

#######################################
# END-TO-END WORKFLOW TESTS
#######################################

@test "workflow: complete end-to-end execution" {
    local workflow_start_time=$(date +%s)
    
    # Execute all workflow stages in sequence
    echo "Starting complete workflow: $WORKFLOW_NAME"
    
    # Stage 1: Audio ingestion
    local test_audio="$WORKFLOW_DATA_DIR/input/complete_test.wav"
    echo "mock complete audio data" > "$test_audio"
    
    # Stage 2: Speech to text
    run curl -s -X POST "$WHISPER_BASE_URL/transcribe" -F "audio=@${test_audio}"
    assert_success
    local transcription=$(echo "$output" | jq -r '.text')
    
    # Stage 3: AI processing
    run curl -s -X POST "$OLLAMA_BASE_URL/api/generate" \
        -H "Content-Type: application/json" \
        -d '{"model":"llama3.1:8b","prompt":"Process: '"$transcription"'","stream":false}'
    assert_success
    local ai_result=$(echo "$output" | jq -r '.response')
    
    # Stage 4: Data storage
    run curl -s -G "$QUESTDB_BASE_URL/exec" \
        --data-urlencode "query=INSERT INTO workflow_metrics VALUES ('$WORKFLOW_ID', 'complete', now(), 'success')" \
        --data-urlencode "fmt=json"
    assert_success
    
    # Stage 5: Workflow orchestration
    run curl -s -X POST "$N8N_BASE_URL/api/v1/workflows/execute" \
        -H "Content-Type: application/json" \
        -d '{"workflowId":"complete_pipeline","data":{"result":"'"$ai_result"'"}}'
    assert_success
    
    # Validate complete workflow duration
    local workflow_end_time=$(date +%s)
    local total_duration=$((workflow_end_time - workflow_start_time))
    assert_less_than "$total_duration" "$MAX_WORKFLOW_DURATION"
    
    echo "Complete workflow executed in ${total_duration}s"
}

@test "workflow: error handling and recovery" {
    # Test workflow behavior when services fail
    
    # Simulate Whisper service failure
    mock::http::set_endpoint_response "$WHISPER_BASE_URL/transcribe" \
        '{"error":"Service temporarily unavailable"}' 503
    
    # Attempt transcription (should fail)
    run curl -s -X POST "$WHISPER_BASE_URL/transcribe" -F "audio=@/dev/null"
    assert_failure
    
    # Simulate service recovery
    mock::http::set_endpoint_response "$WHISPER_BASE_URL/transcribe" \
        '{"text":"Recovered transcription","language":"en","duration":3.0}' 200
    
    # Retry transcription (should succeed)
    run curl -s -X POST "$WHISPER_BASE_URL/transcribe" -F "audio=@/dev/null"
    assert_success
    assert_json_field_equals "$output" ".text" "Recovered transcription"
}

@test "workflow: performance and scaling validation" {
    # Test workflow performance with multiple concurrent requests
    
    local concurrent_requests=3
    local pids=()
    
    # Start multiple workflow executions
    for ((i=1; i<=concurrent_requests; i++)); do
        (
            local request_id="concurrent_$i"
            curl -s -X POST "$N8N_BASE_URL/api/v1/workflows/execute" \
                -H "Content-Type: application/json" \
                -d '{"workflowId":"performance_test","data":{"request_id":"'"$request_id"'"}}' \
                > "$WORKFLOW_DATA_DIR/output/concurrent_$i.json"
        ) &
        pids+=($!)
    done
    
    # Wait for all requests to complete
    for pid in "${pids[@]}"; do
        wait "$pid"
    done
    
    # Verify all requests completed successfully
    for ((i=1; i<=concurrent_requests; i++)); do
        local result_file="$WORKFLOW_DATA_DIR/output/concurrent_$i.json"
        assert_file_exists "$result_file"
        
        run jq -r '.status' "$result_file"
        assert_success
        assert_output "running"
    done
    
    echo "Successfully processed $concurrent_requests concurrent requests"
}

#######################################
# WORKFLOW VALIDATION AND CLEANUP
#######################################

@test "workflow: final validation and metrics" {
    # Validate that all workflow stages completed successfully
    echo "Validating workflow completion..."
    
    # Check that all expected files were created
    assert_directory_exists "$WORKFLOW_DATA_DIR/input"
    assert_directory_exists "$WORKFLOW_DATA_DIR/output" 
    assert_directory_exists "$WORKFLOW_DATA_DIR/temp"
    
    # Verify workflow configuration is still valid
    assert_file_exists "$WORKFLOW_CONFIG_FILE"
    
    # Check mock verification results
    mock::verify::print_summary
    
    # Validate resource health after workflow completion
    for resource in "${WORKFLOW_RESOURCES[@]}"; do
        assert_resource_healthy "$resource"
    done
    
    echo "Workflow validation completed successfully"
}

#######################################
# CUSTOM WORKFLOW HELPER FUNCTIONS
# Add your own workflow-specific test functions here
#######################################

# Helper: Simulate file upload to workflow
workflow_upload_file() {
    local file_path="$1"
    local upload_endpoint="$2"
    
    curl -s -X POST "$upload_endpoint" \
        -F "file=@${file_path}" \
        -H "X-Workflow-ID: $WORKFLOW_ID"
}

# Helper: Check workflow stage status
workflow_check_stage() {
    local stage_name="$1"
    
    curl -s -G "$QUESTDB_BASE_URL/exec" \
        --data-urlencode "query=SELECT status FROM workflow_metrics WHERE workflow_id='$WORKFLOW_ID' AND stage='$stage_name'" \
        --data-urlencode "fmt=json" \
        | jq -r '.dataset[0][0] // "not_found"'
}

# Helper: Wait for workflow stage completion
workflow_wait_for_stage() {
    local stage_name="$1"
    local timeout="${2:-60}"
    local elapsed=0
    
    while [[ $elapsed -lt $timeout ]]; do
        local status=$(workflow_check_stage "$stage_name")
        if [[ "$status" == "completed" ]]; then
            return 0
        fi
        sleep 2
        elapsed=$((elapsed + 2))
    done
    
    echo "Timeout waiting for stage: $stage_name" >&2
    return 1
}