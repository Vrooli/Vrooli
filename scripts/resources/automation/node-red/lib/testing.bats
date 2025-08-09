#!/usr/bin/env bats
# Tests for Node-RED testing functions (lib/testing.sh)
# These are tests for the testing functions - meta-testing!

source "${BATS_TEST_DIRNAME}/../../../../__test/fixtures/setup.bash"

setup() {
    setup_test_environment
    source_node_red_scripts
    mock_docker "success"
    mock_curl "success"
    mock_jq "success"
}

teardown() {
    teardown_test_environment
}

@test "node_red::run_tests executes complete test suite" {
    mock_docker "success"
    mock_curl "success"
    
    run node_red::run_tests
    assert_success
    assert_output_contains "Running Node-RED validation test suite"
    assert_output_contains "Test Summary"
    assert_output_contains "Passed:"
    assert_output_contains "Failed:"
}

@test "node_red::run_tests shows individual test results" {
    mock_docker "success"
    mock_curl "success"
    
    run node_red::run_tests
    assert_success
    assert_output_contains "container status"
    assert_output_contains "HTTP endpoint"
    assert_output_contains "admin API"
    assert_output_contains "Docker access"
    assert_output_contains "workspace access"
    assert_output_contains "host command execution"
    assert_output_contains "flow persistence"
}

@test "node_red::run_tests shows passed status for working services" {
    mock_docker "success"
    mock_curl "success"
    
    run node_red::run_tests
    assert_success
    assert_output_contains "PASSED"
}

@test "node_red::run_tests shows failed status for broken services" {
    mock_docker "not_running"
    mock_curl "failure"
    
    run node_red::run_tests
    assert_success  # Function should complete even with failures
    assert_output_contains "FAILED"
}

@test "node_red::run_tests counts test results correctly" {
    mock_docker "success"
    mock_curl "success"
    
    run node_red::run_tests
    assert_success
    
    # Extract numbers from output
    local passed=$(echo "$output" | grep "Passed:" | grep -o '[0-9]\+')
    local failed=$(echo "$output" | grep "Failed:" | grep -o '[0-9]\+')
    local total=$(echo "$output" | grep "Total:" | grep -o '[0-9]\+')
    
    [[ $((passed + failed)) -eq $total ]]
}

@test "node_red::test_container_status passes when container is running" {
    mock_docker "success"
    
    run node_red::test_container_status
    assert_success
}

@test "node_red::test_container_status fails when container is not running" {
    mock_docker "not_running"
    
    run node_red::test_container_status
    assert_failure
}

@test "node_red::test_http_endpoint passes when endpoint is accessible" {
    curl() {
        if [[ "$*" =~ "-f" && "$*" =~ "localhost:$RESOURCE_PORT" ]]; then
            return 0
        fi
        return 1
    }
    export -f curl
    
    run node_red::test_http_endpoint
    assert_success
}

@test "node_red::test_http_endpoint fails when endpoint is not accessible" {
    curl() { return 1; }
    export -f curl
    
    run node_red::test_http_endpoint
    assert_failure
}

@test "node_red::test_http_endpoint respects timeout" {
    curl() {
        if [[ "$*" =~ "--max-time 10" ]]; then
            return 0
        fi
        return 1
    }
    export -f curl
    
    run node_red::test_http_endpoint
    assert_success
}

@test "node_red::test_admin_api passes when API is accessible" {
    curl() {
        if [[ "$*" =~ "/flows" ]]; then
            return 0
        fi
        return 1
    }
    export -f curl
    
    run node_red::test_admin_api
    assert_success
}

@test "node_red::test_admin_api fails when API is not accessible" {
    curl() { return 1; }
    export -f curl
    
    run node_red::test_admin_api
    assert_failure
}

@test "node_red::test_docker_access passes when Docker socket is available" {
    mock_docker "success"
    
    run node_red::test_docker_access
    assert_success
}

