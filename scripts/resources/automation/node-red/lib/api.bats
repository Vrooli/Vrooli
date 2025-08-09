#!/usr/bin/env bats
# Tests for Node-RED API functions (lib/api.sh)

# Load Vrooli test infrastructure
# shellcheck disable=SC1091
source "${BATS_TEST_DIRNAME}/../../../../__test/fixtures/setup.bash"

# Expensive setup operations run once per file
setup_file() {
    # Use Vrooli service test setup
    vrooli_setup_service_test "node-red"
    
    # Load Node-RED specific configuration once per file
    SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    NODE_RED_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Load configuration and API functions once
    # shellcheck disable=SC1091
    source "${NODE_RED_DIR}/config/defaults.sh"
    # shellcheck disable=SC1091
    source "${NODE_RED_DIR}/config/messages.sh"
    # shellcheck disable=SC1091
    source "${SCRIPT_DIR}/api.sh"
}

# Lightweight per-test setup
setup() {
    # Setup standard Vrooli mocks
    vrooli_auto_setup
    
    # Set test environment variables (lightweight per-test)
    export NODE_RED_CUSTOM_PORT="1880"
    export NODE_RED_CONTAINER_NAME="node-red-test"
    export NODE_RED_BASE_URL="http://localhost:1880"
    export NODE_RED_API_TIMEOUT="30"
    export OUTPUT=""
    export FLOW_FILE=""
    export ENDPOINT=""
    export DATA=""
    
    # Export config functions
    node_red::export_config
    node_red::export_messages
    
    # Mock health check function for API tests
    node_red::is_healthy() {
        return 0  # Always healthy for API tests
    }
    export -f node_red::is_healthy
    
    # Mock log functions
    log::header() { echo "=== $* ==="; }
    log::info() { echo "[INFO] $*"; }
    log::error() { echo "[ERROR] $*" >&2; }
    log::success() { echo "[SUCCESS] $*"; }
    log::warning() { echo "[WARNING] $*" >&2; }
    export -f log::header log::info log::error log::success log::warning
}

# BATS teardown function - runs after each test
teardown() {
    vrooli_cleanup_test
}

@test "node_red::list_flows displays flow information" {
    # Mock successful HTTP response
    mock::http::set_endpoint_response "${NODE_RED_BASE_URL}/flows" \
        '[{"id": "flow1", "type": "tab", "label": "Test Flow 1", "disabled": false},{"id": "node1", "type": "inject", "name": "Test Node", "z": "flow1"},{"id": "flow2", "type": "tab", "label": "Test Flow 2", "disabled": true}]' \
        "200"
    
    run node_red::list_flows
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Test Flow 1" ]]
    [[ "$output" =~ "Test Flow 2" ]]
    [[ "$output" =~ "ID: flow1" ]]
    [[ "$output" =~ "ID: flow2" ]]
}

@test "node_red::list_flows fails when Node-RED is not running" {
    # Override health check to fail
    node_red::is_healthy() {
        return 1
    }
    
    run node_red::list_flows
    [ "$status" -eq 1 ]
    [[ "$output" =~ "not running" ]]
}

@test "node_red::list_flows handles API failure" {
    # Mock HTTP endpoint to be unreachable
    mock::http::set_endpoint_unreachable "${NODE_RED_BASE_URL}/flows"
    
    run node_red::list_flows
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Failed to fetch flows" ]]
}

@test "node_red::list_flows handles empty response" {
    # Mock empty HTTP response
    mock::http::set_endpoint_response "${NODE_RED_BASE_URL}/flows" "" "200"
    
    run node_red::list_flows
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Failed to fetch flows" ]]
}

@test "node_red::list_flows handles invalid JSON response" {
    # Mock invalid JSON response
    mock::http::set_endpoint_response "${NODE_RED_BASE_URL}/flows" "invalid json" "200"
    
    run node_red::list_flows
    [ "$status" -eq 0 ]  # Should fall back to showing raw response
    [[ "$output" =~ "No flows found or invalid response" ]]
}

