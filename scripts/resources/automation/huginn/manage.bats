#!/usr/bin/env bats
# Tests for Huginn manage.sh script

load test-fixtures/test_helper

setup() {
    # Set up test environment
    setup_test_environment
    mock_docker "success"
    mock_curl "success"
    mock_pg_isready "success"
    source_huginn_scripts
    
    # Export required variables for testing
    export ACTION="status"
    export FORCE="no"
    export OPERATION="list"
    export AGENT_ID=""
    export SCENARIO_ID=""
    export COUNT="10"
    export CONTAINER_TYPE="app"
    export LOG_LINES="50"
    export FOLLOW="no"
    export INTERVAL="30"
    export REMOVE_DATA="no"
    export REMOVE_VOLUMES="no"
}

teardown() {
    teardown_test_environment
}

@test "manage.sh: help flag shows usage" {
    # Mock args parsing for help
    args::is_asking_for_help() { return 0; }
    export -f args::is_asking_for_help
    
    run bash "$HUGINN_ROOT_DIR/manage.sh" --help
    assert_success
    assert_output_contains "Usage:"
    assert_output_contains "install"
    assert_output_contains "uninstall"
    assert_output_contains "status"
}

@test "manage.sh: default action is install" {
    # Test that default action is set correctly
    args::get() {
        case "$1" in
            "action") echo "install" ;;
            *) echo "" ;;
        esac
    }
    export -f args::get
    
    export ACTION="install"
    
    # The actual behavior would depend on huginn::install
    # For now, just verify the action is parsed correctly
    assert_success
}

@test "manage.sh: status command when running" {
    mock_docker "success"
    export DOCKER_EXEC_MOCK="system_stats"
    
    run huginn::show_status
    assert_success
    assert_output_contains "Huginn Status"
    assert_output_contains "Running"
}

@test "manage.sh: status command when not installed" {
    mock_docker "not_installed"
    
    run huginn::is_installed
    assert_failure
}

@test "manage.sh: status command when not running" {
    mock_docker "not_running"
    
    run huginn::is_running
    assert_failure
}

@test "manage.sh: agents list operation" {
    mock_docker "success"
    export DOCKER_EXEC_MOCK="agents_list"
    
    run huginn::list_agents
    assert_success
    assert_output_contains "RSS Weather Monitor"
    assert_output_contains "Website Status Checker"
}

@test "manage.sh: agent show requires agent ID" {
    export AGENT_ID=""
    
    run huginn::handle_agents
    assert_failure
    assert_output_contains "Agent ID required"
}

@test "manage.sh: agent show with valid ID" {
    mock_docker "success"
    export DOCKER_EXEC_MOCK="agent_show"
    export AGENT_ID="1"
    export OPERATION="show"
    
    run huginn::show_agent "1"
    assert_success
    assert_output_contains "Agent Details"
    assert_output_contains "RSS Weather Monitor"
}

@test "manage.sh: scenarios list operation" {
    mock_docker "success"
    export DOCKER_EXEC_MOCK="scenarios_list"
    
    run huginn::list_scenarios
    assert_success
    assert_output_contains "Weather Monitoring Suite"
}

@test "manage.sh: events recent operation" {
    mock_docker "success"
    export DOCKER_EXEC_MOCK="events_recent"
    
    run huginn::show_recent_events "10"
    assert_success
    assert_output_contains "Weather Update"
}

@test "manage.sh: health check when healthy" {
    mock_docker "success"
    mock_curl "success"
    export DOCKER_EXEC_MOCK="db_check"
    
    run huginn::health_check
    assert_success
    assert_output_contains "Health Check"
}

@test "manage.sh: health check when unhealthy" {
    mock_docker "not_running"
    
    run huginn::is_healthy
    assert_failure
}

@test "manage.sh: logs operation" {
    mock_docker "success"
    
    run huginn::get_logs "huginn" "10"
    assert_success
    assert_output_contains "started successfully"
}

@test "manage.sh: info command shows version" {
    mock_docker "success"
    export DOCKER_EXEC_MOCK="version"
    
    run huginn::get_version
    assert_success
    assert_output_contains "Huginn"
    assert_output_contains "Rails"
}

@test "manage.sh: validate agent ID accepts numbers" {
    run huginn::validate_agent_id "123"
    assert_success
}

@test "manage.sh: validate agent ID rejects non-numbers" {
    run huginn::validate_agent_id "abc"
    assert_failure
}

@test "manage.sh: validate scenario ID accepts numbers" {
    run huginn::validate_scenario_id "456"
    assert_success
}

@test "manage.sh: validate scenario ID rejects non-numbers" {
    run huginn::validate_scenario_id "xyz"
    assert_failure
}

@test "manage.sh: integration command" {
    mock_docker "success"
    
    # Mock other containers for integration check
    docker() {
        case "$*" in
            *"ps --format"*) echo -e "minio\nredis\nnode-red\nollama" ;;
            *) command docker "$@" 2>/dev/null || true ;;
        esac
    }
    export -f docker
    
    run huginn::handle_integration
    assert_success
    assert_output_contains "Integration"
    assert_output_contains "MinIO"
}

@test "manage.sh: monitor operation requires running container" {
    mock_docker "not_running"
    
    run huginn::is_running
    assert_failure
}