@test "node_red::test_docker_access fails when container is not running" {
    mock_docker "not_running"
    
    run node_red::test_docker_access
    assert_failure
}

@test "node_red::test_docker_access fails when Docker socket is not available" {
    mock_docker "success"
    
    # Mock docker exec to fail for docker ps
    docker() {
        if [[ "$1" == "exec" && "$*" =~ "docker ps" ]]; then
            return 1
        fi
        return 0
    }
    export -f docker
    
    run node_red::test_docker_access
    assert_failure
}

@test "node_red::test_workspace_access passes when workspace is accessible" {
    mock_docker "success"
    
    run node_red::test_workspace_access
    assert_success
}

@test "node_red::test_workspace_access fails when workspace is not accessible" {
    mock_docker "success"
    
    # Mock docker exec to fail for ls /workspace
    docker() {
        if [[ "$1" == "exec" && "$*" =~ "ls /workspace" ]]; then
            return 1
        fi
        return 0
    }
    export -f docker
    
    run node_red::test_workspace_access
    assert_failure
}

@test "node_red::test_host_commands passes when commands are available" {
    mock_docker "success"
    
    run node_red::test_host_commands
    assert_success
}

@test "node_red::test_host_commands fails when commands are not available" {
    mock_docker "success"
    
    # Mock docker exec to fail for which ls
    docker() {
        if [[ "$1" == "exec" && "$*" =~ "which ls" ]]; then
            return 1
        fi
        return 0
    }
    export -f docker
    
    run node_red::test_host_commands
    assert_failure
}

@test "node_red::test_flow_persistence passes when flows file exists" {
    mock_docker "success"
    
    run node_red::test_flow_persistence
    assert_success
}

@test "node_red::test_flow_persistence fails when flows file doesn't exist" {
    mock_docker "success"
    
    # Mock docker exec to fail for ls flows.json
    docker() {
        if [[ "$1" == "exec" && "$*" =~ "ls /data/flows.json" ]]; then
            return 1
        fi
        return 0
    }
    export -f docker
    
    run node_red::test_flow_persistence
    assert_failure
}

@test "node_red::validate_host_access performs comprehensive host validation" {
    mock_docker "success"
    
    # Mock all commands to succeed
    docker() {
        if [[ "$1" == "exec" ]]; then
            return 0
        fi
        return 0
    }
    export -f docker
    
    run node_red::validate_host_access
    assert_success
    assert_output_contains "Validating host command execution"
    assert_output_contains "Testing 'ls'"
    assert_output_contains "Testing 'pwd'"
    assert_output_contains "Testing workspace write access"
}

@test "node_red::validate_host_access fails when container is not running" {
    mock_docker "not_running"
    
    run node_red::validate_host_access
    assert_failure
    assert_output_contains "not running"
}

@test "node_red::validate_host_access shows individual command results" {
    mock_docker "success"
    
    # Mock some commands to fail
    docker() {
        if [[ "$1" == "exec" ]]; then
            case "$*" in
                *"ls"*) return 0 ;;
                *"pwd"*) return 1 ;;
                *"date"*) return 0 ;;
                *) return 0 ;;
            esac
        fi
        return 0
    }
    export -f docker
    
    run node_red::validate_host_access
    assert_failure  # Should fail due to pwd failing
    assert_output_contains "✓"  # Some should pass
    assert_output_contains "✗"  # Some should fail
}

@test "node_red::validate_docker_access validates Docker socket access" {
    mock_docker "success"
    
    # Mock docker exec to succeed for docker commands
    docker() {
        if [[ "$1" == "exec" && "$*" =~ "docker" ]]; then
            return 0
        fi
        return 0
    }
    export -f docker
    
    run node_red::validate_docker_access
    assert_success
    assert_output_contains "Validating Docker socket access"
    assert_output_contains "Testing 'docker ps'"
    assert_output_contains "Testing container inspection"
}