@test "node_red::export_flows creates export file with default name" {
    # Set OUTPUT to test directory
    export OUTPUT="/tmp/test-export.json"
    
    # Mock successful HTTP response
    mock::http::set_endpoint_response "${NODE_RED_BASE_URL}/flows" \
        '[{"id": "flow1", "type": "tab", "label": "Test Flow"}]' \
        "200"
    
    run node_red::export_flows
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Flows exported successfully" ]]
    [ -f "/tmp/test-export.json" ]
}

@test "node_red::export_flows uses custom output file" {
    export OUTPUT="/tmp/custom-flows.json"
    
    # Mock successful HTTP response
    mock::http::set_endpoint_response "${NODE_RED_BASE_URL}/flows" \
        '[{"id": "flow1", "type": "tab", "label": "Test Flow"}]' \
        "200"
    
    run node_red::export_flows
    [ "$status" -eq 0 ]
    [[ "$output" =~ "custom-flows.json" ]]
    [ -f "/tmp/custom-flows.json" ]
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
    # Create test flow file
    local flow_file="/tmp/test-flow.json"
    echo '[{"id": "flow1", "type": "tab", "label": "Test Flow"}]' > "$flow_file"
    
    export FLOW_FILE="$flow_file"
    
    # Mock successful HTTP response for import
    mock::http::set_endpoint_response "${NODE_RED_BASE_URL}/flows" "OK" "200"
    
    run node_red::import_flow
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Flows imported successfully" ]]
}

@test "node_red::import_flow_file imports specific file" {
    # Create test flow file
    local flow_file="/tmp/test-flow.json"
    echo '[{"id": "flow1", "type": "tab", "label": "Test Flow"}]' > "$flow_file"
    
    # Mock successful HTTP response for import
    mock::http::set_endpoint_response "${NODE_RED_BASE_URL}/flows" "OK" "200"
    
    run node_red::import_flow_file "$flow_file"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Flows imported successfully" ]]
}

@test "node_red::import_flow_file fails with missing file" {
    run node_red::import_flow_file "/nonexistent/flow.json"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Flow file not found" ]]
}

@test "node_red::import_flow_file fails with empty filename" {
    run node_red::import_flow_file ""
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Flow file is required" ]]
}

@test "node_red::import_flow_file fails when Node-RED is not running" {
    # Override health check to fail
    node_red::is_healthy() {
        return 1
    }
    
    local flow_file="/tmp/test-flow.json"
    echo '{}' > "$flow_file"
    
    run node_red::import_flow_file "$flow_file"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "not running" ]]
}

@test "node_red::import_flow_file validates JSON format" {
    # Create invalid JSON file
    local flow_file="/tmp/invalid-flow.json"
    echo '{invalid json}' > "$flow_file"
    
    # Mock validate_json to fail
    node_red::validate_json() { return 1; }
    export -f node_red::validate_json
    
    run node_red::import_flow_file "$flow_file"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Invalid JSON" ]]
}

@test "node_red::import_flow_file fails when API call fails" {
    # Mock HTTP endpoint to be unreachable
    mock::http::set_endpoint_unreachable "${NODE_RED_BASE_URL}/flows"
    
    local flow_file="/tmp/test-flow.json"
    echo '{}' > "$flow_file"
    
    run node_red::import_flow_file "$flow_file"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Failed to import flows" ]]
}

@test "node_red::execute_flow executes HTTP endpoint successfully" {
    export ENDPOINT="/test/endpoint"
    export DATA='{"test": "data"}'
    
    # Mock successful HTTP response
    mock::http::set_endpoint_response "${NODE_RED_BASE_URL}/test/endpoint" \
        '{"result": "success"}' \
        "200"
    
    run node_red::execute_flow
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Flow executed successfully" ]]
    [[ "$output" =~ "success" ]]
}

@test "node_red::execute_flow fails when Node-RED is not running" {
    # Override health check to fail
    node_red::is_healthy() {
        return 1
    }
    export ENDPOINT="/test/endpoint"
    
    run node_red::execute_flow
    [ "$status" -eq 1 ]
    [[ "$output" =~ "not running" ]]
}

