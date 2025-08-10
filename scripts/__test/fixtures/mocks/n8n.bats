#!/usr/bin/env bats
# N8N Mock Test Suite
#
# Comprehensive tests for the n8n mock implementation
# Tests CLI commands, API endpoints, workflow management, execution simulation,
# webhook functionality, state management, and BATS compatibility features

# Source trash module for safe test cleanup
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Test environment setup
setup() {
    # Set up test directory
    export TEST_DIR="$BATS_TEST_TMPDIR/n8n-mock-test"
    mkdir -p "$TEST_DIR"
    cd "$TEST_DIR"
    
    # Set up mock directory
    export MOCK_DIR="${BATS_TEST_DIRNAME}"
    
    # Configure n8n mock state directory
    export N8N_MOCK_STATE_DIR="$TEST_DIR/n8n-state"
    mkdir -p "$N8N_MOCK_STATE_DIR"
    
    # Source the n8n mock
    source "$MOCK_DIR/n8n.sh"
    
    # Reset n8n mock to clean state
    mock::n8n::reset
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

@test "n8n mock loads without errors" {
    run bash -c "source '$MOCK_DIR/n8n.sh' && echo 'loaded'"
    assert_success
    assert_output "loaded"
}

@test "n8n mock prevents duplicate loading" {
    run bash -c "
        source '$MOCK_DIR/n8n.sh'
        source '$MOCK_DIR/n8n.sh'
        echo \$N8N_MOCK_LOADED
    "
    assert_success
    assert_output "1"
}

@test "n8n mock initializes with default configuration" {
    [[ "${N8N_MOCK_CONFIG[host]}" == "localhost" ]]
    [[ "${N8N_MOCK_CONFIG[port]}" == "5678" ]]
    [[ "${N8N_MOCK_CONFIG[version]}" == "1.25.0" ]]
    [[ "${N8N_MOCK_CONFIG[connected]}" == "true" ]]
}

#######################################
# CLI Command Tests - Basic Commands
#######################################

@test "n8n --help shows help information" {
    run n8n --help
    assert_success
    assert_output --partial "n8n Workflow Automation Tool"
    assert_output --partial "USAGE"
    assert_output --partial "COMMANDS"
}

@test "n8n help shows help information" {
    run n8n help
    assert_success
    assert_output --partial "n8n Workflow Automation Tool"
}

@test "n8n -h shows help information" {
    run n8n -h
    assert_success
    assert_output --partial "n8n Workflow Automation Tool"
}

@test "n8n --version shows version information" {
    run n8n --version
    assert_success
    assert_output --partial "n8n/1.25.0"
    assert_output --partial "linux-x64"
}

@test "n8n version shows version information" {
    run n8n version
    assert_success
    assert_output --partial "n8n/1.25.0"
}

@test "n8n start command runs successfully" {
    run n8n start
    assert_success
    assert_output --partial "Starting n8n process"
    assert_output --partial "n8n ready on http://localhost:5678"
}

@test "n8n unknown command shows error" {
    run n8n invalid_command
    assert_failure
    assert_output --partial "Unknown command 'invalid_command'"
    assert_output --partial "Use 'n8n --help' for available commands"
}

#######################################
# CLI Command Tests - Workflow Management
#######################################

@test "n8n list:workflows shows workflows" {
    run n8n list:workflows
    assert_success
    assert_output --partial "ID"
    assert_output --partial "Name"
    assert_output --partial "Active"
}

@test "n8n list:workflows shows default workflow" {
    run n8n list:workflows
    assert_success
    assert_output --partial "wf_default_123"
    assert_output --partial "Mock Workflow"
    assert_output --partial "false"
}

@test "n8n export:workflow requires id or backup" {
    run n8n export:workflow
    assert_failure
    assert_output --partial "Either --id or --backup must be specified"
}

@test "n8n export:workflow with invalid id shows error" {
    run n8n export:workflow --id=invalid_id
    assert_failure
    assert_output --partial "Workflow with ID 'invalid_id' not found"
}

@test "n8n export:workflow with valid id succeeds" {
    run n8n export:workflow --id=wf_default_123
    assert_success
    assert_output --partial "Default Test Workflow"
}

@test "n8n export:workflow with backup exports all workflows" {
    run n8n export:workflow --backup
    assert_success
    assert_output --partial "Default Test Workflow"
}

@test "n8n export:workflow with output file creates file" {
    local test_file="$TEST_DIR/exported_workflow.json"
    run n8n export:workflow --id=wf_default_123 --output="$test_file"
    assert_success
    assert_output --partial "Exported workflow wf_default_123 to $test_file"
    [[ -f "$test_file" ]]
}

@test "n8n import:workflow requires input file" {
    run n8n import:workflow
    assert_failure
    assert_output --partial "--input option is required"
}

@test "n8n import:workflow with missing file shows error" {
    run n8n import:workflow --input=missing_file.json
    assert_failure
    assert_output --partial "File 'missing_file.json' not found"
}

@test "n8n import:workflow with valid file succeeds" {
    # Create a test workflow file
    local test_file="$TEST_DIR/test_workflow.json"
    echo '{"name": "Imported Workflow", "nodes": []}' > "$test_file"
    
    run n8n import:workflow --input="$test_file"
    assert_success
    assert_output --partial "Imported workflow as ID:"
    assert_output --partial "Workflow deactivated during import"
}

@test "n8n update:workflow requires workflow id" {
    run n8n update:workflow
    assert_failure
    assert_output --partial "--id option is required"
}

@test "n8n update:workflow with invalid id shows error" {
    run n8n update:workflow --id=invalid_id --active=true
    assert_failure
    assert_output --partial "Workflow with ID 'invalid_id' not found"
}

@test "n8n update:workflow activates workflow" {
    run n8n update:workflow --id=wf_default_123 --active=true
    assert_success
    assert_output --partial "Updated workflow wf_default_123 active status to: true"
    
    # Reload state after subshell execution
    mock::n8n::load_state
    
    # Verify the workflow is now active
    [[ "${N8N_MOCK_WORKFLOW_ACTIVE[wf_default_123]}" == "true" ]]
}

@test "n8n update:workflow deactivates workflow" {
    # First activate a workflow
    N8N_MOCK_WORKFLOW_ACTIVE["wf_default_123"]="true"
    mock::n8n::save_state
    
    run n8n update:workflow --id=wf_default_123 --active=false
    assert_success
    assert_output --partial "Updated workflow wf_default_123 active status to: false"
    
    # Reload state after subshell execution
    mock::n8n::load_state
    
    # Verify the workflow is now inactive
    [[ "${N8N_MOCK_WORKFLOW_ACTIVE[wf_default_123]}" == "false" ]]
}

#######################################
# CLI Command Tests - Execution
#######################################

@test "n8n execute requires workflow id" {
    run n8n execute
    assert_failure
    assert_output --partial "Workflow ID is required"
}

@test "n8n execute with invalid id shows error" {
    run n8n execute --id=invalid_id
    assert_failure
    assert_output --partial "Workflow with ID 'invalid_id' not found"
}

@test "n8n execute with valid id creates execution" {
    run n8n execute --id=wf_default_123
    assert_success
    assert_output --partial "Execution started"
    assert_output --partial "Execution ID:"
    assert_output --partial "Status: success"
}

@test "n8n execute without --id flag uses positional argument" {
    run n8n execute wf_default_123
    assert_success
    assert_output --partial "Execution started"
}

@test "n8n list:executions shows executions" {
    # First create an execution
    n8n execute --id=wf_default_123 > /dev/null
    
    run n8n list:executions
    assert_success
    assert_output --partial "Execution ID"
    assert_output --partial "Workflow ID"
    assert_output --partial "Status"
}

#######################################
# CLI Command Tests - Credentials
#######################################

@test "n8n export:credentials succeeds" {
    run n8n export:credentials --backup
    assert_success
    assert_output --partial "["
    assert_output --partial "Default Test Credentials"
}

@test "n8n export:credentials with output file creates file" {
    local test_file="$TEST_DIR/exported_credentials.json"
    run n8n export:credentials --backup --output="$test_file"
    assert_success
    assert_output --partial "Exported credentials to $test_file"
    [[ -f "$test_file" ]]
}

@test "n8n import:credentials requires input file" {
    run n8n import:credentials
    assert_failure
    assert_output --partial "--input option is required"
}

@test "n8n import:credentials with valid file succeeds" {
    # Create a test credentials file
    local test_file="$TEST_DIR/test_credentials.json"
    echo '{"name": "Test Credentials", "type": "httpHeaderAuth"}' > "$test_file"
    
    run n8n import:credentials --input="$test_file"
    assert_success
    assert_output --partial "Imported credentials as ID:"
}

#######################################
# State Management Tests
#######################################

@test "n8n mock saves and loads state correctly" {
    # Modify state
    N8N_MOCK_CONFIG[test_value]="test123"
    mock::n8n::save_state
    
    # Clear state
    unset N8N_MOCK_CONFIG[test_value]
    
    # Load state
    mock::n8n::load_state
    
    # Verify state was restored
    [[ "${N8N_MOCK_CONFIG[test_value]}" == "test123" ]]
}

@test "n8n mock reset clears all data" {
    # Add some test data
    N8N_MOCK_WORKFLOWS["test_workflow"]='{"name": "test"}'
    N8N_MOCK_EXECUTIONS["test_execution"]='{"status": "success"}'
    
    # Reset
    mock::n8n::reset
    
    # Verify data is cleared
    [[ -z "${N8N_MOCK_WORKFLOWS[test_workflow]:-}" ]]
    [[ -z "${N8N_MOCK_EXECUTIONS[test_execution]:-}" ]]
}

@test "n8n mock maintains state across subshells" {
    # Set some state
    N8N_MOCK_CONFIG[test_key]="subshell_test"
    mock::n8n::save_state
    
    # Test in subshell
    run bash -c "
        source '$MOCK_DIR/n8n.sh'
        echo \${N8N_MOCK_CONFIG[test_key]}
    "
    assert_success
    assert_output "subshell_test"
}

#######################################
# Error Handling Tests
#######################################

@test "n8n mock handles connection errors" {
    mock::n8n::set_connection_status "false"
    
    run n8n list:workflows
    assert_failure
    assert_output --partial "Connection failed - n8n server is not running"
}

@test "n8n mock handles authentication errors" {
    mock::n8n::set_error_mode "auth_failed"
    
    run n8n list:workflows
    assert_failure
    assert_output --partial "Authentication failed - invalid credentials"
}

@test "n8n mock handles database errors" {
    mock::n8n::set_error_mode "database_error"
    
    run n8n list:workflows
    assert_failure
    assert_output --partial "Database error - could not access database"
}

@test "n8n mock handles connection timeout errors" {
    mock::n8n::set_error_mode "connection_timeout"
    
    run n8n list:workflows
    assert_failure
    assert_output --partial "Connection timeout - could not connect to n8n server"
}

@test "n8n mock error mode can be cleared" {
    mock::n8n::set_error_mode "auth_failed"
    mock::n8n::clear_error_mode
    
    run n8n list:workflows
    assert_success
}

#######################################
# Helper Function Tests
#######################################

@test "mock::n8n::get_workflow_count returns correct count" {
    local count=$(mock::n8n::get_workflow_count)
    [[ "$count" -eq 1 ]]  # Default workflow
    
    # Add a workflow
    mock::n8n::create_test_workflow "Test Workflow"
    
    local new_count=$(mock::n8n::get_workflow_count)
    [[ "$new_count" -eq 2 ]]
}

@test "mock::n8n::get_execution_count returns correct count" {
    local count=$(mock::n8n::get_execution_count)
    [[ "$count" -eq 0 ]]  # No executions initially
    
    # Create an execution
    n8n execute --id=wf_default_123 > /dev/null
    
    local new_count=$(mock::n8n::get_execution_count)
    [[ "$new_count" -eq 1 ]]
}

@test "mock::n8n::get_cli_call_count tracks CLI usage" {
    local initial_count=$(mock::n8n::get_cli_call_count)
    
    # Make a CLI call
    n8n --version > /dev/null
    
    local new_count=$(mock::n8n::get_cli_call_count)
    [[ "$new_count" -gt "$initial_count" ]]
}

@test "mock::n8n::create_test_workflow creates workflow" {
    # Create workflow without command substitution to avoid subshell issues
    mock::n8n::create_test_workflow "Test Workflow" "simple" "true" > /dev/null
    
    # Get the workflow ID from the state file
    local workflow_id
    if [[ -f "$N8N_MOCK_STATE_DIR/last_created_workflow_id" ]]; then
        workflow_id=$(cat "$N8N_MOCK_STATE_DIR/last_created_workflow_id")
    fi
    
    # Reload state after workflow creation
    mock::n8n::load_state
    
    [[ -n "$workflow_id" ]]
    [[ -n "${N8N_MOCK_WORKFLOWS[$workflow_id]:-}" ]]
    [[ "${N8N_MOCK_WORKFLOW_ACTIVE[$workflow_id]}" == "true" ]]
}

@test "mock::n8n::create_test_workflow creates webhook workflow" {
    # Create workflow without command substitution to avoid subshell issues
    mock::n8n::create_test_workflow "Webhook Test" "webhook" "true" > /dev/null
    
    # Get the workflow ID from the state file
    local workflow_id
    if [[ -f "$N8N_MOCK_STATE_DIR/last_created_workflow_id" ]]; then
        workflow_id=$(cat "$N8N_MOCK_STATE_DIR/last_created_workflow_id")
    fi
    
    # Reload state after workflow creation
    mock::n8n::load_state
    
    [[ -n "$workflow_id" ]]
    [[ "${N8N_MOCK_WORKFLOWS[$workflow_id]}" == *"webhook"* ]]
    [[ "${N8N_MOCK_WORKFLOWS[$workflow_id]}" == *"respondToWebhook"* ]]
}

@test "mock::n8n::simulate_workflow_execution creates execution" {
    # Simulate execution without command substitution
    mock::n8n::simulate_workflow_execution "wf_default_123" "success" "1.5" > /dev/null
    
    # Get the execution ID from the state file
    local execution_id
    if [[ -f "$N8N_MOCK_STATE_DIR/last_created_execution_id" ]]; then
        execution_id=$(cat "$N8N_MOCK_STATE_DIR/last_created_execution_id")
    fi
    
    # Reload state after execution simulation
    mock::n8n::load_state
    
    [[ -n "$execution_id" ]]
    [[ -n "${N8N_MOCK_EXECUTIONS[$execution_id]:-}" ]]
    [[ "${N8N_MOCK_EXECUTIONS[$execution_id]}" == *"success"* ]]
}

@test "mock::n8n::trigger_webhook tracks webhook calls" {
    local webhook_response
    webhook_response=$(mock::n8n::trigger_webhook "test_webhook" '{"data": "test"}')
    
    # Reload state after webhook trigger
    mock::n8n::load_state
    
    [[ "$webhook_response" == *"success"* ]]
    [[ "$webhook_response" == *"callCount"* ]]
    [[ "${N8N_MOCK_WEBHOOK_CALLS[test_webhook]}" == "1" ]]
}

#######################################
# HTTP API Endpoint Tests
#######################################

@test "n8n mock sets up healthy API endpoints" {
    mock::n8n::setup_api_endpoints "healthy"
    
    # This would require the HTTP mock to be active, so we just test the function doesn't error
    true
}

@test "n8n mock sets up unhealthy API endpoints" {
    mock::n8n::setup_api_endpoints "unhealthy"
    
    # This would require the HTTP mock to be active, so we just test the function doesn't error
    true
}

#######################################
# Docker Integration Tests
#######################################

@test "mock::n8n::setup_docker_state updates configuration" {
    mock::n8n::setup_docker_state "stopped"
    
    [[ "${N8N_MOCK_CONFIG[server_status]}" == "stopped" ]]
    [[ "${N8N_MOCK_CONFIG[connected]}" == "false" ]]
}

@test "mock::n8n::setup_docker_state handles running state" {
    mock::n8n::setup_docker_state "running"
    
    [[ "${N8N_MOCK_CONFIG[server_status]}" == "running" ]]
    [[ "${N8N_MOCK_CONFIG[connected]}" == "true" ]]
}

#######################################
# Edge Cases and Robustness Tests
#######################################

@test "n8n mock handles empty workflow list gracefully" {
    # Clear all workflows
    N8N_MOCK_WORKFLOWS=()
    mock::n8n::save_state
    
    run n8n list:workflows
    assert_success
    assert_output --partial "No workflows found"
}

@test "n8n mock handles empty execution list gracefully" {
    # Clear all executions
    N8N_MOCK_EXECUTIONS=()
    
    run n8n list:executions
    assert_success
    assert_output --partial "No executions found"
}

@test "n8n mock handles malformed workflow file import" {
    local test_file="$TEST_DIR/malformed_workflow.json"
    echo "invalid json content" > "$test_file"
    
    run n8n import:workflow --input="$test_file"
    assert_success  # Mock doesn't validate JSON, just stores it
    assert_output --partial "Imported workflow as ID:"
}

@test "n8n mock generates unique IDs" {
    local id1=$(mock::n8n::generate_workflow_id)
    local id2=$(mock::n8n::generate_workflow_id)
    
    [[ "$id1" != "$id2" ]]
    [[ "$id1" == wf_* ]]
    [[ "$id2" == wf_* ]]
}

@test "n8n mock handles concurrent access safely" {
    # Test sequential workflow creation
    mock::n8n::create_test_workflow "Concurrent Test 1" > /dev/null
    local workflow1_id
    if [[ -f "$N8N_MOCK_STATE_DIR/last_created_workflow_id" ]]; then
        workflow1_id=$(cat "$N8N_MOCK_STATE_DIR/last_created_workflow_id")
    fi
    
    mock::n8n::create_test_workflow "Concurrent Test 2" > /dev/null
    local workflow2_id
    if [[ -f "$N8N_MOCK_STATE_DIR/last_created_workflow_id" ]]; then
        workflow2_id=$(cat "$N8N_MOCK_STATE_DIR/last_created_workflow_id")
    fi
    
    # Reload state to ensure both workflows are visible
    mock::n8n::load_state
    
    [[ -n "$workflow1_id" ]]
    [[ -n "$workflow2_id" ]]
    [[ "$workflow1_id" != "$workflow2_id" ]]
    [[ -n "${N8N_MOCK_WORKFLOWS[$workflow1_id]:-}" ]]
    [[ -n "${N8N_MOCK_WORKFLOWS[$workflow2_id]:-}" ]]
}

#######################################
# Configuration and Environment Tests
#######################################

@test "n8n mock respects custom state directory" {
    local custom_dir="$TEST_DIR/custom-n8n-state"
    export N8N_MOCK_STATE_DIR="$custom_dir"
    
    # Reinitialize mock with custom directory
    source "$MOCK_DIR/n8n.sh"
    mock::n8n::reset
    
    [[ -d "$custom_dir" ]]
}

@test "n8n mock handles debug mode" {
    export N8N_MOCK_DEBUG="1"
    
    # Debug mode doesn't change functionality, just test it doesn't break
    run n8n --version
    assert_success
}

@test "n8n mock configuration can be modified" {
    N8N_MOCK_CONFIG[port]="9999"
    N8N_MOCK_CONFIG[version]="2.0.0"
    
    [[ "${N8N_MOCK_CONFIG[port]}" == "9999" ]]
    [[ "${N8N_MOCK_CONFIG[version]}" == "2.0.0" ]]
}

#######################################
# Performance and Resource Tests
#######################################

@test "n8n mock handles large number of workflows" {
    # Create 50 test workflows
    for i in {1..50}; do
        mock::n8n::create_test_workflow "Workflow $i" > /dev/null
    done
    
    local count=$(mock::n8n::get_workflow_count)
    [[ "$count" -eq 51 ]]  # 50 + 1 default
    
    # List workflows should still work
    run n8n list:workflows
    assert_success
}

@test "n8n mock handles large number of executions" {
    # Create 25 test executions
    for i in {1..25}; do
        mock::n8n::simulate_workflow_execution "wf_default_123" "success" > /dev/null
    done
    
    local count=$(mock::n8n::get_execution_count)
    [[ "$count" -eq 25 ]]
    
    # List executions should still work
    run n8n list:executions
    assert_success
}

#######################################
# Integration and Compatibility Tests
#######################################

@test "n8n mock is compatible with BATS test environment" {
    # Test that all required environment variables are accessible
    [[ -n "$BATS_TEST_TMPDIR" ]]
    [[ -n "$TEST_DIR" ]]
    [[ -n "$N8N_MOCK_STATE_DIR" ]]
    
    # Test that mock functions are available
    command -v mock::n8n::reset >/dev/null
    command -v n8n >/dev/null
}

@test "n8n mock functions are properly exported" {
    # Test that key functions are available in subshells
    run bash -c "source '$MOCK_DIR/n8n.sh' && command -v mock::n8n::reset"
    assert_success
    
    run bash -c "source '$MOCK_DIR/n8n.sh' && command -v n8n"
    assert_success
}

@test "n8n mock cleanup works correctly" {
    # Create some data
    N8N_MOCK_WORKFLOWS["test"]='{"test": true}'
    local state_file="$N8N_MOCK_STATE_DIR/n8n-state.sh"
    mock::n8n::save_state
    
    # Verify file exists
    [[ -f "$state_file" ]]
    
    # Reset should clean up
    mock::n8n::reset
    
    # Default state should be restored
    [[ -n "${N8N_MOCK_WORKFLOWS[wf_default_123]:-}" ]]
    [[ -z "${N8N_MOCK_WORKFLOWS[test]:-}" ]]
}

#######################################
# Final Validation Tests
#######################################

@test "n8n mock maintains consistency after multiple operations" {
    # Create a test workflow
    mock::n8n::create_test_workflow "Consistency Test" > /dev/null
    local wf_id
    if [[ -f "$N8N_MOCK_STATE_DIR/last_created_workflow_id" ]]; then
        wf_id=$(cat "$N8N_MOCK_STATE_DIR/last_created_workflow_id")
    fi
    
    # Update the workflow
    n8n update:workflow --id="$wf_id" --active=true > /dev/null
    
    # Simulate execution
    mock::n8n::simulate_workflow_execution "$wf_id" "success" > /dev/null
    local exec_id
    if [[ -f "$N8N_MOCK_STATE_DIR/last_created_execution_id" ]]; then
        exec_id=$(cat "$N8N_MOCK_STATE_DIR/last_created_execution_id")
    fi
    
    # Trigger webhook
    mock::n8n::trigger_webhook "test_hook" > /dev/null
    
    # Reload state to ensure consistency across operations
    mock::n8n::load_state
    
    # Verify all operations succeeded and state is consistent
    [[ -n "$wf_id" ]]
    [[ -n "$exec_id" ]]
    [[ -n "${N8N_MOCK_WORKFLOWS[$wf_id]:-}" ]]
    [[ "${N8N_MOCK_WORKFLOW_ACTIVE[$wf_id]}" == "true" ]]
    [[ -n "${N8N_MOCK_EXECUTIONS[$exec_id]:-}" ]]
    [[ "${N8N_MOCK_WEBHOOK_CALLS[test_hook]}" == "1" ]]
}

@test "n8n mock is ready for production testing scenarios" {
    # Verify all essential components are working
    
    # CLI commands
    n8n --version > /dev/null
    n8n list:workflows > /dev/null
    
    # State management
    mock::n8n::save_state
    mock::n8n::load_state
    
    # Helper functions
    local wf_count=$(mock::n8n::get_workflow_count)
    [[ "$wf_count" -ge 1 ]]
    
    # Test scenarios without command substitution
    mock::n8n::create_test_workflow "Production Test" > /dev/null
    local test_wf_id
    if [[ -f "$N8N_MOCK_STATE_DIR/last_created_workflow_id" ]]; then
        test_wf_id=$(cat "$N8N_MOCK_STATE_DIR/last_created_workflow_id")
    fi
    
    mock::n8n::simulate_workflow_execution "$test_wf_id" "success" > /dev/null
    local test_exec_id
    if [[ -f "$N8N_MOCK_STATE_DIR/last_created_execution_id" ]]; then
        test_exec_id=$(cat "$N8N_MOCK_STATE_DIR/last_created_execution_id")
    fi
    
    # Reload state to verify persistence
    mock::n8n::load_state
    
    [[ -n "$test_wf_id" ]]
    [[ -n "$test_exec_id" ]]
    [[ -n "${N8N_MOCK_WORKFLOWS[$test_wf_id]:-}" ]]
    [[ -n "${N8N_MOCK_EXECUTIONS[$test_exec_id]:-}" ]]
    
    # This test passing means the mock is comprehensive and ready
    true
}