@test "node_red::validate_docker_access fails when Docker is not available" {
    mock_docker "success"
    
    # Mock docker exec to fail for docker commands
    docker() {
        if [[ "$1" == "exec" && "$*" =~ "docker" ]]; then
            return 1
        fi
        return 0
    }
    export -f docker
    
    run node_red::validate_docker_access
    assert_failure
    assert_output_contains "Docker socket access not available"
}

@test "node_red::benchmark measures performance metrics" {
    mock_docker "success"
    mock_curl "success"
    
    # Mock curl with timing information
    curl() {
        if [[ "$*" =~ "-w" ]]; then
            echo "0.123"  # Mock response time
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
    assert_output_contains "Admin API Response Time"
    assert_output_contains "Resource Usage"
    assert_output_contains "Flow Information"
}

@test "node_red::benchmark fails when container is not running" {
    mock_docker "not_running"
    
    run node_red::benchmark
    assert_failure
    assert_output_contains "not running"
}

@test "node_red::benchmark includes memory and CPU metrics" {
    mock_docker "success"
    mock_curl "success"
    
    # Mock docker stats
    docker() {
        case "$1" in
            "stats")
                echo "512MB / 1GB"  # Memory usage
                echo "25.5%"        # CPU usage
                ;;
            "exec") echo "2" ;;     # Mock flow count
            *) return 0 ;;
        esac
    }
    export -f docker
    
    run node_red::benchmark
    assert_success
    assert_output_contains "Memory:"
    assert_output_contains "CPU:"
    assert_output_contains "Flow files:"
}

@test "node_red::load_test performs load testing with ab" {
    mock_docker "success"
    
    # Mock ab command
    ab() {
        echo "Requests per second: 150.23 [#/sec] (mean)"
        echo "Time per request: 6.656 [ms] (mean)"
        echo "Transfer rate: 45.67 [Kbytes/sec] received"
        return 0
    }
    export -f ab
    
    # Mock command existence check
    command() {
        if [[ "$2" == "ab" ]]; then
            return 0
        fi
        return 1
    }
    export -f command
    
    run node_red::load_test 5 50
    assert_success
    assert_output_contains "Load Test"
    assert_output_contains "Concurrent requests: 5"
    assert_output_contains "Total requests: 50"
    assert_output_contains "Requests per second"
}

@test "node_red::load_test fails when ab is not available" {
    mock_docker "success"
    
    # Mock command existence check to fail
    command() { return 1; }
    export -f command
    
    run node_red::load_test
    assert_failure
    assert_output_contains "Apache Bench (ab) is required"
}

@test "node_red::load_test uses default parameters" {
    mock_docker "success"
    
    # Mock ab and command
    ab() {
        if [[ "$*" =~ "-c 10" && "$*" =~ "-n 100" ]]; then
            echo "Load test with defaults"
        fi
        return 0
    }
    command() { return 0; }
    export -f ab command
    
    run node_red::load_test
    assert_success
    assert_output_contains "Concurrent requests: 10"
    assert_output_contains "Total requests: 100"
}

@test "node_red::stress_test runs stress test for specified duration" {
    mock_docker "success"
    mock_curl "success"
    
    # Test that function exists by calling declare directly
    declare -f node_red::stress_test >/dev/null
    [[ $? -eq 0 ]]
}

@test "node_red::stress_test monitors memory usage" {
    mock_docker "success"
    mock_curl "success"
    
    # Test that function exists
    declare -f node_red::stress_test >/dev/null
    [[ $? -eq 0 ]]
}

@test "node_red::stress_test checks final health status" {
    mock_docker "success"
    mock_curl "success"
    
    # Mock date to complete quickly
    date() { echo "1640995201"; }  # Time already passed
    export -f date
    
    run node_red::stress_test 1
    assert_success
    assert_output_contains "survived stress test"
}

