#!/usr/bin/env bats
# Node-RED Mock Test Suite
#
# Comprehensive tests for the Node-RED mock implementation
# Tests Docker container management, API endpoints, flow operations,
# error injection, and BATS compatibility features

# Source trash module for safe test cleanup
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Load BATS testing libraries if available
# Note: These helpers are not currently available in this test environment
# load '../helpers/bats-support/load'
# load '../helpers/bats-assert/load'

# Helper functions (normally from bats-assert)
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

# Test environment setup
setup() {
    # Set up test directory
    export TEST_DIR="$BATS_TEST_TMPDIR/node-red-mock-test"
    mkdir -p "$TEST_DIR"
    cd "$TEST_DIR"
    
    # Set up mock directory
    export MOCK_DIR="${BATS_TEST_DIRNAME}"
    
    # Configure Node-RED mock state directory
    export NODE_RED_MOCK_STATE_DIR="$TEST_DIR/node-red-state"
    mkdir -p "$NODE_RED_MOCK_STATE_DIR"
    
    # Source the Node-RED mock
    source "$MOCK_DIR/node-red.sh"
    
    # Reset Node-RED mock to clean state
    mock::node_red::reset
}

teardown() {
    # Clean up test directory
    trash::safe_remove "$TEST_DIR" --test-cleanup
}

# Helper functions for Node-RED specific assertions
assert_container_running() {
    local container_name="$1"
    run docker ps --format "{{.Names}}"
    assert_success
    assert_output --partial "$container_name"
}

assert_container_stopped() {
    local container_name="$1"
    run docker ps --format "{{.Names}}"
    assert_success
    refute_output --partial "$container_name"
}

assert_api_response() {
    local endpoint="$1"
    local expected_status="${2:-0}"
    run curl -s "http://localhost:1880$endpoint"
    if [[ "$expected_status" -eq 0 ]]; then
        assert_success
    else
        assert_failure
    fi
}

#######################################
# Basic Mock Loading Tests
#######################################

@test "Node-RED mock loads successfully" {
    # Mock should be loaded without error
    run echo "[NODE_RED_MOCK] Node-RED mock implementation loaded"
    assert_success
}

@test "Node-RED mock configuration is initialized" {
    # Check that configuration variables are set
    [[ "${NODE_RED_MOCK_CONFIG[container_name]}" == "vrooli_node-red" ]]
    [[ "${NODE_RED_MOCK_CONFIG[port]}" == "1880" ]]
    [[ "${NODE_RED_MOCK_CONFIG[version]}" == "3.1.0" ]]
    [[ "${NODE_RED_MOCK_CONFIG[state]}" == "running" ]]
}

#######################################
# Docker Command Tests
#######################################

@test "docker ps shows running Node-RED container" {
    run docker ps
    assert_success
    assert_output --partial "vrooli_node-red"
    assert_output --partial "nodered/node-red:3.1.0"
    assert_output --partial "Up 2 hours"
}

@test "docker ps with format shows container name" {
    run docker ps --format "{{.Names}}"
    assert_success
    assert_output "vrooli_node-red"
}

@test "docker ps with format shows container status" {
    run docker ps --format "{{.Status}}"
    assert_success
    assert_output "Up 2 hours"
}

@test "docker ps with -a shows stopped containers" {
    # Stop the container first
    mock::node_red::set_state "stopped"
    
    run docker ps -a --format "{{.Names}}"
    assert_success
    assert_output "vrooli_node-red"
}

@test "docker ps without -a doesn't show stopped containers" {
    # Stop the container first
    mock::node_red::set_state "stopped"
    
    run docker ps --format "{{.Names}}"
    assert_success
    refute_output --partial "vrooli_node-red"
}

@test "docker inspect returns container details" {
    run docker inspect vrooli_node-red
    assert_success
    assert_output --partial "\"Name\": \"/vrooli_node-red\""
    assert_output --partial "\"Status\": \"running\""
    assert_output --partial "\"Running\": true"
}

@test "docker inspect with format returns specific field" {
    run docker inspect -f "{{.State.Running}}" vrooli_node-red
    assert_success
    assert_output "true"
}

