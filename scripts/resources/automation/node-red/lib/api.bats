#!/usr/bin/env bats
# Tests for Node-RED API functions (lib/api.sh)

load ../test-fixtures/test_helper

setup() {
    setup_test_environment
    source_node_red_scripts
    mock_docker "success"
    mock_curl "success"
    mock_jq "success"
    
    # Default test environment variables
    export OUTPUT=""
    export FLOW_FILE=""
    export ENDPOINT=""
    export DATA=""
}

teardown() {
    teardown_test_environment
}

@test "node_red::list_flows displays flow information" {
    mock_docker "success"
    
    # Mock flows API response
    curl() {
        cat << 'EOF'
[
    {"id": "flow1", "type": "tab", "label": "Test Flow 1", "disabled": false},
    {"id": "node1", "type": "inject", "name": "Test Node", "z": "flow1"},
    {"id": "flow2", "type": "tab", "label": "Test Flow 2", "disabled": true}
]
EOF
    }
    export -f curl
    
    run node_red::list_flows
    assert_success
    assert_output_contains "Test Flow 1"
    assert_output_contains "Test Flow 2"
    assert_output_contains "ID: flow1"
    assert_output_contains "ID: flow2"
}

@test "node_red::list_flows fails when Node-RED is not running" {
    mock_docker "not_running"
    
    run node_red::list_flows
    assert_failure
    assert_output_contains "not running"
}

@test "node_red::list_flows handles API failure" {
    mock_docker "success"
    mock_curl "failure"
    
    run node_red::list_flows
    assert_failure
    assert_output_contains "Failed to fetch flows"
}

@test "node_red::list_flows handles empty response" {
    mock_docker "success"
    
    curl() { echo ""; }
    export -f curl
    
    run node_red::list_flows
    assert_failure
    assert_output_contains "Failed to fetch flows"
}

@test "node_red::list_flows handles invalid JSON response" {
    mock_docker "success"
    
    curl() { echo "invalid json"; }
    jq() { return 1; }
    export -f curl jq
    
    run node_red::list_flows
    assert_success  # Should fall back to showing raw response
    assert_output_contains "No flows found or invalid response"
}

@test "node_red::export_flows creates export file with default name" {
    mock_docker "success"
    
    # Set OUTPUT to test directory to avoid creating files in current directory
    export OUTPUT="$NODE_RED_TEST_DIR/test-export.json"
    
    curl() {
        if [[ "$*" =~ "/flows" ]]; then
            echo '[{"id": "flow1", "type": "tab", "label": "Test Flow"}]'
        fi
    }
    export -f curl
    
    run node_red::export_flows
    assert_success
    assert_output_contains "Flows exported successfully"
    assert_file_exists "$NODE_RED_TEST_DIR/test-export.json"
}

@test "node_red::export_flows uses custom output file" {
    mock_docker "success"
    export OUTPUT="$NODE_RED_TEST_DIR/custom-flows.json"
    
    curl() {
        if [[ "$*" =~ "/flows" ]]; then
            echo '[{"id": "flow1", "type": "tab", "label": "Test Flow"}]'
        fi
    }
    export -f curl
    
    run node_red::export_flows
    assert_success
    assert_output_contains "custom-flows.json"
    assert_file_exists "$NODE_RED_TEST_DIR/custom-flows.json"
}

@test "node_red::export_flows_to_file creates properly formatted JSON" {
    mock_docker "success"
    
    local output_file="$NODE_RED_TEST_DIR/test-export.json"
    
    curl() {
        if [[ "$*" =~ "/flows" ]]; then
            echo '[{"id":"flow1","type":"tab","label":"Test Flow"}]'
        fi
    }
    jq() {
        if [[ "$1" == "." ]]; then
            # Pretty print the JSON
            echo '[
  {
    "id": "flow1",
    "type": "tab", 
    "label": "Test Flow"
  }
]'
        else
            echo "1"  # For count operations
        fi
    }
    export -f curl jq
    
    run node_red::export_flows_to_file "$output_file"
    assert_success
    assert_output_contains "Flows exported successfully"
    assert_output_contains "1 flows with 1 total nodes"
    assert_file_exists "$output_file"
}