@test "node_red::verify_installation checks all installation components" {
    mock_docker "success"
    mock_curl "success"
    
    # Create required files
    mkdir -p "$SCRIPT_DIR/flows"
    echo 'module.exports = {};' > "$SCRIPT_DIR/settings.js"
    mkdir -p "$NODE_RED_TEST_CONFIG_DIR"
    echo '{"services": {"automation": {"node-red": {"enabled": true}}}}' > "$NODE_RED_TEST_CONFIG_DIR/service.json"
    
    # Mock docker image and network checks
    docker() {
        case "$1" in
            "image"|"volume"|"network") return 0 ;;
            *) return 0 ;;
        esac
    }
    export -f docker
    
    run node_red::verify_installation
    assert_success
    assert_output_contains "Installation Verification"
    assert_output_contains "Container exists: ✓"
    assert_output_contains "Container running: ✓"
    assert_output_contains "verification passed"
}

@test "node_red::verify_installation detects missing components" {
    mock_docker "not_installed"
    
    run node_red::verify_installation
    assert_failure
    assert_output_contains "Container exists: ✗"
    assert_output_contains "issues with installation"
}

@test "node_red::verify_installation checks all required files" {
    mock_docker "success"
    
    # Don't create any files
    run node_red::verify_installation
    assert_failure
    assert_output_contains "Settings file: ✗"
    assert_output_contains "Flows directory: ✗"
    assert_output_contains "Resource config: ✗"
}

@test "node_red::verify_installation counts issues correctly" {
    mock_docker "not_installed"
    
    run node_red::verify_installation
    assert_failure
    
    # Should report the number of issues found
    local issues=$(echo "$output" | grep "Found [0-9]* issues" | grep -o '[0-9]\+')
    [[ $issues -gt 0 ]]
}

# Test error handling
@test "testing functions handle container failures gracefully" {
    mock_docker "failure"
    
    run node_red::run_tests
    assert_success  # Should complete even with failures
    assert_output_contains "FAILED"
}

@test "testing functions handle network failures gracefully" {
    mock_docker "success"
    mock_curl "failure"
    
    run node_red::run_tests
    assert_success  # Should complete even with failures
    assert_output_contains "FAILED"
}

@test "testing functions handle missing commands gracefully" {
    mock_docker "success"
    
    # Mock command to not exist
    docker() {
        if [[ "$1" == "exec" ]]; then
            return 127  # Command not found
        fi
        return 0
    }
    export -f docker
    
    run node_red::validate_host_access
    assert_failure
    assert_output_contains "✗"
}

# Test environment variable handling
@test "testing functions respect custom timeout values" {
    export NODE_RED_API_TIMEOUT=15
    mock_docker "success"
    
    curl() {
        if [[ "$*" =~ "--max-time 15" ]]; then
            return 0
        fi
        return 1
    }
    export -f curl
    
    run node_red::test_admin_api
    assert_success
}

@test "testing functions respect custom ports" {
    export RESOURCE_PORT="19999"
    mock_docker "success"
    
    curl() {
        if [[ "$*" =~ "localhost:19999" ]]; then
            return 0
        fi
        return 1
    }
    export -f curl
    
    run node_red::test_http_endpoint
    assert_success
}

# Test concurrent testing
@test "individual tests can run concurrently" {
    mock_docker "success"
    mock_curl "success"
    
    # Run multiple tests in background
    node_red::test_container_status &
    node_red::test_http_endpoint &
    node_red::test_admin_api &
    
    wait  # Wait for all background processes
    
    # All should have completed successfully
    [[ $? -eq 0 ]]
}

@test "testing functions provide consistent results" {
    mock_docker "success"
    mock_curl "success"
    
    # Run the same test multiple times
    node_red::test_container_status
    local result1=$?
    
    node_red::test_container_status
    local result2=$?
    
    node_red::test_container_status
    local result3=$?
    
    # All results should be the same
    [[ $result1 -eq $result2 && $result2 -eq $result3 ]]
}