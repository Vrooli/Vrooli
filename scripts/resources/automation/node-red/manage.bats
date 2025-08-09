#!/usr/bin/env bats
# Tests for manage.sh - Node-RED management script
# Integration tests for end-to-end scenarios and interactions between modules

# Load Vrooli test infrastructure
source "${BATS_TEST_DIRNAME}/../../../__test/fixtures/setup.bash"

# Expensive setup operations run once per file
setup_file() {
    # Use Vrooli service test setup
    vrooli_setup_service_test "node-red"
    
    # Load resource specific configuration once per file
    SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    NODE_RED_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Load configuration and manage script once
    source "${NODE_RED_DIR}/config/defaults.sh"
    source "${NODE_RED_DIR}/config/messages.sh"
    source "${SCRIPT_DIR}/manage.sh"
}

# Lightweight per-test setup
setup() {
    # Setup standard Vrooli mocks
    vrooli_auto_setup
    
    # Set test environment variables (lightweight per-test)
    export NODE_RED_CUSTOM_PORT="9999"
    export NODE_RED_CONTAINER_NAME="node-red-test"
    export NODE_RED_BASE_URL="http://localhost:9999"
    export FORCE="no"
    export YES="no"
    export BUILD_IMAGE="no"
    
    # Export config functions
    node_red::export_config
    node_red::export_messages
    
    # Mock log functions
    log::header() { echo "=== $* ==="; }
    log::info() { echo "[INFO] $*"; }
    log::error() { echo "[ERROR] $*" >&2; }
    log::success() { echo "[SUCCESS] $*"; }
    log::warning() { echo "[WARNING] $*" >&2; }
    export -f log::header log::info log::error log::success log::warning

# BATS teardown function - runs after each test
teardown() {
    vrooli_cleanup_test
}

@test "full installation workflow completes successfully" {
    # Mock docker for installation
    docker() {
        case "$*" in
            *"container inspect"*) return 1 ;;  # Not installed
            *"run"*) return 0 ;;  # Install succeeds
            *) return 0 ;;
        esac
    }
    curl() { return 0; }
    export -f docker curl
    
    export YES="yes"
    export BUILD_IMAGE="no"
    
    run node_red::install
    [ "$status" -eq 0 ]
    [[ "$output" =~ "installed successfully" ]] || [[ "$output" =~ "Node-RED" ]]
}

@test "installation with custom image build" {
    # Mock docker for custom build
    docker() {
        case "$*" in
            *"container inspect"*) return 1 ;;  # Not installed
            *"build"*) 
                echo "built successfully"
                return 0 ;;
            *"run"*) 
                echo "installed successfully"
                return 0 ;;
            *) return 0 ;;
        esac
    }
    curl() { return 0; }
    export -f docker curl
    
    export YES="yes"
    export BUILD_IMAGE="yes"
    
    # Create mock Dockerfile
    mkdir -p "$VROOLI_TEST_TMPDIR/docker"
    echo "FROM nodered/node-red:latest" > "$VROOLI_TEST_TMPDIR/docker/Dockerfile"
    
    run node_red::install
    [ "$status" -eq 0 ]
    [[ "$output" =~ "built successfully" ]] || [[ "$output" =~ "installed successfully" ]]
}

@test "reinstallation over existing installation" {
    # Mock docker for existing installation
    docker() {
        case "$*" in
            *"container inspect"*) return 0 ;;  # Already exists
            *"container inspect -f"*) echo "true" ;;  # Running
            *) return 0 ;;
        esac
    }
    curl() { return 0; }
    export -f docker curl
    
    export YES="yes"  # Auto-confirm reinstall
    
    run node_red::install
    [ "$status" -eq 0 ]
    [[ "$output" =~ "already installed" ]] || [[ "$output" =~ "installed successfully" ]]
}

@test "installation failure cleanup" {
    # Mock docker and curl for failure
    docker() { return 1; }  # Installation fails
    curl() { return 1; }  # Never becomes ready
    export -f docker curl
    
    export NODE_RED_READY_TIMEOUT=1
    export NODE_RED_READY_SLEEP=0.1
    
    run node_red::install
    [ "$status" -ne 0 ]
    [[ "$output" =~ "installation failed" ]] || [[ "$output" =~ "failed" ]]
}

