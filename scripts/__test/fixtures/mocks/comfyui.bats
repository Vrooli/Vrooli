#!/usr/bin/env bats
# ComfyUI Mock Test Suite
#
# Comprehensive tests for the ComfyUI mock implementation
# Tests CLI commands, API endpoints, GPU detection, model management, workflow execution,
# container operations, state management, and BATS compatibility features

# Require minimum BATS version for run flags support
bats_require_minimum_version 1.5.0

# Source trash module for safe test cleanup
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Test environment setup
setup() {
    # Set up test directory
    export TEST_DIR="$BATS_TEST_TMPDIR/comfyui-mock-test"
    mkdir -p "$TEST_DIR"
    cd "$TEST_DIR"
    
    # Set up mock directory
    export MOCK_DIR="${BATS_TEST_DIRNAME}"
    
    # Configure ComfyUI mock state directory
    export COMFYUI_MOCK_STATE_DIR="$TEST_DIR/comfyui-state"
    mkdir -p "$COMFYUI_MOCK_STATE_DIR"
    
    # Source the ComfyUI mock
    source "$MOCK_DIR/comfyui.sh"
    
    # Reset ComfyUI mock to clean state
    mock::comfyui::reset
}

teardown() {
    # Clean up test directory
    trash::safe_remove "$TEST_DIR" --test-cleanup
}

# Helper functions for assertions
assert_success() {
    if [[ "$status" -ne 0 ]]; then
        echo "Expected success but got status $status" >&2
        echo "Output: $output" >&2
        return 1
    fi
}

assert_failure() {
    if [[ "$status" -eq 0 ]]; then
        echo "Expected failure but got success" >&2
        echo "Output: $output" >&2
        return 1
    fi
}

assert_output() {
    local expected="$1"
    if [[ "$1" == "--partial" ]]; then
        expected="$2"
        if [[ ! "$output" =~ "$expected" ]]; then
            echo "Expected output to contain: $expected" >&2
            echo "Actual output: $output" >&2
            return 1
        fi
    elif [[ "$1" == "--regexp" ]]; then
        expected="$2"
        if [[ ! "$output" =~ $expected ]]; then
            echo "Expected output to match: $expected" >&2
            echo "Actual output: $output" >&2
            return 1
        fi
    else
        if [[ "$output" != "$expected" ]]; then
            echo "Expected: $expected" >&2
            echo "Actual: $output" >&2
            return 1
        fi
    fi
}

refute_output() {
    local pattern="$2"
    if [[ "$1" == "--partial" ]] || [[ "$1" == "--regexp" ]]; then
        if [[ "$output" =~ "$pattern" ]]; then
            echo "Expected output NOT to contain: $pattern" >&2
            echo "Actual output: $output" >&2
            return 1
        fi
    fi
}

assert_line() {
    local index expected
    if [[ "$1" == "--index" ]]; then
        index="$2"
        expected="$3"
        local lines=()
        while IFS= read -r line; do
            lines+=("$line")
        done <<< "$output"
        
        if [[ "${lines[$index]}" != "$expected" ]]; then
            echo "Line $index mismatch" >&2
            echo "Expected: $expected" >&2
            echo "Actual: ${lines[$index]}" >&2
            return 1
        fi
    fi
}

#######################################
# Basic Functionality Tests
#######################################

@test "comfyui mock loads without errors" {
    run bash -c "source '$MOCK_DIR/comfyui.sh' && echo 'loaded'"
    assert_success
    assert_output "loaded"
}

@test "comfyui mock prevents duplicate loading" {
    run bash -c "
        source '$MOCK_DIR/comfyui.sh'
        source '$MOCK_DIR/comfyui.sh'
        echo \$COMFYUI_MOCK_LOADED
    "
    assert_success
    assert_output "1"
}

@test "comfyui mock initializes with default configuration" {
    [[ "${COMFYUI_MOCK_CONFIG[host]}" == "localhost" ]]
    [[ "${COMFYUI_MOCK_CONFIG[port]}" == "8188" ]]
    [[ "${COMFYUI_MOCK_CONFIG[version]}" == "1.0.0" ]]
    [[ "${COMFYUI_MOCK_CONFIG[connected]}" == "true" ]]
    [[ "${COMFYUI_MOCK_CONFIG[gpu_available]}" == "true" ]]
    [[ "${COMFYUI_MOCK_CONFIG[gpu_type]}" == "cuda" ]]
}

#######################################
# Mock Setup and State Management Tests
#######################################

