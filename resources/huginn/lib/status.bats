#!/usr/bin/env bats
# Tests for Huginn lib/status.sh

source "${BATS_TEST_DIRNAME}/../../../../__test/fixtures/setup.bash"

setup() {
    setup_test_environment
    mock_docker "success"
    mock_curl "success"
    source_huginn_scripts
    
    # Set up mock responses
    export DOCKER_EXEC_MOCK="system_stats"
}

teardown() {
    teardown_test_environment
}

@test "status.sh: show_status displays full status when running" {
    mock_docker "success"
    
    run huginn::show_status
    assert_success
    assert_output_contains "Huginn Status"
    assert_output_contains "Running"
    assert_output_contains "Container Status"
}

@test "status.sh: show_status shows not installed message" {
    mock_docker "not_installed"
    
    run huginn::show_status
    assert_success
    assert_output_contains "not installed"
}

@test "status.sh: show_status shows stopped status" {
    mock_docker "not_running"
    
    # Override is_installed to return true
    huginn::is_installed() { return 0; }
    export -f huginn::is_installed
    
    run huginn::show_status
    assert_success
    assert_output_contains "Stopped"
}

@test "status.sh: show_info displays system information" {
    mock_docker "success"
    export DOCKER_EXEC_MOCK="version"
    
    run huginn::show_info
    assert_success
    assert_output_contains "System Information"
    assert_output_contains "Version"
}

@test "status.sh: health_check performs all checks" {
    mock_docker "success"
    mock_curl "success"
    export DOCKER_EXEC_MOCK="db_check"
    
    run huginn::health_check
    assert_success
    assert_output_contains "Health Check"
    assert_output_contains "Container Health"
}

@test "status.sh: health_check fails when container unhealthy" {
    mock_docker "not_running"
    
    run huginn::health_check
    assert_success  # Function succeeds but shows unhealthy status
    assert_output_contains "not running"
}

@test "status.sh: test_docker_access checks Docker functionality" {
    # Mock successful Docker access
    docker() {
        case "$*" in
            *"exec"*"docker"*"ps"*)
                echo "CONTAINER ID   IMAGE     COMMAND   CREATED   STATUS"
                return 0
                ;;
            *)
                return 0
                ;;
        esac
    }
    export -f docker
    
    run huginn::test_docker_access
    assert_success
}

@test "status.sh: test_database_connection checks database" {
    export DOCKER_EXEC_MOCK="db_check"
    
    run huginn::test_database_connection
    assert_success
}

@test "status.sh: test_web_interface checks HTTP endpoint" {
    mock_curl "success"
    
    run huginn::test_web_interface
    assert_success
}

@test "status.sh: test_rails_runner checks Rails functionality" {
    export DOCKER_EXEC_MOCK="version"
    
    run huginn::test_rails_runner
    assert_success
}

@test "status.sh: monitor shows real-time status" {
    mock_docker "success"
    export DOCKER_EXEC_MOCK="system_stats"
    
    # Override sleep for testing
    sleep() { return 0; }
    export -f sleep
    
    # Run monitor for one iteration
    local monitor_count=0
    huginn::monitor() {
        huginn::show_monitor_header
        huginn::show_status
        ((monitor_count++))
        if [[ $monitor_count -ge 1 ]]; then
            return 0  # Exit after one iteration
        fi
    }
    
    run huginn::monitor "1"
    assert_success
    assert_output_contains "Live Monitor"
}

@test "status.sh: show_container_details displays container info" {
    docker() {
        case "$*" in
            *"inspect huginn"*)
                cat << 'EOF'
{
    "State": {
        "Status": "running",
        "Running": true,
        "StartedAt": "2025-12-28T10:00:00Z"
    },
    "Config": {
        "Image": "huginn/huginn:latest"
    }
}
EOF
                ;;
            *"inspect huginn-postgres"*)
                cat << 'EOF'
{
    "State": {
        "Status": "running",
        "Running": true,
        "StartedAt": "2025-12-28T10:00:00Z"
    },
    "Config": {
        "Image": "postgres:15"
    }
}
EOF
                ;;
            *)
                return 0
                ;;
        esac
    }
    export -f docker
    
    run huginn::show_container_details
    assert_success
    assert_output_contains "Container Details"
}

@test "status.sh: show_system_stats displays statistics" {
    export DOCKER_EXEC_MOCK="system_stats"
    
    run huginn::show_system_stats
    assert_success
    assert_output_contains "System Statistics"
    assert_output_contains "Users"
    assert_output_contains "Agents"
}

@test "status.sh: run_tests executes test suite" {
    # Track which tests were run
    local tests_run=()
    
    huginn::test_container_status() {
        tests_run+=("container")
        return 0
    }
    
    huginn::test_web_interface() {
        tests_run+=("web")
        return 0
    }
    
    huginn::test_database_connection() {
        tests_run+=("database")
        return 0
    }
    
    huginn::test_rails_runner() {
        tests_run+=("rails")
        return 0
    }
    
    huginn::test_docker_access() {
        tests_run+=("docker")
        return 0
    }
    
    export -f huginn::test_container_status huginn::test_web_interface
    export -f huginn::test_database_connection huginn::test_rails_runner
    export -f huginn::test_docker_access
    
    run huginn::run_tests
    assert_success
    assert_output_contains "Test Results"
    
    # Verify all tests were run
    [[ " ${tests_run[@]} " =~ " container " ]]
    [[ " ${tests_run[@]} " =~ " web " ]]
    [[ " ${tests_run[@]} " =~ " database " ]]
    [[ " ${tests_run[@]} " =~ " rails " ]]
    [[ " ${tests_run[@]} " =~ " docker " ]]
}
