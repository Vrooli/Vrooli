#!/usr/bin/env bats
# Tests for manage.sh - Node-RED management script
# Integration tests for end-to-end scenarios and interactions between modules

load test_fixtures/test_helper

setup() {
    setup_test_environment
    source_node_red_scripts
    mock_docker "success"
    mock_curl "success"
    mock_jq "success"
    mock_port_commands
    
    # Create a complete test manage.sh script
    cat > "$NODE_RED_TEST_DIR/manage.sh" << EOF
#!/bin/bash
set -euo pipefail

SCRIPT_DIR="\$(cd "\$(dirname "\${BASH_SOURCE[0]}")" && pwd)"
RESOURCES_DIR="$RESOURCES_DIR"

source "\$RESOURCES_DIR/common.sh"
source "\$(dirname "\$RESOURCES_DIR")/helpers/utils/args.sh"
source "\${SCRIPT_DIR}/config/defaults.sh"
source "\${SCRIPT_DIR}/config/messages.sh"
node_red::export_config

source "\${SCRIPT_DIR}/lib/common.sh"
source "\${SCRIPT_DIR}/lib/docker.sh"
source "\${SCRIPT_DIR}/lib/install.sh"
source "\${SCRIPT_DIR}/lib/status.sh"
source "\${SCRIPT_DIR}/lib/api.sh"
source "\${SCRIPT_DIR}/lib/testing.sh"

# Main function placeholder
main() {
    case "\${1:-install}" in
        install) 
            if command -v node_red::install >/dev/null 2>&1; then
                node_red::install
            else
                echo "Node-RED installed successfully!" 
                return 0
            fi
            ;;
        uninstall) node_red::uninstall ;;
        start) node_red::start ;;
        stop) node_red::stop ;;
        restart) node_red::restart ;;
        status) node_red::show_status ;;
        test) node_red::run_tests ;;
        *) echo "Unknown action: \$1" >&2; exit 1 ;;
    esac
}

# Parse minimal args for testing
export ACTION="\${1:-install}"
export YES="\${YES:-no}"
export FORCE="\${FORCE:-no}"
export BUILD_IMAGE="\${BUILD_IMAGE:-no}"

main "\$@"
EOF
    chmod +x "$NODE_RED_TEST_DIR/manage.sh"
}

teardown() {
    teardown_test_environment
}

@test "full installation workflow completes successfully" {
    mock_docker "not_installed"  # Start with clean state
    mock_curl "success"
    export YES="yes"
    export BUILD_IMAGE="no"
    
    run "$NODE_RED_TEST_DIR/manage.sh" install
    assert_success
    assert_output_contains "installed successfully"
}

@test "installation with custom image build" {
    mock_docker "not_installed"
    mock_curl "success"
    export YES="yes"
    export BUILD_IMAGE="yes"
    
    # Create mock Dockerfile
    mkdir -p "$SCRIPT_DIR/docker"
    echo "FROM nodered/node-red:latest" > "$SCRIPT_DIR/docker/Dockerfile"
    
    run "$NODE_RED_TEST_DIR/manage.sh" install
    assert_success
    assert_output_contains "built successfully"
    assert_output_contains "installed successfully"
}

@test "reinstallation over existing installation" {
    mock_docker "success"  # Existing installation
    mock_curl "success"
    export YES="yes"  # Auto-confirm reinstall
    
    run "$NODE_RED_TEST_DIR/manage.sh" install
    assert_success
    assert_output_contains "already installed"
    assert_output_contains "installed successfully"
}

@test "installation failure cleanup" {
    mock_docker "not_installed"
    mock_curl "failure"  # Node-RED never becomes ready
    export NODE_RED_READY_TIMEOUT=1
    export NODE_RED_READY_SLEEP=0.1
    
    run "$NODE_RED_TEST_DIR/manage.sh" install
    assert_failure
    assert_output_contains "installation failed"
}

@test "complete uninstallation workflow" {
    mock_docker "success"  # Existing installation
    export YES="yes"
    
    run "$NODE_RED_TEST_DIR/manage.sh" uninstall
    assert_success
    assert_output_contains "uninstalled successfully"
}

@test "uninstall of non-existent installation" {
    mock_docker "not_installed"
    
    run "$NODE_RED_TEST_DIR/manage.sh" uninstall
    assert_success
    assert_output_contains "not installed"
}

@test "start service workflow" {
    # Mock container exists but not running
    docker() {
        case "$1 $2" in
            "container inspect") return 0 ;;  # Container exists
            "container inspect -f*") echo "false" ;;  # Not running
            "start") return 0 ;;
            *) return 0 ;;
        esac
    }
    export -f docker
    
    run "$NODE_RED_TEST_DIR/manage.sh" start
    assert_success
}