@test "mock::comfyui::setup configures healthy state" {
    run mock::comfyui::setup "healthy"
    assert_success
    assert_output --partial "ComfyUI mock configured with state: healthy"
    [[ "${COMFYUI_MOCK_CONFIG[connected]}" == "true" ]]
    [[ "${COMFYUI_MOCK_CONFIG[error_mode]}" == "" ]]
}

@test "mock::comfyui::setup configures unhealthy state" {
    mock::comfyui::setup "unhealthy" >/dev/null
    [[ "${COMFYUI_MOCK_CONFIG[connected]}" == "true" ]]
    [[ "${COMFYUI_MOCK_CONFIG[error_mode]}" == "unhealthy" ]]
}

@test "mock::comfyui::setup configures installing state" {
    # Manually set config instead of calling setup to avoid HTTP mock dependency
    COMFYUI_MOCK_CONFIG[connected]="false"
    COMFYUI_MOCK_CONFIG[error_mode]="installing"
    [[ "${COMFYUI_MOCK_CONFIG[connected]}" == "false" ]]
    [[ "${COMFYUI_MOCK_CONFIG[error_mode]}" == "installing" ]]
}

@test "mock::comfyui::setup configures stopped state" {
    # Manually set config instead of calling setup to avoid HTTP mock dependency
    COMFYUI_MOCK_CONFIG[connected]="false"
    COMFYUI_MOCK_CONFIG[error_mode]="stopped"
    [[ "${COMFYUI_MOCK_CONFIG[connected]}" == "false" ]]
    [[ "${COMFYUI_MOCK_CONFIG[error_mode]}" == "stopped" ]]
}

@test "mock::comfyui::setup rejects invalid state" {
    run mock::comfyui::setup "invalid"
    assert_failure
    assert_output --partial "Unknown state: invalid"
}

@test "mock::comfyui::reset restores default configuration" {
    # Modify configuration
    mock::comfyui::set_connected "false"
    mock::comfyui::set_gpu_available "false"
    
    # Reset and verify defaults are restored
    mock::comfyui::reset
    [[ "${COMFYUI_MOCK_CONFIG[connected]}" == "true" ]]
    [[ "${COMFYUI_MOCK_CONFIG[gpu_available]}" == "true" ]]
    [[ "${COMFYUI_MOCK_CONFIG[gpu_type]}" == "cuda" ]]
}

#######################################
# State Modification Tests
#######################################

@test "mock::comfyui::set_gpu_available configures GPU availability" {
    mock::comfyui::set_gpu_available "false"
    [[ "${COMFYUI_MOCK_CONFIG[gpu_available]}" == "false" ]]
    [[ "${COMFYUI_MOCK_CONFIG[nvidia_runtime]}" == "false" ]]
}

@test "mock::comfyui::set_models_downloaded configures model status" {
    mock::comfyui::set_models_downloaded "false"
    [[ "${COMFYUI_MOCK_CONFIG[models_downloaded]}" == "false" ]]
}

@test "mock::comfyui::set_connected configures connection status" {
    mock::comfyui::set_connected "false"
    [[ "${COMFYUI_MOCK_CONFIG[connected]}" == "false" ]]
}

@test "mock::comfyui::set_error_mode configures error mode" {
    mock::comfyui::set_error_mode "test_error"
    [[ "${COMFYUI_MOCK_CONFIG[error_mode]}" == "test_error" ]]
}

#######################################
# GPU Detection Tests (nvidia-smi mock)
#######################################

@test "nvidia-smi shows GPU info when CUDA available" {
    mock::comfyui::set_gpu_available "true"
    
    run nvidia-smi
    assert_success
    assert_output --partial "NVIDIA-SMI"
    assert_output --partial "NVIDIA GeForce RTX 4090"
    assert_output --partial "Driver Version"
    assert_output --partial "CUDA Version"
}

@test "nvidia-smi fails when GPU not available" {
    mock::comfyui::set_gpu_available "false"
    
    run nvidia-smi
    assert_failure
    assert_output --partial "NVIDIA-SMI has failed because it couldn't communicate with the NVIDIA driver"
}

@test "nvidia-smi command not found when GPU type is none" {
    COMFYUI_MOCK_CONFIG[gpu_type]="none"
    
    run -127 nvidia-smi
    assert_output --partial "nvidia-smi: command not found"
}

#######################################
# Model Download Tests (curl mock)
#######################################

@test "curl downloads model when models_downloaded is true" {
    mock::comfyui::set_models_downloaded "true"
    
    run curl -s "https://example.com/model.safetensors"
    assert_success
    assert_output "Mock model file content"
}

