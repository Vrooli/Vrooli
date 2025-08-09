#!/usr/bin/env bats
# Tests for Huginn manage.sh script

# Load Vrooli test infrastructure
source "${BATS_TEST_DIRNAME}/../../../__test/fixtures/setup.bash"

# Expensive setup operations run once per file
setup_file() {
    # Use Vrooli service test setup
    vrooli_setup_service_test "huginn"
    
    # Load resource specific configuration once per file
    SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    HUGINN_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Load configuration and manage script once
    source "${HUGINN_DIR}/config/defaults.sh"
    source "${HUGINN_DIR}/config/messages.sh"
    source "${SCRIPT_DIR}/manage.sh"
}

# Lightweight per-test setup 
setup() {
    # Setup standard Vrooli mocks
    vrooli_auto_setup
    
    # Set test environment variables (lightweight per-test)
    export HUGINN_CUSTOM_PORT="9999"
    export HUGINN_CONTAINER_NAME="huginn-test"
    export HUGINN_BASE_URL="http://localhost:9999"
    export FORCE="no"
    export YES="no"
    export ACTION="status"
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
    
    # Export config functions
    huginn::export_config
    huginn::export_messages
    
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

@test "manage.sh: help flag shows usage" {
    # Mock args parsing for help
    args::is_asking_for_help() { return 0; }
    export -f args::is_asking_for_help
    
    run huginn::show_help
    assert_success
    [[ "$output" =~ "Usage:" ]]
    [[ "$output" =~ "install" ]]
    [[ "$output" =~ "uninstall" ]]
    [[ "$output" =~ "status" ]]
}

@test "manage.sh: default action is install" {
    # Test that default action is set correctly
    huginn::parse_arguments
    
    [ "$ACTION" = "install" ]
}

@test "manage.sh: status command when running" {
    # Mock docker success for status check
    docker() {
        case "$*" in
            *"container inspect"*) return 0 ;;
            *"container inspect -f"*) echo "true" ;;
            *) return 0 ;;
        esac
    }
    export -f docker
    
    run huginn::show_status
    assert_success
    [[ "$output" =~ "Huginn Status" ]]
    [[ "$output" =~ "Running" ]]
}

@test "manage.sh: status command when not installed" {
    # Mock docker to return not installed
    docker() {
        case "$*" in
            *"container inspect"*) return 1 ;;
            *) return 1 ;;
        esac
    }
    export -f docker
    
    run huginn::is_installed
    assert_failure
}

@test "manage.sh: status command when not running" {
    # Mock docker to return not running
    docker() {
        case "$*" in
            *"container inspect"*) return 0 ;;
            *"container inspect -f"*) echo "false" ;;
            *) return 0 ;;
        esac
    }
    export -f docker
    
    run huginn::is_running
    assert_failure
}

@test "manage.sh: agents list operation" {
    # Mock docker exec for agents list
    docker() {
        case "$*" in
            *"exec"*"agents"*) 
                echo "RSS Weather Monitor"
                echo "Website Status Checker"
                ;;
            *) return 0 ;;
        esac
    }
    export -f docker
    
    run huginn::list_agents
    assert_success
    [[ "$output" =~ "RSS Weather Monitor" ]]
    [[ "$output" =~ "Website Status Checker" ]]
}

@test "manage.sh: agent show requires agent ID" {
    export AGENT_ID=""
    export OPERATION="show"
    
    run huginn::handle_agents
    assert_failure
    [[ "$output" =~ "Agent ID required" ]]
}

@test "manage.sh: agent show with valid ID" {
    # Mock docker exec for agent show
    docker() {
        case "$*" in
            *"exec"*"agent"*"1"*)
                echo "Agent Details"
                echo "RSS Weather Monitor"
                ;;
            *) return 0 ;;
        esac
    }
    export -f docker
    
    export AGENT_ID="1"
    export OPERATION="show"
    
    run huginn::show_agent "1"
    assert_success
    [[ "$output" =~ "Agent Details" ]]
    [[ "$output" =~ "RSS Weather Monitor" ]]
}

@test "manage.sh: scenarios list operation" {
    # Mock docker exec for scenarios list
    docker() {
        case "$*" in
            *"exec"*"scenarios"*)
                echo "Weather Monitoring Suite"
                ;;
            *) return 0 ;;
        esac
    }
    export -f docker
    
    run huginn::list_scenarios
    assert_success
    [[ "$output" =~ "Weather Monitoring Suite" ]]
}

@test "manage.sh: events recent operation" {
    # Mock docker exec for recent events
    docker() {
        case "$*" in
            *"exec"*"events"*"10"*)
                echo "Weather Update"
                ;;
            *) return 0 ;;
        esac
    }
    export -f docker
    
    run huginn::show_recent_events "10"
    assert_success
    [[ "$output" =~ "Weather Update" ]]
}

@test "manage.sh: health check when healthy" {
    # Mock docker and curl for health check
    docker() { return 0; }
    curl() { return 0; }
    export -f docker curl
    
    run huginn::health_check
    assert_success
    [[ "$output" =~ "Health Check" ]]
}

@test "manage.sh: health check when unhealthy" {
    # Mock docker to return unhealthy
    docker() { return 1; }
    export -f docker
    
    run huginn::is_healthy
    assert_failure
}

@test "manage.sh: logs operation" {
    # Mock docker logs
    docker() {
        case "$*" in
            *"logs"*"huginn"*"10"*)
                echo "started successfully"
                ;;
            *) return 0 ;;
        esac
    }
    export -f docker
    
    run huginn::get_logs "huginn" "10"
    assert_success
    [[ "$output" =~ "started successfully" ]]
}

@test "manage.sh: info command shows version" {
    # Mock docker exec for version
    docker() {
        case "$*" in
            *"exec"*"version"*)
                echo "Huginn"
                echo "Rails"
                ;;
            *) return 0 ;;
        esac
    }
    export -f docker
    
    run huginn::get_version
    assert_success
    [[ "$output" =~ "Huginn" ]]
    [[ "$output" =~ "Rails" ]]
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
    # Mock other containers for integration check
    docker() {
        case "$*" in
            *"ps --format"*) echo -e "minio\nredis\nnode-red\nollama" ;;
            *) return 0 ;;
        esac
    }
    export -f docker
    
    run huginn::handle_integration
    assert_success
    [[ "$output" =~ "Integration" ]]
    [[ "$output" =~ "MinIO" ]]
}

@test "manage.sh: monitor operation requires running container" {
    # Mock docker to return not running
    docker() {
        case "$*" in
            *"container inspect -f"*) echo "false" ;;
            *) return 0 ;;
        esac
    }
    export -f docker
    
    run huginn::is_running
    assert_failure
}