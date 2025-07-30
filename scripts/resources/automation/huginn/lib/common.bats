#!/usr/bin/env bats
# Tests for Huginn lib/common.sh

load ../test_fixtures/test_helper

setup() {
    setup_test_environment
    mock_docker "success"
    source_huginn_scripts
}

teardown() {
    teardown_test_environment
}

@test "common.sh: check_docker succeeds when Docker is available" {
    # Mock system::is_command to return true for docker
    system::is_command() { [[ "$1" == "docker" ]]; }
    export -f system::is_command
    
    run huginn::check_docker
    assert_success
}

@test "common.sh: check_docker fails when Docker is not available" {
    # Mock system::is_command to return false
    system::is_command() { return 1; }
    export -f system::is_command
    
    run huginn::check_docker
    assert_failure
}

@test "common.sh: container_exists returns true when container exists" {
    mock_docker "success"
    
    run huginn::container_exists
    assert_success
}

@test "common.sh: container_exists returns false when container doesn't exist" {
    mock_docker "not_installed"
    
    run huginn::container_exists
    assert_failure
}

@test "common.sh: db_container_exists returns true when database exists" {
    mock_docker "success"
    
    run huginn::db_container_exists
    assert_success
}

@test "common.sh: is_installed returns true when both containers exist" {
    mock_docker "success"
    
    run huginn::is_installed
    assert_success
}

@test "common.sh: is_installed returns false when containers don't exist" {
    mock_docker "not_installed"
    
    run huginn::is_installed
    assert_failure
}

@test "common.sh: is_running returns true when containers are running" {
    mock_docker "success"
    
    run huginn::is_running
    assert_success
}

@test "common.sh: is_running returns false when containers are stopped" {
    mock_docker "not_running"
    
    run huginn::is_running
    assert_failure
}

@test "common.sh: is_healthy returns true when Huginn responds" {
    mock_docker "success"
    mock_curl "success"
    
    run huginn::is_healthy
    assert_success
}

@test "common.sh: is_healthy returns false when Huginn doesn't respond" {
    mock_docker "success"
    mock_curl "failure"
    
    run huginn::is_healthy
    assert_failure
}

@test "common.sh: wait_for_ready succeeds when Huginn becomes ready" {
    mock_docker "success"
    mock_curl "success"
    
    # Override the max attempts for faster testing
    export HUGINN_HEALTH_CHECK_MAX_ATTEMPTS=1
    
    run huginn::wait_for_ready
    assert_success
    assert_output_contains "ready"
}

@test "common.sh: wait_for_ready times out when Huginn doesn't start" {
    mock_docker "success"
    mock_curl "failure"
    
    # Override the max attempts for faster testing
    export HUGINN_HEALTH_CHECK_MAX_ATTEMPTS=1
    export HUGINN_HEALTH_CHECK_INTERVAL=0
    
    run huginn::wait_for_ready
    assert_failure
    assert_output_contains "Health check failed"
}

@test "common.sh: get_logs returns container logs" {
    mock_docker "success"
    
    run huginn::get_logs "huginn" "10"
    assert_success
    assert_output_contains "started successfully"
}

@test "common.sh: get_logs fails for non-existent container" {
    # Mock docker to show container doesn't exist
    docker() {
        case "$*" in
            *"ps -a --format"*) echo "" ;;
            *) return 1 ;;
        esac
    }
    export -f docker
    
    run huginn::get_logs "non-existent" "10"
    assert_failure
    assert_output_contains "not found"
}

@test "common.sh: rails_runner executes Ruby code" {
    mock_docker "success"
    export DOCKER_EXEC_MOCK="db_check"
    
    run huginn::rails_runner 'puts "DB_OK"'
    assert_success
    assert_output_contains "DB_OK"
}

@test "common.sh: rails_runner fails when container not running" {
    mock_docker "not_running"
    
    run huginn::rails_runner 'puts "test"'
    assert_failure
}

@test "common.sh: ensure_network creates network if needed" {
    # Mock docker to show network doesn't exist
    docker() {
        case "$*" in
            *"network ls"*) echo "" ;;
            *"network create"*) return 0 ;;
            *) return 0 ;;
        esac
    }
    export -f docker
    
    run huginn::ensure_network
    assert_success
}

@test "common.sh: ensure_data_directories creates directories" {
    # Use test directory
    export HUGINN_DATA_DIR="$HUGINN_TEST_DIR/data"
    export HUGINN_DB_DIR="$HUGINN_TEST_DIR/db"
    export HUGINN_UPLOADS_DIR="$HUGINN_TEST_DIR/uploads"
    
    run huginn::ensure_data_directories
    assert_success
    
    assert_directory_exists "$HUGINN_DATA_DIR"
    assert_directory_exists "$HUGINN_DB_DIR"
    assert_directory_exists "$HUGINN_UPLOADS_DIR"
}

@test "common.sh: get_system_stats returns JSON" {
    mock_docker "success"
    export DOCKER_EXEC_MOCK="system_stats"
    
    run huginn::get_system_stats
    assert_success
    assert_output_contains "users"
    assert_output_contains "agents"
    assert_output_contains "scenarios"
}

@test "common.sh: check_database returns true when connected" {
    mock_docker "success"
    export DOCKER_EXEC_MOCK="db_check"
    
    run huginn::check_database
    assert_success
}

@test "common.sh: get_version returns version string" {
    mock_docker "success"
    export DOCKER_EXEC_MOCK="version"
    
    run huginn::get_version
    assert_success
    assert_output_contains "Huginn"
    assert_output_contains "Rails"
}

@test "common.sh: validate_agent_id accepts valid IDs" {
    run huginn::validate_agent_id "123"
    assert_success
    
    run huginn::validate_agent_id "1"
    assert_success
    
    run huginn::validate_agent_id "999999"
    assert_success
}

@test "common.sh: validate_agent_id rejects invalid IDs" {
    run huginn::validate_agent_id "abc"
    assert_failure
    
    run huginn::validate_agent_id ""
    assert_failure
    
    run huginn::validate_agent_id "12.3"
    assert_failure
    
    run huginn::validate_agent_id "-5"
    assert_failure
}

@test "common.sh: validate_scenario_id accepts valid IDs" {
    run huginn::validate_scenario_id "456"
    assert_success
}

@test "common.sh: validate_scenario_id rejects invalid IDs" {
    run huginn::validate_scenario_id "xyz"
    assert_failure
}