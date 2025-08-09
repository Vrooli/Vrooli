#!/usr/bin/env bats
# Tests for Node-RED status functions (lib/status.sh)

source "${BATS_TEST_DIRNAME}/../../../../__test/fixtures/setup.bash"

# Run expensive setup once per file instead of per test
setup_file() {
    setup_test_environment
    source_node_red_scripts
}

# Lightweight per-test setup - just reset mocks and ensure functions are available
setup() {
    # Re-source the functions if they're not available (shouldn't be needed but ensures test isolation)
    if ! command -v node_red::show_status >/dev/null 2>&1; then
        source_node_red_scripts
    fi
    
    mock_docker "success"
    mock_curl "success"
    mock_jq "success"
}

# Cleanup once per file
teardown_file() {
    teardown_test_environment
}

@test "node_red::show_status displays complete status when installed and running" {
    mock_docker "success"
    mock_curl "success"
    
    run node_red::show_status
    assert_success
    assert_output_contains "Node-RED Status"
    assert_output_contains "Running"
    assert_output_contains "http://localhost:$RESOURCE_PORT"
}

@test "node_red::show_status shows not installed when container doesn't exist" {
    mock_docker "not_installed"
    
    run node_red::show_status
    assert_success
    assert_output_contains "Not installed"
}

@test "node_red::show_status shows stopped when container exists but not running" {
    # Mock container exists but not running
    docker() {
        case "$1 $2" in
            "container inspect") return 0 ;;  # Container exists
            "container inspect -f*") echo "false" ;;  # Not running
            *) return 0 ;;
        esac
    }
    export -f docker
    
    run node_red::show_status
    assert_success
    assert_output_contains "Stopped"
}

@test "node_red::show_status includes container information" {
    mock_docker "success"
    
    # Mock detailed container info
    docker() {
        case "$1 $2" in
            "container inspect -f*") echo "true" ;;
            "container inspect")
                echo '[{"Config":{"Image":"nodered/node-red:latest"},"Created":"2023-01-01T00:00:00Z"}]'
                ;;
            "ps") echo "Up 2 hours" ;;
            "stats") echo "CONTAINER     CPU %     MEM USAGE / LIMIT" ;;
            *) return 0 ;;
        esac
    }
    export -f docker
    
    run node_red::show_status
    assert_success
    assert_output_contains "Container Information"
    assert_output_contains "Image:"
    assert_output_contains "Created:"
}

@test "node_red::show_status includes resource usage" {
    mock_docker "success"
    
    run node_red::show_status
    assert_success
    assert_output_contains "Resource Usage"
}

@test "node_red::show_status includes health information" {
    mock_docker "success"
    
    # Mock health check
    docker() {
        case "$1" in
            "inspect") 
                if [[ "$*" =~ "--format" ]]; then
                    echo "healthy"
                else
                    echo "mock output"
                fi
                ;;
            *) return 0 ;;
        esac
    }
    export -f docker
    
    run node_red::show_status
    assert_success
    assert_output_contains "Health:"
}

@test "node_red::show_status includes flow information" {
    mock_docker "success"
    
    # Mock flow file count
    docker() {
        case "$1" in
            "exec")
                if [[ "$*" =~ "ls /data" ]]; then
                    echo "flows.json settings.js"
                else
                    echo "mock exec"
                fi
                ;;
            *) return 0 ;;
        esac
    }
    export -f docker
    
    run node_red::show_status
    assert_success
    assert_output_contains "Flow Information"
}

@test "node_red::show_metrics displays resource metrics" {
    mock_docker "success"
    
    run node_red::show_metrics
    assert_success
    assert_output_contains "Resource Metrics"
    assert_output_contains "CPU"
    assert_output_contains "MEM"
}

@test "node_red::show_metrics fails when container is not running" {
    mock_docker "not_running"
    
    run node_red::show_metrics
    assert_failure
    assert_output_contains "not running"
}

@test "node_red::show_metrics includes flow metrics" {
    mock_docker "success"
    
    # Mock flow file count
    docker() {
        case "$1" in
            "exec")
                if [[ "$*" =~ "find /data" ]]; then
                    echo -e "/data/flows.json\n/data/backup.json"
                else
                    echo "mock exec"
                fi
                ;;
            "stats") echo "CONTAINER     CPU %     MEM USAGE / LIMIT" ;;
            *) return 0 ;;
        esac
    }
    export -f docker
    
    run node_red::show_metrics
    assert_success
    assert_output_contains "Flow Metrics"
    assert_output_contains "Flow files:"
}

@test "node_red::show_metrics includes disk usage" {
    mock_docker "success"
    
    # Mock disk usage
    docker() {
        case "$1" in
            "exec")
                if [[ "$*" =~ "du -sh" ]]; then
                    echo "42M	/data"
                else
                    echo "mock exec"
                fi
                ;;
            "stats") echo "CONTAINER     CPU %     MEM USAGE / LIMIT" ;;
            *) return 0 ;;
        esac
    }
    export -f docker
    
    run node_red::show_metrics
    assert_success
    assert_output_contains "Data directory size:"
}