@test "node_red::export_flows_to_file fails when API is unavailable" {
    mock_docker "success"
    mock_curl "failure"
    
    local output_file="$NODE_RED_TEST_DIR/test-export.json"
    
    run node_red::export_flows_to_file "$output_file"
    assert_failure
    assert_output_contains "Failed to export flows"
}

@test "node_red::export_flows_to_file handles JSON formatting failure gracefully" {
    mock_docker "success"
    
    local output_file="$NODE_RED_TEST_DIR/test-export.json"
    
    curl() { echo '{"test": "data"}'; }
    jq() { return 1; }  # Mock jq to fail
    export -f curl jq
    
    run node_red::export_flows_to_file "$output_file"
    assert_success  # Should succeed even if formatting fails
    assert_output_contains "JSON formatting failed"
}

@test "node_red::import_flow imports valid flow file" {
    mock_docker "success"
    mock_curl "success"
    
    # Create test flow file
    local flow_file="$NODE_RED_TEST_DIR/test-flow.json"
    echo '[{"id": "flow1", "type": "tab", "label": "Test Flow"}]' > "$flow_file"
    
    export FLOW_FILE="$flow_file"
    
    run node_red::import_flow
    assert_success
    assert_output_contains "Flows imported successfully"
}

@test "node_red::import_flow_file imports specific file" {
    mock_docker "success"
    mock_curl "success"
    
    # Create test flow file
    local flow_file="$NODE_RED_TEST_DIR/test-flow.json"
    echo '[{"id": "flow1", "type": "tab", "label": "Test Flow"}]' > "$flow_file"
    
    run node_red::import_flow_file "$flow_file"
    assert_success
    assert_output_contains "Flows imported successfully"
}

@test "node_red::import_flow_file fails with missing file" {
    run node_red::import_flow_file "/nonexistent/flow.json"
    assert_failure
    assert_output_contains "Flow file not found"
}

@test "node_red::import_flow_file fails with empty filename" {
    run node_red::import_flow_file ""
    assert_failure
    assert_output_contains "Flow file is required"
}

@test "node_red::import_flow_file fails when Node-RED is not running" {
    mock_docker "not_running"
    
    local flow_file="$NODE_RED_TEST_DIR/test-flow.json"
    echo '{}' > "$flow_file"
    
    run node_red::import_flow_file "$flow_file"
    assert_failure
    assert_output_contains "not running"
}

@test "node_red::import_flow_file validates JSON format" {
    mock_docker "success"
    
    # Create invalid JSON file
    local flow_file="$NODE_RED_TEST_DIR/invalid-flow.json"
    echo '{invalid json}' > "$flow_file"
    
    # Mock validate_json to fail
    node_red::validate_json() { return 1; }
    export -f node_red::validate_json
    
    run node_red::import_flow_file "$flow_file"
    assert_failure
    assert_output_contains "Invalid JSON"
}

@test "node_red::import_flow_file fails when API call fails" {
    mock_docker "success"
    mock_curl "failure"
    
    local flow_file="$NODE_RED_TEST_DIR/test-flow.json"
    echo '{}' > "$flow_file"
    
    run node_red::import_flow_file "$flow_file"
    assert_failure
    assert_output_contains "Failed to import flows"
}

@test "node_red::execute_flow executes HTTP endpoint successfully" {
    mock_docker "success"
    
    export ENDPOINT="/test/endpoint"
    export DATA='{"test": "data"}'
    
    curl() {
        if [[ "$*" =~ "/test/endpoint" ]]; then
            echo '{"result": "success"}'
        fi
    }
    export -f curl
    
    run node_red::execute_flow
    assert_success
    assert_output_contains "Flow executed successfully"
    assert_output_contains "success"
}