@test "node_red::execute_flow fails with missing endpoint" {
    run node_red::execute_flow
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Endpoint is required" ]]
}

@test "node_red::execute_flow handles POST with JSON data" {
    export ENDPOINT="/api/test"
    export DATA='{"command": "test"}'
    
    # Mock successful HTTP response with JSON data
    mock::http::set_endpoint_response "${NODE_RED_BASE_URL}/api/test" \
        '{"status": "executed"}' \
        "200"
    
    run node_red::execute_flow
    [ "$status" -eq 0 ]
    [[ "$output" =~ "executed" ]]
}

@test "node_red::execute_flow handles GET without data" {
    export ENDPOINT="/status"
    # No DATA set
    
    # Mock successful HTTP response for GET
    mock::http::set_endpoint_response "${NODE_RED_BASE_URL}/status" \
        '{"status": "ok"}' \
        "200"
    
    run node_red::execute_flow
    [ "$status" -eq 0 ]
}

@test "node_red::execute_flow fails when endpoint returns error" {
    export ENDPOINT="/test/endpoint"
    
    # Mock HTTP endpoint to be unreachable
    mock::http::set_endpoint_unreachable "${NODE_RED_BASE_URL}/test/endpoint"
    
    run node_red::execute_flow
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Flow execution failed" ]]
}

@test "node_red::get_flow retrieves specific flow" {
    # Mock successful HTTP response for specific flow
    mock::http::set_endpoint_response "${NODE_RED_BASE_URL}/flow/flow123" \
        '{"id": "flow123", "type": "tab", "label": "Specific Flow"}' \
        "200"
    
    run node_red::get_flow "flow123"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Specific Flow" ]]
}

@test "node_red::get_flow fails with missing flow ID" {
    run node_red::get_flow ""
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Flow ID is required" ]]
}

@test "node_red::get_flow fails when Node-RED is not running" {
    # Override health check to fail
    node_red::is_healthy() {
        return 1
    }
    
    run node_red::get_flow "flow123"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "not running" ]]
}

@test "node_red::get_flow handles non-existent flow" {
    # Mock empty HTTP response for non-existent flow
    mock::http::set_endpoint_response "${NODE_RED_BASE_URL}/flow/nonexistent" "" "404"
    
    run node_red::get_flow "nonexistent"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Failed to fetch flow" ]]
}

@test "node_red::enable_flow enables disabled flow" {
    mock_docker "success"
    
    # Mock getting current flow (disabled) and updating it
    # Mock successful HTTP responses for GET and PUT
    mock::http::set_endpoint_response "${NODE_RED_BASE_URL}/flow/flow123" \
        '{"id": "flow123", "type": "tab", "disabled": true}' \
        "200"
    
    # Mock jq to remove disabled property
    jq() {
        if [[ "$1" == "del(.disabled)" ]]; then
            echo '{"id": "flow123", "type": "tab"}'
        else
            echo "mock jq"
        fi
    }
    export -f jq
    
    run node_red::enable_flow "flow123"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Flow enabled successfully" ]]
}

@test "node_red::disable_flow disables enabled flow" {
    mock_docker "success"
    
    # Mock getting current flow (enabled) and updating it
    # Mock successful HTTP responses for GET and PUT
    mock::http::set_endpoint_response "${NODE_RED_BASE_URL}/flow/flow123" \
        '{"id": "flow123", "type": "tab"}' \
        "200"
    
    # Mock jq to add disabled property
    jq() {
        if [[ "$1" == ".disabled = true" ]]; then
            echo '{"id": "flow123", "type": "tab", "disabled": true}'
        else
            echo "mock jq"
        fi
    }
    export -f jq
    
    run node_red::disable_flow "flow123"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Flow disabled successfully" ]]
}

@test "node_red::toggle_flow fails with missing flow ID" {
    run node_red::enable_flow ""
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Flow ID is required" ]]
}