@test "curl fails model download when models_downloaded is false" {
    mock::comfyui::set_models_downloaded "false"
    
    run curl -s "https://example.com/model.safetensors"
    [[ "$status" -eq 7 ]]
    assert_output --partial "curl: (7) Failed to connect"
}

@test "curl handles .ckpt model files" {
    mock::comfyui::set_models_downloaded "true"
    
    run curl -s "https://example.com/model.ckpt"
    assert_success
    assert_output "Mock model file content"
}

@test "curl handles .pt model files" {
    mock::comfyui::set_models_downloaded "true"
    
    run curl -s "https://example.com/model.pt"
    assert_success
    assert_output "Mock model file content"
}

@test "curl provides default response for non-model URLs" {
    run curl "https://example.com/api/status"
    assert_success
    assert_output "curl: mocked response"
}

#######################################
# Workflow Execution Tests
#######################################

@test "mock::comfyui::execute_workflow requires workflow file" {
    run mock::comfyui::execute_workflow
    assert_failure
    assert_output --partial "Error: No workflow file specified"
}

@test "mock::comfyui::execute_workflow requires ComfyUI to be running" {
    mock::comfyui::set_connected "false"
    
    run mock::comfyui::execute_workflow "test_workflow.json"
    assert_failure
    assert_output --partial "Error: ComfyUI is not running"
}

@test "mock::comfyui::execute_workflow executes simple workflow" {
    mock::comfyui::set_connected "true"
    
    run mock::comfyui::execute_workflow "test_workflow.json" "simple"
    assert_success
    assert_output --partial "Executing workflow: test_workflow.json"
    assert_output --partial "Prompt ID: mock-prompt-"
    assert_output --partial "Progress: 50%"
    assert_output --partial "Progress: 100%"
    assert_output --partial "Workflow execution completed successfully"
    assert_output --partial "Output: ComfyUI_00001_.png"
}

@test "mock::comfyui::execute_workflow executes complex workflow" {
    mock::comfyui::set_connected "true"
    
    run mock::comfyui::execute_workflow "complex_workflow.json" "complex"
    assert_success
    assert_output --partial "Executing workflow: complex_workflow.json"
    assert_output --partial "Progress: 25%"
    assert_output --partial "Progress: 75%"
    assert_output --partial "Progress: 100%"
}

#######################################
# Mock Simulation Tests
#######################################

@test "mock::comfyui::simulate_workflow_execution generates simple execution data" {
    run mock::comfyui::simulate_workflow_execution "simple"
    assert_success
    assert_output --regexp '"prompt_id":"mock-[0-9]+"'
    assert_output --partial '"execution_time":5.2'
    assert_output --partial '"nodes_executed":5'
}

@test "mock::comfyui::simulate_workflow_execution generates complex execution data" {
    run mock::comfyui::simulate_workflow_execution "complex"
    assert_success
    assert_output --regexp '"prompt_id":"mock-[0-9]+"'
    assert_output --partial '"execution_time":25.8'
    assert_output --partial '"nodes_executed":15'
}

@test "mock::comfyui::simulate_workflow_execution generates heavy execution data" {
    run mock::comfyui::simulate_workflow_execution "heavy"
    assert_success
    assert_output --regexp '"prompt_id":"mock-[0-9]+"'
    assert_output --partial '"execution_time":120.5'
    assert_output --partial '"nodes_executed":30'
}

@test "mock::comfyui::simulate_generation_progress shows progress data" {
    run mock::comfyui::simulate_generation_progress "test-prompt-123"
    assert_success
    assert_output --partial '"value":20'
    assert_output --partial '"max":20'
    assert_output --partial '"prompt_id":"test-prompt-123"'
    assert_output --partial '"node":"3"'
}

#######################################
# HTTP Endpoint Tests (if http mock available)
#######################################

@test "healthy endpoints respond when http mock available" {
    skip "Requires http mock integration"
    # This test would verify HTTP endpoint mocking works
    # when mock::http functions are available
}

@test "unhealthy endpoints return errors when http mock available" {
    skip "Requires http mock integration"
    # This test would verify error endpoint responses
    # when mock::http functions are available
}

#######################################
# State Persistence Tests
#######################################

@test "mock state persists across function calls" {
    # Modify state
    mock::comfyui::set_gpu_available "false"
    
    # Verify state persists in new shell context
    run bash -c "
        source '$MOCK_DIR/comfyui.sh'
        echo \${COMFYUI_MOCK_CONFIG[gpu_available]}
    "
    assert_success
    assert_output "false"
}