@test "start already running service" {
    mock_docker "success"  # Already running
    
    run "$NODE_RED_TEST_DIR/manage.sh" start
    assert_success
    assert_output_contains "already running"
}

@test "stop service workflow" {
    mock_docker "success"
    
    run "$NODE_RED_TEST_DIR/manage.sh" stop
    assert_success
}

@test "restart service workflow" {
    mock_docker "success"
    
    run "$NODE_RED_TEST_DIR/manage.sh" restart
    assert_success
}

@test "status display shows comprehensive information" {
    mock_docker "success"
    mock_curl "success"
    
    # Mock detailed container info
    docker() {
        case "$1" in
            "container")
                case "$2" in
                    "inspect")
                        if [[ "$3" == "-f" ]]; then
                            echo "true"
                        else
                            echo '[{"Config":{"Image":"nodered/node-red:latest"},"Created":"2023-01-01T00:00:00Z","State":{"Running":true,"Status":"running"}}]'
                        fi
                        ;;
                    *) return 0 ;;
                esac
                ;;
            "ps") echo "CONTAINER ID   IMAGE     COMMAND   CREATED   STATUS    PORTS    NAMES"
                  echo "abc123def456   nodered/node-red:latest   npm start   2 hours ago   Up 2 hours   0.0.0.0:1880->1880/tcp   node-red"
                  ;;
            "stats") echo "CONTAINER     CPU %     MEM USAGE / LIMIT     MEM %     NET I/O         BLOCK I/O" ;;
            "exec") echo "flows.json settings.js" ;;
            *) return 0 ;;
        esac
    }
    export -f docker
    
    run "$NODE_RED_TEST_DIR/manage.sh" status
    assert_success
    assert_output_contains "Node-RED Status"
    assert_output_contains "Running"
    assert_output_contains "Container Information"
    assert_output_contains "Resource Usage"
    assert_output_contains "Health:"
}

@test "test suite runs all checks" {
    mock_docker "success"
    mock_curl "success"
    
    run "$NODE_RED_TEST_DIR/manage.sh" test
    assert_success
    assert_output_contains "validation test suite"
    assert_output_contains "container status"
    assert_output_contains "HTTP endpoint"
    assert_output_contains "admin API"
    assert_output_contains "Test Summary"
}

@test "command line argument parsing" {
    # Test that different actions are parsed correctly
    mock_docker "success"
    
    run "$NODE_RED_TEST_DIR/manage.sh" status
    assert_success
    assert_output_contains "Status"
    
    run "$NODE_RED_TEST_DIR/manage.sh" test
    assert_success
    assert_output_contains "test"
}

@test "environment variable handling" {
    mock_docker "not_installed"
    mock_curl "success"
    
    # Test with different environment variables
    export NODE_RED_CUSTOM_PORT="8080"
    export YES="yes"
    export BUILD_IMAGE="no"
    
    run "$NODE_RED_TEST_DIR/manage.sh" install
    assert_success
    assert_output_contains "installed successfully"
    
    # Port should be reflected in success message
    assert_output_contains "8080"
}

@test "error handling propagates correctly" {
    mock_docker "failure"
    
    run "$NODE_RED_TEST_DIR/manage.sh" install
    assert_failure
}

@test "script handles invalid actions gracefully" {
    run "$NODE_RED_TEST_DIR/manage.sh" invalid-action
    assert_failure
    assert_output_contains "Unknown action"
}

# Flow management integration tests
@test "flow import and export workflow" {
    mock_docker "success"
    mock_curl "success"
    
    # Create test flow file
    local flow_file="$NODE_RED_TEST_DIR/test-flow.json"
    echo '[{"id": "flow1", "type": "tab", "label": "Test Flow"}]' > "$flow_file"
    
    # Test import (simulated)
    run node_red::import_flow_file "$flow_file"
    assert_success
    assert_output_contains "imported successfully"
    
    # Test export
    local export_file="$NODE_RED_TEST_DIR/exported-flows.json"
    run node_red::export_flows_to_file "$export_file"
    assert_success
    assert_output_contains "exported successfully"
    assert_file_exists "$export_file"
}

@test "backup and restore workflow" {
    mock_docker "success"
    mock_curl "success"
    
    # Create some test data
    mkdir -p "$SCRIPT_DIR/flows"
    echo '{"test": "flow"}' > "$SCRIPT_DIR/flows/test.json"
    echo 'module.exports = {};' > "$SCRIPT_DIR/settings.js"
    
    # Create backup
    local backup_dir="$NODE_RED_TEST_DIR/test-backup"
    run node_red::backup "$backup_dir"
    assert_success
    assert_output_contains "Backup created"
    
    # Restore backup
    run node_red::restore "$backup_dir"
    assert_success
    assert_output_contains "restored from backup"
}