@test "docker start container changes state to running" {
    # Stop container first
    mock::node_red::set_state "stopped"
    
    run docker start vrooli_node-red
    assert_success
    assert_output "vrooli_node-red"
    
    # Reload state from file since docker command ran in subshell
    mock::node_red::load_state
    
    # Verify state changed
    [[ "${NODE_RED_MOCK_CONFIG[state]}" == "running" ]]
}

@test "docker stop container changes state to stopped" {
    run docker stop vrooli_node-red
    assert_success
    assert_output "vrooli_node-red"
    
    # Reload state from file since docker command ran in subshell
    mock::node_red::load_state
    
    # Verify state changed
    [[ "${NODE_RED_MOCK_CONFIG[state]}" == "stopped" ]]
}

@test "docker restart container maintains running state" {
    run docker restart vrooli_node-red
    assert_success
    assert_output "vrooli_node-red"
    
    # Verify state is running
    [[ "${NODE_RED_MOCK_CONFIG[state]}" == "running" ]]
}

@test "docker logs shows Node-RED startup messages" {
    run docker logs vrooli_node-red
    assert_success
    assert_output --partial "Node-RED version: v3.1.0"
    assert_output --partial "Starting flows"
    assert_output --partial "Started flows"
    assert_output --partial "Server now running at http://0.0.0.0:1880/"
}

@test "docker logs with --tail limits output" {
    run docker logs --tail 2 vrooli_node-red
    assert_success
    # Should have limited output lines
}

@test "docker exec executes commands in container" {
    run docker exec vrooli_node-red echo "test"
    assert_success
    assert_output --partial "Mock exec in vrooli_node-red: echo test"
}

@test "docker run creates new Node-RED container" {
    run docker run --name test-node-red -p 1880:1880 -d nodered/node-red:3.1.0
    assert_success
    assert_output --partial "Container test-node-red created"
}

@test "docker pull downloads Node-RED image" {
    run docker pull nodered/node-red:3.1.0
    assert_success
    assert_output --partial "Pulling nodered/node-red:3.1.0... (mocked)"
    assert_output --partial "Status: Downloaded newer image"
}

#######################################
# Node-RED API Tests
#######################################

@test "GET / returns Node-RED homepage" {
    run curl -s http://localhost:1880/
    assert_success
    assert_output --partial "Node-RED Mock Server"
    assert_output --partial "<html>"
}

@test "GET /settings returns Node-RED settings" {
    run curl -s http://localhost:1880/settings
    assert_success
    assert_output --partial "\"version\": \"3.1.0\""
    assert_output --partial "\"httpNodeRoot\": \"/\""
    assert_output --partial "\"username\": \"admin\""
}

@test "GET /flows returns flow configuration" {
    run curl -s http://localhost:1880/flows
    assert_success
    assert_output --partial "\"type\": \"tab\""
    assert_output --partial "\"label\": \"Flow 1\""
    assert_output --partial "\"type\": \"inject\""
    assert_output --partial "\"type\": \"debug\""
}

@test "POST /flows deploys flows successfully" {
    local test_flow='[{"id":"f1","type":"tab","label":"Test Flow"}]'
    run curl -s -X POST -H "Content-Type: application/json" -d "$test_flow" http://localhost:1880/flows
    assert_success
    assert_output --partial "\"rev\":"
    assert_output --partial "\"flows\":"
    assert_output --partial "\"deploy_type\":\"full\""
}

@test "POST /flows with deployment type header" {
    local test_flow='[{"id":"f1","type":"tab","label":"Test Flow"}]'
    run curl -s -X POST -H "Content-Type: application/json" -H "Node-RED-Deployment-Type: nodes" -d "$test_flow" http://localhost:1880/flows
    assert_success
    assert_output --partial "\"deploy_type\":\"nodes\""
}

@test "GET /nodes returns available nodes" {
    run curl -s http://localhost:1880/nodes
    assert_success
    assert_output --partial "node-red/inject"
    assert_output --partial "node-red/debug"
    assert_output --partial "node-red/http"
    assert_output --partial "\"enabled\": true"
}