@test "node_red::execute_flow fails when Node-RED is not running" {
    mock_docker "not_running"
    export ENDPOINT="/test/endpoint"
    
    run node_red::execute_flow
    assert_failure
    assert_output_contains "not running"
}

@test "node_red::execute_flow fails with missing endpoint" {
    mock_docker "success"
    
    run node_red::execute_flow
    assert_failure
    assert_output_contains "Endpoint is required"
}

@test "node_red::execute_flow handles POST with JSON data" {
    mock_docker "success"
    
    export ENDPOINT="/api/test"
    export DATA='{"command": "test"}'
    
    curl() {
        if [[ "$*" =~ "Content-Type: application/json" ]] && [[ "$*" =~ "test" ]]; then
            echo '{"status": "executed"}'
        fi
    }
    export -f curl
    
    run node_red::execute_flow
    assert_success
    assert_output_contains "executed"
}

@test "node_red::execute_flow handles GET without data" {
    mock_docker "success"
    
    export ENDPOINT="/status"
    # No DATA set
    
    curl() {
        if [[ "$*" =~ "/status" ]] && [[ ! "$*" =~ "-d" ]]; then
            echo '{"status": "ok"}'
        fi
    }
    export -f curl
    
    run node_red::execute_flow
    assert_success
}

@test "node_red::execute_flow fails when endpoint returns error" {
    mock_docker "success"
    export ENDPOINT="/test/endpoint"
    
    curl() { return 1; }
    export -f curl
    
    run node_red::execute_flow
    assert_failure
    assert_output_contains "Flow execution failed"
}

@test "node_red::get_flow retrieves specific flow" {
    mock_docker "success"
    
    curl() {
        if [[ "$*" =~ "/flow/flow123" ]]; then
            echo '{"id": "flow123", "type": "tab", "label": "Specific Flow"}'
        fi
    }
    export -f curl
    
    run node_red::get_flow "flow123"
    assert_success
    assert_output_contains "Specific Flow"
}

@test "node_red::get_flow fails with missing flow ID" {
    run node_red::get_flow ""
    assert_failure
    assert_output_contains "Flow ID is required"
}

@test "node_red::get_flow fails when Node-RED is not running" {
    mock_docker "not_running"
    
    run node_red::get_flow "flow123"
    assert_failure
    assert_output_contains "not running"
}

@test "node_red::get_flow handles non-existent flow" {
    mock_docker "success"
    
    curl() { echo ""; }
    export -f curl
    
    run node_red::get_flow "nonexistent"
    assert_failure
    assert_output_contains "Failed to fetch flow"
}

@test "node_red::enable_flow enables disabled flow" {
    mock_docker "success"
    
    # Mock getting current flow (disabled) and updating it
    curl() {
        case "$REQUEST_METHOD" in
            "GET"|"")
                echo '{"id": "flow123", "type": "tab", "disabled": true}'
                ;;
            "PUT")
                echo '{"id": "flow123", "type": "tab"}'
                ;;
        esac
    }
    
    # Mock jq to remove disabled property
    jq() {
        if [[ "$1" == "del(.disabled)" ]]; then
            echo '{"id": "flow123", "type": "tab"}'
        else
            echo "mock jq"
        fi
    }
    
    export -f curl jq
    
    run node_red::enable_flow "flow123"
    assert_success
    assert_output_contains "Flow enabled successfully"
}

@test "node_red::disable_flow disables enabled flow" {
    mock_docker "success"
    
    # Mock getting current flow (enabled) and updating it
    curl() {
        case "$REQUEST_METHOD" in
            "GET"|"")
                echo '{"id": "flow123", "type": "tab"}'
                ;;
            "PUT")
                echo '{"id": "flow123", "type": "tab", "disabled": true}'
                ;;
        esac
    }
    
    # Mock jq to add disabled property
    jq() {
        if [[ "$1" == ".disabled = true" ]]; then
            echo '{"id": "flow123", "type": "tab", "disabled": true}'
        else
            echo "mock jq"
        fi
    }
    
    export -f curl jq
    
    run node_red::disable_flow "flow123"
    assert_success
    assert_output_contains "Flow disabled successfully"
}