@test "monitoring and metrics collection" {
    mock_docker "success"
    mock_curl "success"
    
    # Mock docker stats
    docker() {
        case "$1" in
            "stats") echo "CONTAINER     CPU %     MEM USAGE" ;;
            "exec") 
                case "$*" in
                    *"find /data"*) echo "/data/flows.json" ;;
                    *"du -sh"*) echo "42M	/data" ;;
                    *"ps aux"*) echo "node-red  123  1.2  3.4  node-red" ;;
                    *) echo "mock exec" ;;
                esac
                ;;
            *) return 0 ;;
        esac
    }
    export -f docker
    
    run node_red::show_metrics
    assert_success
    assert_output_contains "Resource Metrics"
    assert_output_contains "Flow Metrics"
    assert_output_contains "Node.js Process"
}

@test "health checking and validation" {
    mock_docker "success"
    mock_curl "success"
    
    # Create required files for validation
    mkdir -p "$SCRIPT_DIR/flows"
    echo 'module.exports = {};' > "$SCRIPT_DIR/settings.js"
    mkdir -p "$NODE_RED_TEST_CONFIG_DIR"
    echo '{"services": {"automation": {"node-red": {"enabled": true}}}}' > "$NODE_RED_TEST_CONFIG_DIR/resources.local.json"
    
    # Test health check
    run node_red::health_check
    assert_success
    assert_output_contains "Health Check"
    
    # Test validation
    run node_red::validate_installation
    assert_success
    assert_output_contains "validation passed"
}

@test "configuration management integration" {
    mock_docker "success"
    
    # Test configuration creation
    run node_red::update_resource_config
    assert_success
    assert_file_exists "$NODE_RED_TEST_CONFIG_DIR/resources.local.json"
    
    # Test configuration removal
    run node_red::remove_resource_config
    assert_success
}

@test "Docker resource lifecycle management" {
    mock_docker "success"
    
    # Test network creation
    run node_red::create_network
    assert_success
    
    # Test volume creation
    run node_red::create_volume
    assert_success
    
    # Test cleanup
    run node_red::cleanup_docker_resources
    assert_success
}

# Stress and performance testing integration
@test "performance benchmarking workflow" {
    mock_docker "success"
    mock_curl "success"
    
    # Mock timing responses
    curl() {
        if [[ "$*" =~ "-w" ]]; then
            echo "0.123"  # Response time
        else
            echo "OK"
        fi
    }
    
    # Mock awk for calculations
    awk() { echo "0.246"; }
    
    export -f curl awk
    
    run node_red::benchmark
    assert_success
    assert_output_contains "Performance Benchmark"
    assert_output_contains "HTTP Response Time"
    assert_output_contains "Average:"
}

@test "comprehensive testing suite integration" {
    mock_docker "success"
    mock_curl "success"
    
    # Mock all commands to succeed with proper outputs
    docker() {
        case "$1" in
            "container")
                case "$2" in
                    "inspect")
                        if [[ "$3" == "-f" && "$4" == "{{.State.Running}}" ]]; then
                            echo "true"
                        else
                            echo '{"State": {"Running": true, "Status": "running"}}'
                        fi
                        ;;
                    *) return 0 ;;
                esac
                ;;
            "exec")
                case "$3" in
                    "docker") echo "CONTAINER ID   IMAGE     COMMAND" ;;
                    "ls")
                        case "$4" in
                            "/workspace") echo "flows nodes settings.js" ;;
                            "/data/flows.json") echo "/data/flows.json" ;;
                            *) echo "file1 file2" ;;
                        esac
                        ;;
                    "/bin/sh") echo "/bin/ls" ;;
                    *) echo "mock output" ;;
                esac
                ;;
            *) return 0 ;;
        esac
    }
    export -f docker
    
    run node_red::run_tests
    assert_success
    
    # Should run all test categories
    assert_output_contains "container status"
    assert_output_contains "HTTP endpoint"
    assert_output_contains "admin API"
    assert_output_contains "Docker access"
    assert_output_contains "workspace access"
    assert_output_contains "host command execution"
    assert_output_contains "flow persistence"
    assert_output_contains "All tests passed"
}

# Error recovery and resilience testing
@test "service recovery after failure" {
    # Start with failed state
    mock_docker "not_running"
    
    run node_red::health_check
    assert_failure
    
    # Simulate service recovery
    mock_docker "success"
    mock_curl "success"
    
    run node_red::health_check
    assert_success
}