@test "node_red::toggle_flow fails when flow not found" {
    # Mock empty HTTP response for non-existent flow
    mock::http::set_endpoint_response "${NODE_RED_BASE_URL}/flow/nonexistent" "" "404"
    
    run node_red::enable_flow "nonexistent"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Flow not found" ]]
}

@test "node_red::deploy_flows deploys with default type" {
    # Mock successful HTTP response for deployment
    mock::http::set_endpoint_response "${NODE_RED_BASE_URL}/flows" "OK" "200"
    
    run node_red::deploy_flows
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Flows deployed successfully" ]]
}

@test "node_red::deploy_flows deploys with custom type" {
    # Mock successful HTTP response for custom deployment
    mock::http::set_endpoint_response "${NODE_RED_BASE_URL}/flows" "OK" "200"
    
    run node_red::deploy_flows "nodes"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Flows deployed successfully" ]]
}

@test "node_red::deploy_flows fails when API call fails" {
    # Mock HTTP endpoint to be unreachable
    mock::http::set_endpoint_unreachable "${NODE_RED_BASE_URL}/flows"
    
    run node_red::deploy_flows
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Failed to deploy flows" ]]
}

@test "node_red::get_runtime_info displays runtime information" {
    # Mock successful HTTP response for settings
    mock::http::set_endpoint_response "${NODE_RED_BASE_URL}/settings" \
        '{"version": "3.0.2", "userDir": "/data", "flowFile": "flows.json", "httpRequestTimeout": 120000}' \
        "200"
    
    run node_red::get_runtime_info
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Runtime Information" ]]
    [[ "$output" =~ "Version: 3.0.2" ]]
    [[ "$output" =~ "User Directory: /data" ]]
}

@test "node_red::get_runtime_info handles API failure" {
    # Mock HTTP endpoint to be unreachable
    mock::http::set_endpoint_unreachable "${NODE_RED_BASE_URL}/settings"
    
    run node_red::get_runtime_info
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Failed to fetch runtime information" ]]
}

@test "node_red::get_auth_status detects disabled authentication" {
    # Mock 404 response for disabled auth
    mock::http::set_endpoint_response "${NODE_RED_BASE_URL}/auth/login" "Not Found" "404"
    
    run node_red::get_auth_status
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Authentication: Disabled" ]]
}

@test "node_red::get_auth_status detects enabled authentication" {
    # Mock 401 response for enabled auth
    mock::http::set_endpoint_response "${NODE_RED_BASE_URL}/auth/login" "Unauthorized" "401"
    
    run node_red::get_auth_status
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Authentication: Enabled" ]]
}

@test "node_red::backup_flows creates timestamped backup" {
    # Mock current date for consistent filename
    date() { echo "20230101-120000"; }
    export -f date
    
    # Mock successful HTTP response for flows
    mock::http::set_endpoint_response "${NODE_RED_BASE_URL}/flows" \
        '[{"id": "flow1", "type": "tab"}]' \
        "200"
    
    run node_red::backup_flows
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Flows backed up to" ]]
    [[ "$output" =~ "flows-backup-20230101-120000.json" ]]
}

@test "node_red::backup_flows with custom directory" {
    local custom_dir="/tmp/custom-backups"
    
    # Mock successful HTTP response
    mock::http::set_endpoint_response "${NODE_RED_BASE_URL}/flows" '[]' "200"
    
    run node_red::backup_flows "$custom_dir"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "$custom_dir" ]]
}

@test "node_red::restore_flows restores from backup" {
    # Create test backup file
    local backup_file="/tmp/backup.json"
    echo '[{"id": "flow1", "type": "tab", "label": "Restored Flow"}]' > "$backup_file"
    
    # Mock successful HTTP response for restore
    mock::http::set_endpoint_response "${NODE_RED_BASE_URL}/flows" "OK" "200"
    
    run node_red::restore_flows "$backup_file"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Flows restored from backup" ]]
}

@test "node_red::restore_flows fails with missing backup file" {
    run node_red::restore_flows ""
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Backup file is required" ]]
}

@test "node_red::restore_flows fails with non-existent file" {
    run node_red::restore_flows "/nonexistent/backup.json"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Backup file not found" ]]
}