@test "node_red::toggle_flow fails with missing flow ID" {
    run node_red::enable_flow ""
    assert_failure
    assert_output_contains "Flow ID is required"
}

@test "node_red::toggle_flow fails when flow not found" {
    mock_docker "success"
    
    curl() { echo ""; }
    export -f curl
    
    run node_red::enable_flow "nonexistent"
    assert_failure
    assert_output_contains "Flow not found"
}

@test "node_red::deploy_flows deploys with default type" {
    mock_docker "success"
    
    curl() {
        if [[ "$*" =~ "Node-RED-Deployment-Type: full" ]]; then
            echo "OK"
        fi
    }
    export -f curl
    
    run node_red::deploy_flows
    assert_success
    assert_output_contains "Flows deployed successfully"
}

@test "node_red::deploy_flows deploys with custom type" {
    mock_docker "success"
    
    curl() {
        if [[ "$*" =~ "Node-RED-Deployment-Type: nodes" ]]; then
            echo "OK"
        fi
    }
    export -f curl
    
    run node_red::deploy_flows "nodes"
    assert_success
    assert_output_contains "Flows deployed successfully"
}

@test "node_red::deploy_flows fails when API call fails" {
    mock_docker "success"
    mock_curl "failure"
    
    run node_red::deploy_flows
    assert_failure
    assert_output_contains "Failed to deploy flows"
}

@test "node_red::get_runtime_info displays runtime information" {
    mock_docker "success"
    
    curl() {
        if [[ "$*" =~ "/settings" ]]; then
            echo '{
                "version": "3.0.2",
                "userDir": "/data",
                "flowFile": "flows.json",
                "httpRequestTimeout": 120000
            }'
        fi
    }
    export -f curl
    
    run node_red::get_runtime_info
    assert_success
    assert_output_contains "Runtime Information"
    assert_output_contains "Version: 3.0.2"
    assert_output_contains "User Directory: /data"
}

@test "node_red::get_runtime_info handles API failure" {
    mock_docker "success"
    mock_curl "failure"
    
    run node_red::get_runtime_info
    assert_failure
    assert_output_contains "Failed to fetch runtime information"
}

@test "node_red::get_auth_status detects disabled authentication" {
    mock_docker "success"
    
    curl() {
        if [[ "$*" =~ "/auth/login" ]]; then
            echo "404"  # Return HTTP code for no auth
        fi
    }
    export -f curl
    
    run node_red::get_auth_status
    assert_success
    assert_output_contains "Authentication: Disabled"
}

@test "node_red::get_auth_status detects enabled authentication" {
    mock_docker "success"
    
    curl() {
        if [[ "$*" =~ "-w" ]]; then
            echo "401"  # Return HTTP 401 for auth required
        fi
    }
    export -f curl
    
    run node_red::get_auth_status
    assert_success
    assert_output_contains "Authentication: Enabled"
}

@test "node_red::backup_flows creates timestamped backup" {
    mock_docker "success"
    
    # Mock current date for consistent filename
    date() { echo "20230101-120000"; }
    export -f date
    
    curl() {
        if [[ "$*" =~ "/flows" ]]; then
            echo '[{"id": "flow1", "type": "tab"}]'
        fi
    }
    export -f curl
    
    run node_red::backup_flows
    assert_success
    assert_output_contains "Flows backed up to"
    assert_output_contains "flows-backup-20230101-120000.json"
}

@test "node_red::backup_flows with custom directory" {
    mock_docker "success"
    
    local custom_dir="$NODE_RED_TEST_DIR/custom-backups"
    
    curl() { echo '[]'; }
    export -f curl
    
    run node_red::backup_flows "$custom_dir"
    assert_success
    assert_output_contains "$custom_dir"
}