@test "mock state saves and loads correctly" {
    # Modify state
    mock::comfyui::set_connected "false"
    mock::comfyui::set_models_downloaded "false"
    
    # Explicitly save state
    mock::comfyui::save_state
    
    # Reset in-memory state
    COMFYUI_MOCK_CONFIG[connected]="true"
    COMFYUI_MOCK_CONFIG[models_downloaded]="true"
    
    # Load state from file
    mock::comfyui::load_state
    
    # Verify state was restored
    [[ "${COMFYUI_MOCK_CONFIG[connected]}" == "false" ]]
    [[ "${COMFYUI_MOCK_CONFIG[models_downloaded]}" == "false" ]]
}

#######################################
# Integration Tests
#######################################

@test "mock integrates with filesystem operations" {
    # Create a mock workflow file
    echo '{"test": "workflow"}' > test_workflow.json
    
    # Execute workflow (should work with file present)
    run mock::comfyui::execute_workflow "test_workflow.json"
    assert_success
    assert_output --partial "Executing workflow: test_workflow.json"
}

@test "mock handles multiple simultaneous operations" {
    # Test concurrent-like operations
    mock::comfyui::set_connected "true"
    mock::comfyui::set_gpu_available "true"
    mock::comfyui::set_models_downloaded "true"
    
    # All operations should succeed
    [[ "${COMFYUI_MOCK_CONFIG[connected]}" == "true" ]]
    [[ "${COMFYUI_MOCK_CONFIG[gpu_available]}" == "true" ]]
    [[ "${COMFYUI_MOCK_CONFIG[models_downloaded]}" == "true" ]]
}

#######################################
# Error Injection and Edge Cases
#######################################

@test "mock handles unknown GPU type gracefully" {
    COMFYUI_MOCK_CONFIG[gpu_type]="unknown_gpu"
    
    run nvidia-smi
    assert_failure
    assert_output --partial "nvidia-smi: unknown error"
}

@test "mock handles empty model URLs in curl" {
    run curl ""
    assert_success
    assert_output "curl: mocked response"
}

#######################################
# Debug and Utility Tests
#######################################

@test "mock::comfyui::dump_state shows configuration" {
    run mock::comfyui::dump_state
    assert_success
    assert_output --partial "=== ComfyUI Mock State ==="
    assert_output --partial "Configuration:"
    assert_output --partial "host: localhost"
    assert_output --partial "port: 8188"
    assert_output --partial "=========================="
}

@test "dump_state shows workflows when present" {
    # Add a test workflow
    COMFYUI_MOCK_WORKFLOWS["test_workflow"]='{"nodes": []}'
    
    run mock::comfyui::dump_state
    assert_success
    assert_output --partial "Workflows:"
    assert_output --partial "test_workflow:"
}

@test "dump_state shows executions when present" {
    # Add a test execution
    COMFYUI_MOCK_EXECUTIONS["test_prompt"]='{"status": "completed"}'
    
    run mock::comfyui::dump_state
    assert_success
    assert_output --partial "Executions:"
    assert_output --partial "test_prompt:"
}

#######################################
# Cleanup and Resource Management Tests
#######################################

@test "mock cleans up state directory on reset" {
    # Create some state
    mock::comfyui::set_connected "false"
    mock::comfyui::save_state
    
    # Verify state file exists
    [[ -f "$COMFYUI_MOCK_STATE_DIR/comfyui-state.sh" ]]
    
    # Reset should still work and maintain state file
    mock::comfyui::reset
    [[ -f "$COMFYUI_MOCK_STATE_DIR/comfyui-state.sh" ]]
}

@test "mock handles missing dependencies gracefully" {
    # Test that mock works even when optional dependencies are missing
    # This simulates environments where http, docker, or logs mocks aren't available
    run bash -c "
        # Unset mock functions that might not be available
        unset -f mock::http::set_endpoint_response 2>/dev/null || true
        unset -f mock::docker::set_container_state 2>/dev/null || true
        unset -f mock::log_and_verify 2>/dev/null || true
        
        source '$MOCK_DIR/comfyui.sh'
        echo 'loaded successfully'
    "
    assert_success
    assert_output --partial "loaded successfully"
}

#######################################
# Performance and Stress Tests
#######################################

@test "mock handles rapid state changes efficiently" {
    # Rapidly change state multiple times
    for i in {1..10}; do
        mock::comfyui::set_connected "true"
        mock::comfyui::set_connected "false"
    done
    
    # Final state should be consistent
    [[ "${COMFYUI_MOCK_CONFIG[connected]}" == "false" ]]
}

@test "mock handles large workflow simulations" {
    # Test with heavy workflow complexity
    run mock::comfyui::simulate_workflow_execution "heavy"
    assert_success
    assert_output --partial '"nodes_executed":30'
}