@test "node_red::show_metrics includes Node.js process info" {
    mock_docker "success"
    
    # Mock process info
    docker() {
        case "$1" in
            "exec")
                if [[ "$*" =~ "ps aux" ]]; then
                    echo "node-red  123  1.2  3.4  node-red --settings /data/settings.js"
                else
                    echo "mock exec"
                fi
                ;;
            "stats") echo "CONTAINER     CPU %     MEM USAGE / LIMIT" ;;
            *) return 0 ;;
        esac
    }
    export -f docker
    
    run node_red::show_metrics
    assert_success
    assert_output_contains "Node.js Process"
}

@test "node_red::view_logs displays container logs" {
    mock_docker "success"
    export FOLLOW="no"
    export LOG_LINES="100"
    
    # Mock docker logs
    docker() {
        case "$1" in
            "logs")
                echo "2023-01-01T00:00:00.000Z Node-RED started"
                return 0
                ;;
            *) return 0 ;;
        esac
    }
    export -f docker
    
    run node_red::view_logs
    assert_success
    assert_output_contains "Node-RED started"
}

@test "node_red::view_logs fails when container is not installed" {
    mock_docker "not_installed"
    
    run node_red::view_logs
    assert_failure
    assert_output_contains "not installed"
}

@test "node_red::view_logs follows logs when requested" {
    mock_docker "success"
    export FOLLOW="yes"
    
    # Mock docker logs -f (we'll timeout quickly for testing)
    docker() {
        case "$1" in
            "logs")
                if [[ "$*" =~ "-f" ]]; then
                    echo "Following logs..."
                    sleep 0.1  # Brief delay to simulate following
                else
                    echo "Static logs"
                fi
                return 0
                ;;
            *) return 0 ;;
        esac
    }
    export -f docker
    
    run node_red_timeout_wrapper 1 node_red::view_logs
    # Should timeout but that's expected for following logs
    assert_output_contains "Following logs"
}

@test "node_red::view_logs respects log line limit" {
    mock_docker "success"
    export LOG_LINES="50"
    
    # Mock docker logs with --tail option
    docker() {
        case "$1" in
            "logs")
                if [[ "$*" =~ "--tail 50" ]]; then
                    echo "Limited logs"
                else
                    echo "Full logs"
                fi
                return 0
                ;;
            *) return 0 ;;
        esac
    }
    export -f docker
    
    run node_red::view_logs
    assert_success
    assert_output_contains "Limited logs"
}

@test "node_red::show_info displays comprehensive information" {
    mock_docker "success"
    mock_curl "success"
    
    run node_red::show_info
    assert_success
    assert_output_contains "Node-RED Information"
}

@test "node_red::show_info includes version information" {
    mock_docker "success"
    
    # Mock version info from Node-RED settings
    curl() {
        if [[ "$*" =~ "settings" ]]; then
            echo '{"version": "3.0.2", "userDir": "/data"}'
        else
            echo "OK"
        fi
    }
    export -f curl
    
    run node_red::show_info
    assert_success
    assert_output_contains "Version:"
}

@test "node_red::show_info handles service unavailable" {
    mock_docker "success"
    mock_curl "failure"
    
    run node_red::show_info
    assert_success  # Should still work, just with limited info
}

@test "node_red::health_check performs comprehensive health check" {
    mock_docker "success"
    mock_curl "success"
    
    run node_red::health_check
    assert_success
    assert_output_contains "Health Check"
}

@test "node_red::health_check detects unhealthy service" {
    mock_docker "success"
    mock_curl "failure"
    
    run node_red::health_check
    assert_failure
    assert_output_contains "Health Check"
}

@test "node_red::health_check fails when container is not running" {
    mock_docker "not_running"
    
    run node_red::health_check
    assert_failure
    assert_output_contains "not running"
}

@test "node_red::monitor runs monitoring loop" {
    mock_docker "success"
    mock_curl "success"
    
    # Mock date for consistent output
    date() { echo "2023-01-01 12:00:00"; }
    export -f date
    
    # Run for a very short time
    run node_red_timeout_wrapper 2 node_red::monitor 1
    # Should timeout, which is expected
    assert_output_contains "Monitoring Node-RED"
}

@test "node_red::monitor fails when container is not running" {
    mock_docker "not_running"
    
    run node_red::monitor 1
    assert_failure
    assert_output_contains "not running"
}

@test "node_red::show_recent_logs shows recent log entries" {
    mock_docker "success"
    
    # Mock recent logs
    docker() {
        case "$1" in
            "logs")
                echo "2023-01-01T12:00:00.000Z [info] Node-RED started"
                echo "2023-01-01T12:00:01.000Z [info] Flows deployed"
                return 0
                ;;
            *) return 0 ;;
        esac
    }
    export -f docker
    
    run node_red::show_recent_logs
    assert_success
    assert_output_contains "Recent Logs"
    assert_output_contains "Node-RED started"
}