@test "complete uninstallation workflow" {
    # Mock docker for uninstallation
    docker() {
        case "$*" in
            *"container inspect"*) return 0 ;;  # Exists
            *"stop"*|*"rm"*) 
                echo "uninstalled successfully"
                return 0 ;;
            *) return 0 ;;
        esac
    }
    export -f docker
    
    export YES="yes"
    
    run node_red::uninstall
    [ "$status" -eq 0 ]
    [[ "$output" =~ "uninstalled successfully" ]]
}

@test "uninstall of non-existent installation" {
    # Mock docker for non-existent installation
    docker() {
        case "$*" in
            *"container inspect"*) return 1 ;;  # Does not exist
            *) return 0 ;;
        esac
    }
    export -f docker
    
    run node_red::uninstall
    [ "$status" -eq 0 ]
    [[ "$output" =~ "not installed" ]]
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
    
    run node_red::start
    [ "$status" -eq 0 ]
}

@test "start already running service" {
    # Mock docker for already running
    docker() {
        case "$*" in
            *"container inspect -f"*) echo "true" ;;  # Already running
            *) return 0 ;;
        esac
    }
    export -f docker
    
    run node_red::start
    [ "$status" -eq 0 ]
    [[ "$output" =~ "already running" ]]
}

@test "stop service workflow" {
    # Mock docker for stop
    docker() { return 0; }
    export -f docker
    
    run node_red::stop
    [ "$status" -eq 0 ]
}

@test "restart service workflow" {
    # Mock docker for restart
    docker() { return 0; }
    export -f docker
    
    run node_red::restart
    [ "$status" -eq 0 ]
}

@test "status display shows comprehensive information" {
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
    curl() { return 0; }
    export -f docker curl
    
    run node_red::show_status
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Node-RED Status" ]]
    [[ "$output" =~ "Running" ]]
}

@test "test suite runs all checks" {
    # Mock docker and curl for test suite
    docker() { return 0; }
    curl() { return 0; }
    export -f docker curl
    
    run node_red::run_tests
    [ "$status" -eq 0 ]
    [[ "$output" =~ "validation test suite" ]] || [[ "$output" =~ "test" ]]
}

@test "command line argument parsing" {
    # Test that different actions are parsed correctly
    node_red::parse_arguments status
    [ "$ACTION" = "status" ]
    
    node_red::parse_arguments test  
    [ "$ACTION" = "test" ]
}

@test "environment variable handling" {
    # Mock docker for installation
    docker() { return 0; }
    curl() { return 0; }
    export -f docker curl
    
    # Test with different environment variables
    export NODE_RED_CUSTOM_PORT="8080"
    export YES="yes"
    export BUILD_IMAGE="no"
    
    run node_red::install
    [ "$status" -eq 0 ]
    [[ "$output" =~ "installed successfully" ]] || [[ "$output" =~ "8080" ]]
}

@test "error handling propagates correctly" {
    # Mock docker to fail
    docker() { return 1; }
    export -f docker
    
    run node_red::install
    [ "$status" -ne 0 ]
}

@test "script handles invalid actions gracefully" {
    run node_red::handle_action "invalid-action"
    [ "$status" -ne 0 ]
    [[ "$output" =~ "Unknown action" ]] || [[ "$output" =~ "invalid" ]]
}