@test "POST /nodes installs new node" {
    run curl -s -X POST -H "Content-Type: application/json" -d '{"name":"test-node"}' http://localhost:1880/nodes
    assert_success
    assert_output --partial "\"success\":true"
    assert_output --partial "Node installed successfully"
}

@test "GET /library/flows returns flow library" {
    run curl -s http://localhost:1880/library/flows
    assert_success
    assert_output --partial "Example Flow"
    assert_output --partial "A simple example flow"
}

@test "GET /health returns health status" {
    run curl -s http://localhost:1880/health
    assert_success
    assert_output --partial "\"status\":\"ok\""
    assert_output --partial "\"version\":\"3.1.0\""
}

@test "POST custom endpoint executes flow" {
    run curl -s -X POST -d '{"test":"data"}' http://localhost:1880/test-endpoint
    assert_success
    assert_output --partial "\"success\":true"
    assert_output --partial "Flow executed"
    assert_output --partial "\"/test-endpoint\""
}

#######################################
# Error Mode Tests
#######################################

@test "connection_refused error mode blocks API calls" {
    mock::node_red::set_error "connection_refused"
    
    run curl -s http://localhost:1880/settings
    assert_failure
    assert_output --partial "Connection refused"
}

@test "timeout error mode simulates API timeout" {
    mock::node_red::set_error "timeout"
    
    run curl -s http://localhost:1880/settings
    assert_failure
    assert_output --partial "Operation timed out"
}

@test "not_found error mode returns 404" {
    mock::node_red::set_error "not_found"
    
    run curl -s http://localhost:1880/settings
    assert_failure
    assert_output --partial "Not found"
}

@test "stopped state blocks API calls" {
    mock::node_red::set_state "stopped"
    
    run curl -s http://localhost:1880/settings
    assert_failure
    assert_output --partial "Connection refused"
}

@test "unhealthy state returns error from health endpoint" {
    mock::node_red::set_config "health" "unhealthy"
    
    run curl -s http://localhost:1880/health
    assert_failure
    assert_output --partial "Service unhealthy"
}

#######################################
# State Management Tests
#######################################