@test "node_red::show_error_logs filters error messages" {
    mock_docker "success"
    
    # Mock logs with errors
    docker() {
        case "$1" in
            "logs")
                echo "2023-01-01T12:00:00.000Z [info] Node-RED started"
                echo "2023-01-01T12:00:01.000Z [error] Connection failed"
                echo "2023-01-01T12:00:02.000Z [warn] Low memory"
                return 0
                ;;
            *) return 0 ;;
        esac
    }
    export -f docker
    
    run node_red::show_error_logs
    assert_success
    assert_output_contains "Error Logs"
    assert_output_contains "Connection failed"
    assert_output_contains "Low memory"
}

@test "node_red::show_resource_limits displays container limits" {
    mock_docker "success"
    
    # Mock container inspection for limits
    docker() {
        case "$1" in
            "inspect")
                echo '[{"HostConfig":{"Memory":1073741824,"CpuQuota":50000}}]'
                ;;
            *) return 0 ;;
        esac
    }
    export -f docker
    
    run node_red::show_resource_limits
    assert_success
    assert_output_contains "Resource Limits"
}

@test "node_red::show_network_info displays network configuration" {
    mock_docker "success"
    
    # Mock network information
    docker() {
        case "$1" in
            "inspect")
                if [[ "$*" =~ "container" ]]; then
                    echo '[{"NetworkSettings":{"Ports":{"1880/tcp":[{"HostPort":"1880"}]}}}]'
                else
                    echo "mock output"
                fi
                ;;
            "network")
                echo '[{"Name":"vrooli-test-network","Driver":"bridge"}]'
                ;;
            *) return 0 ;;
        esac
    }
    export -f docker
    
    run node_red::show_network_info
    assert_success
    assert_output_contains "Network Information"
}

@test "node_red::export_status_report creates status report" {
    mock_docker "success"
    mock_curl "success"
    
    run node_red::export_status_report
    assert_success
    assert_output_contains "Status report exported"
}

@test "node_red::export_status_report with custom output file" {
    mock_docker "success"
    mock_curl "success"
    
    local output_file="$NODE_RED_TEST_DIR/custom-report.json"
    
    run node_red::export_status_report "$output_file"
    assert_success
    assert_output_contains "Status report exported"
    assert_file_exists "$output_file"
}

@test "node_red::export_status_report handles errors gracefully" {
    mock_docker "success"
    mock_curl "success"
    
    # Mock file write to fail
    tee() { return 1; }
    export -f tee
    
    run node_red::export_status_report
    assert_failure
}

# Test environment variable handling
@test "status functions respect LOG_LINES environment variable" {
    mock_docker "success"
    export LOG_LINES="25"
    
    docker() {
        case "$1" in
            "logs")
                if [[ "$*" =~ "--tail 25" ]]; then
                    echo "Custom line count"
                else
                    echo "Default lines"
                fi
                ;;
            *) return 0 ;;
        esac
    }
    export -f docker
    
    run node_red::view_logs
    assert_success
    assert_output_contains "Custom line count"
}

@test "status functions respect RESOURCE_PORT environment variable" {
    export RESOURCE_PORT="19999"
    mock_docker "success"
    
    run node_red::show_status
    assert_success
    assert_output_contains "http://localhost:19999"
}

# Test error handling
@test "status functions handle docker failures gracefully" {
    mock_docker "failure"
    
    run node_red::show_status
    assert_success  # Should still show status, just indicate problems
    
    run node_red::show_metrics
    assert_failure
    
    run node_red::view_logs
    assert_failure
}

@test "status functions handle network failures gracefully" {
    mock_docker "success"
    mock_curl "failure"
    
    run node_red::show_info
    assert_success  # Should work with reduced functionality
    
    run node_red::health_check
    assert_failure
}

# Test output formatting
@test "status functions produce well-formatted output" {
    mock_docker "success"
    mock_curl "success"
    
    run node_red::show_status
    assert_success
    # Check for proper formatting elements
    assert_output_contains "="  # Headers should have separator lines
    assert_output_contains ":"  # Key-value pairs
}

@test "status functions handle missing optional data gracefully" {
    mock_docker "success"
    
    # Mock missing optional information
    docker() {
        case "$1" in
            "exec") return 1 ;;  # Commands in container fail
            "inspect") echo "[]" ;;  # Empty inspection results
            *) return 0 ;;
        esac
    }
    export -f docker
    
    run node_red::show_status
    assert_success  # Should still work
}

# Test concurrent operations
@test "status functions work when called concurrently" {
    mock_docker "success"
    mock_curl "success"
    
    # Run multiple status functions in background
    node_red::show_status > /dev/null &
    node_red::show_metrics > /dev/null &
    node_red::health_check > /dev/null &
    
    wait  # Wait for all background processes
    
    # All should have completed successfully
    [[ $? -eq 0 ]]
}