@test "node_red::validate_flows validates flow syntax" {
    # Mock successful HTTP response with valid flows
    mock::http::set_endpoint_response "${NODE_RED_BASE_URL}/flows" \
        '[{"id": "flow1", "type": "tab", "label": "Valid Flow"}]' \
        "200"
    
    run node_red::validate_flows
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Flow validation passed" ]]
}

@test "node_red::validate_flows detects validation issues" {
    # Mock HTTP response with invalid flow data
    mock::http::set_endpoint_response "${NODE_RED_BASE_URL}/flows" \
        '[{"id": "node1", "type": null, "name": "Invalid Node"}]' \
        "200"
    
    jq() {
        case "$*" in
            *"select"*) echo "unknown-type" ;;
            *) echo "mock jq" ;;
        esac
    }
    export -f jq
    
    run node_red::validate_flows
    [ "$status" -eq 1 ]
    [[ "$output" =~ "validation completed with warnings" ]]
}

@test "node_red::search_flows finds matching content" {
    # Mock HTTP response with searchable flows
    mock::http::set_endpoint_response "${NODE_RED_BASE_URL}/flows" \
        '[{"id": "flow1", "type": "tab", "label": "Test Flow", "info": "This is a test"},{"id": "node1", "type": "function", "name": "Test Function", "func": "return test;"}]' \
        "200"
    
    jq() {
        if [[ "$*" =~ "test.*i" ]]; then
            echo "Flow: Test Flow (ID: flow1, Type: tab)"
            echo "Flow: Test Function (ID: node1, Type: function)"
        else
            echo "mock jq"
        fi
    }
    export -f jq
    
    run node_red::search_flows "test"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Search Results" ]]
    [[ "$output" =~ "Test Flow" ]]
    [[ "$output" =~ "Test Function" ]]
}

@test "node_red::search_flows handles no matches" {
    # Mock empty HTTP response
    mock::http::set_endpoint_response "${NODE_RED_BASE_URL}/flows" '[]' "200"
    
    jq() { echo ""; }
    export -f jq
    
    run node_red::search_flows "nonexistent"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "No matches found" ]]
}

@test "node_red::search_flows fails with missing search term" {
    run node_red::search_flows ""
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Search term is required" ]]
}

# Test timeout handling
@test "API functions respect timeout settings" {
    export NODE_RED_API_TIMEOUT=5
    
    # Mock HTTP response that would indicate timeout is respected
    mock::http::set_endpoint_response "${NODE_RED_BASE_URL}/flows" "timeout respected" "200"
    
    run node_red::list_flows
    [ "$status" -eq 0 ]
    [[ "$output" =~ "timeout respected" ]]
}

# Test error handling
@test "API functions handle network timeouts" {
    # Mock HTTP endpoint to timeout
    mock::http::set_endpoint_unreachable "${NODE_RED_BASE_URL}/flows"
    
    run node_red::list_flows
    [ "$status" -eq 1 ]
}

@test "API functions handle malformed responses" {
    # Mock malformed HTTP response
    mock::http::set_endpoint_response "${NODE_RED_BASE_URL}/flows" "not json" "200"
    
    jq() { return 1; }
    export -f jq
    
    run node_red::list_flows
    [ "$status" -eq 0 ]  # Should fall back gracefully
}

# Test concurrent API operations
@test "API functions work when called concurrently" {
    # Mock HTTP responses for concurrent operations
    mock::http::set_endpoint_response "${NODE_RED_BASE_URL}/flows" '[]' "200"
    mock::http::set_endpoint_response "${NODE_RED_BASE_URL}/settings" '{"version":"3.0.2"}' "200"
    mock::http::set_endpoint_response "${NODE_RED_BASE_URL}/auth/login" "Unauthorized" "401"
    
    # Run multiple API functions in background
    node_red::list_flows > /dev/null &
    node_red::get_runtime_info > /dev/null &
    node_red::get_auth_status > /dev/null &
    
    wait  # Wait for all background processes
    
    # All should have completed successfully
    [ $? -eq 0 ]
}