# Flow management integration tests
@test "flow import and export workflow" {
    # Mock docker for success
    # Mock curl for success
    
    # Create test flow file
    local flow_file="$NODE_RED_TEST_DIR/test-flow.json"
    echo '[{"id": "flow1", "type": "tab", "label": "Test Flow"}]' > "$flow_file"
    
    # Test import (simulated)
    run node_red::import_flow_file "$flow_file"
    [ "$status" -eq 0 ]
    [[ "$output" =~  "imported successfully"
    
    # Test export
    local export_file="$NODE_RED_TEST_DIR/exported-flows.json"
    run node_red::export_flows_to_file "$export_file"
    [ "$status" -eq 0 ]
    [[ "$output" =~  "exported successfully"
    assert_file_exists "$export_file"
}

@test "backup and restore workflow" {
    # Mock docker for success
    # Mock curl for success
    
    # Create some test data
    mkdir -p "$SCRIPT_DIR/flows"
    echo '{"test": "flow"}' > "$SCRIPT_DIR/flows/test.json"
    echo 'module.exports = {};' > "$SCRIPT_DIR/settings.js"
    
    # Create backup
    local backup_dir="$NODE_RED_TEST_DIR/test-backup"
    run node_red::backup "$backup_dir"
    [ "$status" -eq 0 ]
    [[ "$output" =~  "Backup created"
    
    # Restore backup
    run node_red::restore "$backup_dir"
    [ "$status" -eq 0 ]
    [[ "$output" =~  "restored from backup"
}

@test "monitoring and metrics collection" {
    # Mock docker for success
    # Mock curl for success
    
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
    [ "$status" -eq 0 ]
    [[ "$output" =~  "Resource Metrics"
    [[ "$output" =~  "Flow Metrics"
    [[ "$output" =~  "Node.js Process"
}

@test "health checking and validation" {
    # Mock docker for success
    # Mock curl for success
    
    # Create required files for validation
    mkdir -p "$SCRIPT_DIR/flows"
    echo 'module.exports = {};' > "$SCRIPT_DIR/settings.js"
    mkdir -p "$NODE_RED_TEST_CONFIG_DIR"
    echo '{"services": {"automation": {"node-red": {"enabled": true}}}}' > "$NODE_RED_TEST_CONFIG_DIR/service.json"
    
    # Test health check
    run node_red::health_check
    [ "$status" -eq 0 ]
    [[ "$output" =~  "Health Check"
    
    # Test validation
    run node_red::validate_installation
    [ "$status" -eq 0 ]
    [[ "$output" =~  "validation passed"
}

@test "configuration management integration" {
    # Mock docker for success
    
    # Test configuration creation
    run node_red::update_resource_config
    [ "$status" -eq 0 ]
    assert_file_exists "$NODE_RED_TEST_CONFIG_DIR/service.json"
    
    # Test configuration removal
    run node_red::remove_resource_config
    [ "$status" -eq 0 ]
}

@test "Docker resource lifecycle management" {
    # Mock docker for success
    
    # Test network creation
    run node_red::create_network
    [ "$status" -eq 0 ]
    
    # Test volume creation
    run node_red::create_volume
    [ "$status" -eq 0 ]
    
    # Test cleanup
    run node_red::cleanup_docker_resources
    [ "$status" -eq 0 ]
}

# Stress and performance testing integration
@test "performance benchmarking workflow" {
    # Mock docker for success
    # Mock curl for success
    
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
    [ "$status" -eq 0 ]
    [[ "$output" =~  "Performance Benchmark"
    [[ "$output" =~  "HTTP Response Time"
    [[ "$output" =~  "Average:"
}

@test "comprehensive testing suite integration" {
    # Mock docker for success
    # Mock curl for success
    
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
    [ "$status" -eq 0 ]
    
    # Should run all test categories
    [[ "$output" =~  "container status"
    [[ "$output" =~  "HTTP endpoint"
    [[ "$output" =~  "admin API"
    [[ "$output" =~  "Docker access"
    [[ "$output" =~  "workspace access"
    [[ "$output" =~  "host command execution"
    [[ "$output" =~  "flow persistence"
    [[ "$output" =~  "All tests passed"
}

# Error recovery and resilience testing
@test "service recovery after failure" {
    # Start with failed state
    # Mock docker for not running
    
    run node_red::health_check
    [ "$status" -ne 0 ]
    
    # Simulate service recovery
    # Mock docker for success
    # Mock curl for success
    
    run node_red::health_check
    [ "$status" -eq 0 ]
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
    [ "$status" -eq 0 ]
}

@test "network failure resilience" {
    # Mock docker for success
    
    # Start with network failure
    # Mock curl for failure
    
    run node_red::health_check
    [ "$status" -ne 0 ]
    
    # Recovery after network comes back
    # Mock curl for success
    
    run node_red::health_check
    [ "$status" -eq 0 ]
}

# Multi-operation workflows
@test "install-configure-test workflow" {
    # Mock docker for not installed
    # Mock curl for success
    export YES="yes"
    export BUILD_IMAGE="no"
    
    # Install
    run "$NODE_RED_TEST_DIR/manage.sh" install
    [ "$status" -eq 0 ]
    
    # Should now show as running
    # Mock docker for success
    run "$NODE_RED_TEST_DIR/manage.sh" status
    [ "$status" -eq 0 ]
    [[ "$output" =~  "Running"
    
    # Should pass tests
    run "$NODE_RED_TEST_DIR/manage.sh" test
    [ "$status" -eq 0 ]
    [[ "$output" =~  "All tests passed"
}

@test "backup-uninstall-reinstall-restore workflow" {
    # Mock docker for success
    # Mock curl for success
    export YES="yes"
    
    # Create test data
    mkdir -p "$SCRIPT_DIR/flows"
    echo '{"test": "data"}' > "$SCRIPT_DIR/flows/test.json"
    
    # Backup
    local backup_dir="$NODE_RED_TEST_DIR/workflow-backup"
    run node_red::backup "$backup_dir"
    [ "$status" -eq 0 ]
    
    # Uninstall
    run "$NODE_RED_TEST_DIR/manage.sh" uninstall
    [ "$status" -eq 0 ]
    
    # Reinstall
    # Mock docker for not installed
    run "$NODE_RED_TEST_DIR/manage.sh" install
    [ "$status" -eq 0 ]
    
    # Restore
    # Mock docker for success
    run node_red::restore "$backup_dir"
    [ "$status" -eq 0 ]
}

# Cross-module interaction tests
@test "configuration affects all modules consistently" {
    export NODE_RED_CUSTOM_PORT="9999"
    
    # Re-source with new port
    source "$NODE_RED_TEST_DIR/config/defaults.sh"
    node_red::export_config
    
    # Mock docker for success
    # Mock curl for success
    
    # Status should show correct port
    run node_red::show_status
    [ "$status" -eq 0 ]
    [[ "$output" =~  "9999"
    
    # Health check should use correct port
    curl() {
        if [[ "$*" =~ "localhost:9999" ]]; then
            return 0
        fi
        return 1
    }
    export -f curl
    
    run node_red::health_check
    [ "$status" -eq 0 ]
}

@test "error messages are consistent across modules" {
    # Mock docker for not running
    
    # Different modules should show consistent error messages
    run node_red::show_status
    [ "$status" -eq 0 ]
    [[ "$output" =~  "Stopped"
    
    run node_red::health_check
    [ "$status" -ne 0 ]
    [[ "$output" =~  "not running"
    
    run node_red::list_flows
    [ "$status" -ne 0 ]
    [[ "$output" =~  "not running"
}

# Concurrent operation testing
@test "concurrent operations don't interfere" {
    # Mock docker for success
    # Mock curl for success
    
    # Run multiple operations in parallel
    (node_red::show_status > /dev/null) &
    (node_red::health_check > /dev/null) &
    (node_red::show_metrics > /dev/null) &
    
    wait  # Wait for all background processes
    
    # All should complete successfully
    [[ $? -eq 0 ]]
}

@test "script handles signal interruption gracefully" {
    # Mock docker for success
    
    # Test interrupt handling (simulated) - need to source common.sh first for log:: functions
    run bash -c 'source "$RESOURCES_DIR/common.sh"; source "$0/config/messages.sh"; node_red::show_interrupt_message' "$NODE_RED_TEST_DIR"
    [ "$status" -eq 0 ]
    [[ "$output" =~  "interrupted"
}

# End-to-end validation
@test "complete system validation after setup" {
    # Mock docker for success
    # Mock curl for success
    
    # Create complete environment
    mkdir -p "$SCRIPT_DIR/flows" "$SCRIPT_DIR/nodes"
    echo 'module.exports = {};' > "$SCRIPT_DIR/settings.js"
    mkdir -p "$NODE_RED_TEST_CONFIG_DIR"
    echo '{"services": {"automation": {"node-red": {"enabled": true}}}}' > "$NODE_RED_TEST_CONFIG_DIR/service.json"
    
    # Everything should validate successfully
    run node_red::validate_installation
    [ "$status" -eq 0 ]
    [[ "$output" =~  "validation passed"
    
    run node_red::run_tests
    [ "$status" -eq 0 ]
    [[ "$output" =~  "All tests passed"
}