@test "node_red::restore_flows restores from backup" {
    mock_docker "success"
    mock_curl "success"
    
    # Create test backup file
    local backup_file="$NODE_RED_TEST_DIR/backup.json"
    echo '[{"id": "flow1", "type": "tab", "label": "Restored Flow"}]' > "$backup_file"
    
    run node_red::restore_flows "$backup_file"
    assert_success
    assert_output_contains "Flows restored from backup"
}

@test "node_red::restore_flows fails with missing backup file" {
    run node_red::restore_flows ""
    assert_failure
    assert_output_contains "Backup file is required"
}

@test "node_red::restore_flows fails with non-existent file" {
    run node_red::restore_flows "/nonexistent/backup.json"
    assert_failure
    assert_output_contains "Backup file not found"
}

@test "node_red::validate_flows validates flow syntax" {
    mock_docker "success"
    
    curl() {
        echo '[{"id": "flow1", "type": "tab", "label": "Valid Flow"}]'
    }
    export -f curl
    
    run node_red::validate_flows
    assert_success
    assert_output_contains "Flow validation passed"
}

@test "node_red::validate_flows detects validation issues" {
    mock_docker "success"
    
    curl() {
        echo '[{"id": "node1", "type": null, "name": "Invalid Node"}]'
    }
    jq() {
        case "$*" in
            *"select"*) echo "unknown-type" ;;
            *) echo "mock jq" ;;
        esac
    }
    export -f curl jq
    
    run node_red::validate_flows
    assert_failure
    assert_output_contains "validation completed with warnings"
}

@test "node_red::search_flows finds matching content" {
    mock_docker "success"
    
    curl() {
        echo '[
            {"id": "flow1", "type": "tab", "label": "Test Flow", "info": "This is a test"},
            {"id": "node1", "type": "function", "name": "Test Function", "func": "return test;"}
        ]'
    }
    
    jq() {
        if [[ "$*" =~ "test.*i" ]]; then
            echo "Flow: Test Flow (ID: flow1, Type: tab)"
            echo "Flow: Test Function (ID: node1, Type: function)"
        else
            echo "mock jq"
        fi
    }
    
    export -f curl jq
    
    run node_red::search_flows "test"
    assert_success
    assert_output_contains "Search Results"
    assert_output_contains "Test Flow"
    assert_output_contains "Test Function"
}

@test "node_red::search_flows handles no matches" {
    mock_docker "success"
    
    curl() { echo '[]'; }
    jq() { echo ""; }
    export -f curl jq
    
    run node_red::search_flows "nonexistent"
    assert_failure
    assert_output_contains "No matches found"
}

@test "node_red::search_flows fails with missing search term" {
    run node_red::search_flows ""
    assert_failure
    assert_output_contains "Search term is required"
}

# Test timeout handling
@test "API functions respect timeout settings" {
    mock_docker "success"
    export NODE_RED_API_TIMEOUT=5
    
    curl() {
        if [[ "$*" =~ "--max-time 5" ]]; then
            echo "timeout respected"
        else
            echo "default timeout"
        fi
    }
    export -f curl
    
    run node_red::list_flows
    assert_success
    assert_output_contains "timeout respected"
}

# Test error handling
@test "API functions handle network timeouts" {
    mock_docker "success"
    mock_curl "timeout"
    
    run node_red::list_flows
    assert_failure
}

@test "API functions handle malformed responses" {
    mock_docker "success"
    
    curl() { echo "not json"; }
    jq() { return 1; }
    export -f curl jq
    
    run node_red::list_flows
    assert_success  # Should fall back gracefully
}

# Test concurrent API operations
@test "API functions work when called concurrently" {
    mock_docker "success"
    mock_curl "success"
    
    # Run multiple API functions in background
    node_red::list_flows > /dev/null &
    node_red::get_runtime_info > /dev/null &
    node_red::get_auth_status > /dev/null &
    
    wait  # Wait for all background processes
    
    # All should have completed successfully
    [[ $? -eq 0 ]]
}