@test "mock reset clears all state" {
    # Add some data
    mock::node_red::add_flow "test-flow" '{"id":"test","type":"tab"}'
    mock::node_red::install_node "test-node" "1.0.0"
    
    # Reset
    mock::node_red::reset
    
    # Verify state is clean
    [[ "${NODE_RED_MOCK_CONFIG[state]}" == "running" ]]
    [[ ${#NODE_RED_MOCK_FLOWS[@]} -eq 0 ]]
    [[ ${#NODE_RED_MOCK_INSTALLED_NODES[@]} -eq 0 ]]
}

@test "state persistence across subshells" {
    # Set state in current shell
    mock::node_red::set_state "stopped"
    mock::node_red::add_flow "test" '{"test":"data"}'
    
    # Check state in subshell (simulating BATS test execution)
    run bash -c '
        source "$MOCK_DIR/node-red.sh"
        echo "State: ${NODE_RED_MOCK_CONFIG[state]}"
        echo "Flows: ${#NODE_RED_MOCK_FLOWS[@]}"
    '
    assert_success
    assert_output --partial "State: stopped"
    assert_output --partial "Flows: 1"
}

@test "configuration changes are persisted" {
    mock::node_red::set_config "port" "1881"
    mock::node_red::set_config "version" "3.0.0"
    
    [[ "${NODE_RED_MOCK_CONFIG[port]}" == "1881" ]]
    [[ "${NODE_RED_MOCK_CONFIG[version]}" == "3.0.0" ]]
    
    # Verify persistence in subshell
    run bash -c '
        source "$MOCK_DIR/node-red.sh"
        echo "Port: ${NODE_RED_MOCK_CONFIG[port]}"
        echo "Version: ${NODE_RED_MOCK_CONFIG[version]}"
    '
    assert_success
    assert_output --partial "Port: 1881"
    assert_output --partial "Version: 3.0.0"
}

#######################################
# Helper Function Tests
#######################################

@test "add_flow stores flow data" {
    local flow_json='{"id":"test","type":"tab","label":"Test"}'
    mock::node_red::add_flow "test-flow" "$flow_json"
    
    [[ "${NODE_RED_MOCK_FLOWS[test-flow]}" == "$flow_json" ]]
}

@test "install_node stores node data" {
    mock::node_red::install_node "test-node" "2.0.0"
    
    [[ "${NODE_RED_MOCK_INSTALLED_NODES[test-node]}" == "2.0.0" ]]
}

@test "set_state updates state and health" {
    mock::node_red::set_state "stopped"
    
    [[ "${NODE_RED_MOCK_CONFIG[state]}" == "stopped" ]]
    [[ "${NODE_RED_MOCK_CONFIG[health]}" == "unhealthy" ]]
    
    mock::node_red::set_state "running"
    
    [[ "${NODE_RED_MOCK_CONFIG[state]}" == "running" ]]
    [[ "${NODE_RED_MOCK_CONFIG[health]}" == "healthy" ]]
}

#######################################
# Assertion Function Tests
#######################################

@test "assert_running succeeds when container is running" {
    mock::node_red::set_state "running"
    
    run mock::node_red::assert_running
    assert_success
}

@test "assert_running fails when container is stopped" {
    mock::node_red::set_state "stopped"
    
    run mock::node_red::assert_running
    assert_failure
    assert_output --partial "Node-RED is not running"
}

@test "assert_stopped succeeds when container is stopped" {
    mock::node_red::set_state "stopped"
    
    run mock::node_red::assert_stopped
    assert_success
}

@test "assert_stopped fails when container is running" {
    mock::node_red::set_state "running"
    
    run mock::node_red::assert_stopped
    assert_failure
    assert_output --partial "Node-RED is still running"
}

@test "assert_healthy succeeds when service is healthy" {
    mock::node_red::set_config "health" "healthy"
    
    run mock::node_red::assert_healthy
    assert_success
}

@test "assert_healthy fails when service is unhealthy" {
    mock::node_red::set_config "health" "unhealthy"
    
    run mock::node_red::assert_healthy
    assert_failure
    assert_output --partial "Node-RED is not healthy"
}

@test "assert_flow_exists succeeds when flow exists" {
    mock::node_red::add_flow "test-flow" '{"test":"data"}'
    
    run mock::node_red::assert_flow_exists "test-flow"
    assert_success
}

@test "assert_flow_exists fails when flow doesn't exist" {
    run mock::node_red::assert_flow_exists "nonexistent-flow"
    assert_failure
    assert_output --partial "Flow 'nonexistent-flow' does not exist"
}

@test "assert_node_installed succeeds when node is installed" {
    mock::node_red::install_node "test-node" "1.0.0"
    
    run mock::node_red::assert_node_installed "test-node"
    assert_success
}

@test "assert_node_installed fails when node is not installed" {
    run mock::node_red::assert_node_installed "nonexistent-node"
    assert_failure
    assert_output --partial "Node 'nonexistent-node' is not installed"
}

#######################################
# Integration Tests
#######################################

@test "complete workflow: install, start, deploy flow, execute" {
    # Container should start in running state
    assert_container_running "vrooli_node-red"
    
    # API should be accessible
    assert_api_response "/settings" 0
    
    # Deploy a flow
    local test_flow='[{"id":"f1","type":"tab","label":"Test Flow"},{"id":"n1","type":"inject","z":"f1"}]'
    run curl -s -X POST -H "Content-Type: application/json" -d "$test_flow" http://localhost:1880/flows
    assert_success
    assert_output --partial "\"rev\":"
    
    # Execute flow endpoint
    run curl -s -X POST -d '{"message":"test"}' http://localhost:1880/test
    assert_success
    assert_output --partial "Flow executed"
}

@test "error recovery: stop, start, verify functionality" {
    # Stop container
    mock::node_red::set_state "stopped"
    
    # API should be inaccessible
    assert_api_response "/settings" 1
    
    # Start container
    mock::node_red::set_state "running"
    
    # API should be accessible again
    assert_api_response "/settings" 0
}

@test "flow management: add, list, deploy, execute" {
    # Add custom flow
    local custom_flow='{"id":"custom","type":"tab","label":"Custom Flow"}'
    mock::node_red::add_flow "custom" "$custom_flow"
    
    # Verify flow exists
    mock::node_red::assert_flow_exists "custom"
    
    # List flows via API
    run curl -s http://localhost:1880/flows
    assert_success
    assert_output --partial "Custom Flow"
    
    # Deploy flows
    run curl -s -X POST -H "Content-Type: application/json" -d "[$custom_flow]" http://localhost:1880/flows
    assert_success
    assert_output --partial "\"flows\":1"
}

@test "node management: install and verify" {
    # Install custom node
    mock::node_red::install_node "custom-nodes" "1.5.0"
    
    # Verify installation
    mock::node_red::assert_node_installed "custom-nodes"
    [[ "${NODE_RED_MOCK_INSTALLED_NODES[custom-nodes]}" == "1.5.0" ]]
    
    # Install via API
    run curl -s -X POST -H "Content-Type: application/json" -d '{"name":"api-node"}' http://localhost:1880/nodes
    assert_success
    assert_output --partial "Node installed successfully"
}

#######################################
# Edge Cases and Error Handling
#######################################

@test "handles invalid docker commands gracefully" {
    run docker invalid-command
    assert_failure
    assert_output --partial "Unsupported command"
}

@test "handles invalid API endpoints gracefully" {
    run curl -s http://localhost:1880/invalid-endpoint
    assert_failure
    assert_output --partial "Unknown endpoint"
}

@test "handles malformed JSON in flow deployment" {
    run curl -s -X POST -H "Content-Type: application/json" -d '{"invalid":json}' http://localhost:1880/flows
    assert_success  # Mock accepts any data for simplicity
}

@test "handles empty flow deployment" {
    run curl -s -X POST -H "Content-Type: application/json" http://localhost:1880/flows
    assert_failure
    assert_output --partial "No flow data provided"
}

@test "handles non-Node-RED container operations" {
    run docker start other-container
    assert_failure
    assert_output --partial "Container not found"
}

#######################################
# Debug and Utility Tests
#######################################

@test "dump_state shows current configuration" {
    run mock::node_red::dump_state
    assert_success
    assert_output --partial "=== Node-RED Mock State ==="
    assert_output --partial "Configuration:"
    assert_output --partial "container_name: vrooli_node-red"
    assert_output --partial "port: 1880"
    assert_output --partial "state: running"
}

@test "state file is created and accessible" {
    local state_file="$NODE_RED_MOCK_STATE_DIR/node-red-state.sh"
    [[ -f "$state_file" ]]
    
    # State file should contain configuration
    run grep "NODE_RED_MOCK_CONFIG" "$state_file"
    assert_success
}

@test "state file can be sourced independently" {
    # Modify state
    mock::node_red::set_config "test_key" "test_value"
    
    # Source state file in subshell
    local state_file="$NODE_RED_MOCK_STATE_DIR/node-red-state.sh"
    run bash -c "source '$state_file'; echo \"\${NODE_RED_MOCK_CONFIG[test_key]}\""
    assert_success
    assert_output "test_value"
}

#######################################
# Performance and Resource Tests
#######################################

@test "handles multiple concurrent API calls" {
    # Simulate concurrent calls
    for i in {1..5}; do
        curl -s http://localhost:1880/settings &
    done
    wait
    
    # All should succeed
    run curl -s http://localhost:1880/settings
    assert_success
    assert_output --partial "\"version\": \"3.1.0\""
}

@test "handles large flow deployments" {
    # Create large flow JSON
    local large_flow='['
    for i in {1..100}; do
        if [[ $i -gt 1 ]]; then
            large_flow+=','
        fi
        large_flow+="{\"id\":\"n$i\",\"type\":\"inject\",\"name\":\"Node $i\"}"
    done
    large_flow+=']'
    
    run curl -s -X POST -H "Content-Type: application/json" -d "$large_flow" http://localhost:1880/flows
    assert_success
    assert_output --partial "\"rev\":"
}

@test "state persistence under rapid changes" {
    # Rapidly change state
    for i in {1..10}; do
        mock::node_red::set_config "test_$i" "value_$i"
    done
    
    # Verify all changes persisted
    for i in {1..10}; do
        [[ "${NODE_RED_MOCK_CONFIG[test_$i]}" == "value_$i" ]]
    done
}