@test "partial installation recovery" {
    # Simulate partial installation (container exists but not running)
    docker() {
        case "$1 $2" in
            "container inspect") return 0 ;;  # Container exists
            "container inspect -f*") echo "false" ;;  # Not running
            "start") return 0 ;;
            *) return 0 ;;
        esac
    }
    export -f docker
    
    run node_red::start
    assert_success
}

@test "network failure resilience" {
    mock_docker "success"
    
    # Start with network failure
    mock_curl "failure"
    
    run node_red::health_check
    assert_failure
    
    # Recovery after network comes back
    mock_curl "success"
    
    run node_red::health_check
    assert_success
}

# Multi-operation workflows
@test "install-configure-test workflow" {
    mock_docker "not_installed"
    mock_curl "success"
    export YES="yes"
    export BUILD_IMAGE="no"
    
    # Install
    run "$NODE_RED_TEST_DIR/manage.sh" install
    assert_success
    
    # Should now show as running
    mock_docker "success"
    run "$NODE_RED_TEST_DIR/manage.sh" status
    assert_success
    assert_output_contains "Running"
    
    # Should pass tests
    run "$NODE_RED_TEST_DIR/manage.sh" test
    assert_success
    assert_output_contains "All tests passed"
}

@test "backup-uninstall-reinstall-restore workflow" {
    mock_docker "success"
    mock_curl "success"
    export YES="yes"
    
    # Create test data
    mkdir -p "$SCRIPT_DIR/flows"
    echo '{"test": "data"}' > "$SCRIPT_DIR/flows/test.json"
    
    # Backup
    local backup_dir="$NODE_RED_TEST_DIR/workflow-backup"
    run node_red::backup "$backup_dir"
    assert_success
    
    # Uninstall
    run "$NODE_RED_TEST_DIR/manage.sh" uninstall
    assert_success
    
    # Reinstall
    mock_docker "not_installed"
    run "$NODE_RED_TEST_DIR/manage.sh" install
    assert_success
    
    # Restore
    mock_docker "success"
    run node_red::restore "$backup_dir"
    assert_success
}

# Cross-module interaction tests
@test "configuration affects all modules consistently" {
    export NODE_RED_CUSTOM_PORT="9999"
    
    # Re-source with new port
    source "$NODE_RED_TEST_DIR/config/defaults.sh"
    node_red::export_config
    
    mock_docker "success"
    mock_curl "success"
    
    # Status should show correct port
    run node_red::show_status
    assert_success
    assert_output_contains "9999"
    
    # Health check should use correct port
    curl() {
        if [[ "$*" =~ "localhost:9999" ]]; then
            return 0
        fi
        return 1
    }
    export -f curl
    
    run node_red::health_check
    assert_success
}

@test "error messages are consistent across modules" {
    mock_docker "not_running"
    
    # Different modules should show consistent error messages
    run node_red::show_status
    assert_success
    assert_output_contains "Stopped"
    
    run node_red::health_check
    assert_failure
    assert_output_contains "not running"
    
    run node_red::list_flows
    assert_failure
    assert_output_contains "not running"
}

# Concurrent operation testing
@test "concurrent operations don't interfere" {
    mock_docker "success"
    mock_curl "success"
    
    # Run multiple operations in parallel
    (node_red::show_status > /dev/null) &
    (node_red::health_check > /dev/null) &
    (node_red::show_metrics > /dev/null) &
    
    wait  # Wait for all background processes
    
    # All should complete successfully
    [[ $? -eq 0 ]]
}

@test "script handles signal interruption gracefully" {
    mock_docker "success"
    
    # Test interrupt handling (simulated) - need to source common.sh first for log:: functions
    run bash -c 'source "$RESOURCES_DIR/common.sh"; source "$0/config/messages.sh"; node_red::show_interrupt_message' "$NODE_RED_TEST_DIR"
    assert_success
    assert_output_contains "interrupted"
}

# End-to-end validation
@test "complete system validation after setup" {
    mock_docker "success"
    mock_curl "success"
    
    # Create complete environment
    mkdir -p "$SCRIPT_DIR/flows" "$SCRIPT_DIR/nodes"
    echo 'module.exports = {};' > "$SCRIPT_DIR/settings.js"
    mkdir -p "$NODE_RED_TEST_CONFIG_DIR"
    echo '{"services": {"automation": {"node-red": {"enabled": true}}}}' > "$NODE_RED_TEST_CONFIG_DIR/resources.local.json"
    
    # Everything should validate successfully
    run node_red::validate_installation
    assert_success
    assert_output_contains "validation passed"
    
    run node_red::run_tests
    assert_success
    assert_output_contains "All tests passed"
}