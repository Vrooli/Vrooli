#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Path to the script under test
SCRIPT_PATH="$BATS_TEST_DIRNAME/common.sh"
SEARXNG_DIR="$BATS_TEST_DIRNAME/.."

# Source dependencies in correct order (matching manage.sh)
RESOURCES_DIR="$SEARXNG_DIR/../.."
HELPERS_DIR="$RESOURCES_DIR/../helpers"

# Source utilities first
. "$HELPERS_DIR/utils/log.sh"
. "$HELPERS_DIR/utils/system.sh"
. "$HELPERS_DIR/utils/ports.sh"
. "$HELPERS_DIR/utils/flow.sh"
. "$RESOURCES_DIR/port-registry.sh"

# Helper function for proper sourcing in tests
setup_searxng_test_env() {
    local script_dir="$SEARXNG_DIR"
    local resources_dir="$SEARXNG_DIR/../.."
    local helpers_dir="$resources_dir/../helpers"
    
    # Source utilities first
    source "$helpers_dir/utils/log.sh"
    source "$helpers_dir/utils/system.sh"
    source "$helpers_dir/utils/ports.sh"
    source "$helpers_dir/utils/flow.sh"
    source "$resources_dir/port-registry.sh"
    
    # Source common.sh (provides resources functions)
    source "$resources_dir/common.sh"
    
    # Source config (depends on resources functions)
    source "$script_dir/config/defaults.sh"
    source "$script_dir/config/messages.sh"
    
    # Export configuration
    searxng::export_config
    
    # Source the script under test
    source "$SCRIPT_PATH"
    
    # Mock functions for tests
    resources::add_rollback_action() { 
        echo "Rollback action added: $1" >&2
        return 0
    }
    
    # Mock Docker commands
    docker() {
        case "$1" in
            "container")
                if [[ "$2" == "inspect" ]]; then
                    if [[ "$4" == "$SEARXNG_CONTAINER_NAME" ]]; then
                        case "$DOCKER_MOCK_STATE" in
                            "exists_running_healthy")
                                if [[ "$3" == "--format={{.State.Status}}" ]]; then
                                    echo "running"
                                elif [[ "$3" == "--format={{.State.Health.Status}}" ]]; then
                                    echo "healthy"
                                else
                                    return 0
                                fi
                                ;;
                            "exists_running_unhealthy")
                                if [[ "$3" == "--format={{.State.Status}}" ]]; then
                                    echo "running"
                                elif [[ "$3" == "--format={{.State.Health.Status}}" ]]; then
                                    echo "unhealthy"
                                else
                                    return 0
                                fi
                                ;;
                            "exists_stopped")
                                if [[ "$3" == "--format={{.State.Status}}" ]]; then
                                    echo "stopped"
                                else
                                    return 0
                                fi
                                ;;
                            "not_exists")
                                return 1
                                ;;
                            *)
                                return 0
                                ;;
                        esac
                    fi
                fi
                ;;
            "exec")
                if [[ "$2" == "$SEARXNG_CONTAINER_NAME" ]] && [[ "$*" =~ "searx.__version__" ]]; then
                    echo "1.0.0"
                fi
                ;;
            "logs")
                echo "Mock log output"
                ;;
            *)
                return 0
                ;;
        esac
    }
    
    # Mock curl for health checks
    curl() {
        if [[ "$CURL_MOCK_RESPONSE" == "success" ]]; then
            return 0
        else
            return 1
        fi
    }
    
    # Mock resources function
    resources::is_service_running() {
        if [[ "$SERVICE_RUNNING_MOCK" == "yes" ]]; then
            return 0
        else
            return 1
        fi
    }
    
    # Mock ports function
    ports::validate_port() {
        local port="$1"
        if [[ "$port" =~ ^[0-9]+$ ]] && [[ "$port" -ge 1024 ]] && [[ "$port" -le 65535 ]]; then
            return 0
        else
            return 1
        fi
    }
}

# Helper function for directory tests (avoids readonly variable conflicts)
setup_searxng_test_env_no_defaults() {
    local script_dir="$SEARXNG_DIR"
    local resources_dir="$SEARXNG_DIR/../.."
    local helpers_dir="$resources_dir/../helpers"
    
    source "$helpers_dir/utils/log.sh"
    source "$helpers_dir/utils/system.sh"
    source "$helpers_dir/utils/ports.sh"
    source "$helpers_dir/utils/flow.sh"
    source "$resources_dir/port-registry.sh"
    source "$resources_dir/common.sh"
    source "$script_dir/config/messages.sh"
    source "$SCRIPT_PATH"
    
    # Set minimal required variables
    SEARXNG_PORT="9200"
    SEARXNG_BASE_URL="http://localhost:9200"
    SEARXNG_CONTAINER_NAME="searxng"
    SEARXNG_DATA_DIR="/tmp/searxng-test"
    SEARXNG_LOG_LINES="50"
}

# ============================================================================
# Script Loading Tests
# ============================================================================

@test "sourcing common.sh defines required functions" {
    setup_searxng_test_env
    
    run bash -c "declare -f searxng::is_installed"
    [ "$status" -eq 0 ]
    
    run bash -c "declare -f searxng::is_running"
    [ "$status" -eq 0 ]
    
    run bash -c "declare -f searxng::is_healthy"
    [ "$status" -eq 0 ]
    
    run bash -c "declare -f searxng::get_status"
    [ "$status" -eq 0 ]
}

# ============================================================================
# Installation Check Tests
# ============================================================================

@test "searxng::is_installed returns false when Docker not available" {
    # Mock command to fail for docker
    command() {
        if [[ "$1" == "-v" ]] && [[ "$2" == "docker" ]]; then
            return 1
        fi
        return 0
    }
    
    setup_searxng_test_env
    
    run searxng::is_installed
    [ "$status" -eq 1 ]
}

@test "searxng::is_installed returns false when container does not exist" {
    export DOCKER_MOCK_STATE="not_exists"
    setup_searxng_test_env
    
    run searxng::is_installed
    [ "$status" -eq 1 ]
}

@test "searxng::is_installed returns true when container exists" {
    export DOCKER_MOCK_STATE="exists_stopped"
    setup_searxng_test_env
    
    run searxng::is_installed
    [ "$status" -eq 0 ]
}

# ============================================================================
# Running Check Tests
# ============================================================================

@test "searxng::is_running returns false when not installed" {
    export DOCKER_MOCK_STATE="not_exists"
    setup_searxng_test_env
    
    run searxng::is_running
    [ "$status" -eq 1 ]
}

@test "searxng::is_running returns false when container is stopped" {
    export DOCKER_MOCK_STATE="exists_stopped"
    setup_searxng_test_env
    
    run searxng::is_running
    [ "$status" -eq 1 ]
}

@test "searxng::is_running returns true when container is running" {
    export DOCKER_MOCK_STATE="exists_running_healthy"
    setup_searxng_test_env
    
    run searxng::is_running
    [ "$status" -eq 0 ]
}

# ============================================================================
# Health Check Tests
# ============================================================================

@test "searxng::is_healthy returns false when not running" {
    export DOCKER_MOCK_STATE="exists_stopped"
    setup_searxng_test_env
    
    run searxng::is_healthy
    [ "$status" -eq 1 ]
}

@test "searxng::is_healthy returns true when Docker health check is healthy" {
    export DOCKER_MOCK_STATE="exists_running_healthy"
    setup_searxng_test_env
    
    run searxng::is_healthy
    [ "$status" -eq 0 ]
}

@test "searxng::is_healthy returns false when Docker health check is unhealthy" {
    export DOCKER_MOCK_STATE="exists_running_unhealthy"
    setup_searxng_test_env
    
    run searxng::is_healthy
    [ "$status" -eq 1 ]
}

@test "searxng::is_healthy uses API check when no Docker health check" {
    export DOCKER_MOCK_STATE="exists_running_healthy"
    export SERVICE_RUNNING_MOCK="yes"
    export CURL_MOCK_RESPONSE="success"
    setup_searxng_test_env
    
    # Mock empty health status (no Docker health check configured)
    docker() {
        case "$1" in
            "container")
                if [[ "$2" == "inspect" ]] && [[ "$3" == "--format={{.State.Health.Status}}" ]]; then
                    echo ""  # Empty = no health check configured
                elif [[ "$2" == "inspect" ]] && [[ "$3" == "--format={{.State.Status}}" ]]; then
                    echo "running"
                else
                    return 0
                fi
                ;;
            *)
                return 0
                ;;
        esac
    }
    
    run searxng::is_healthy
    [ "$status" -eq 0 ]
}

# ============================================================================
# Status Tests
# ============================================================================

@test "searxng::get_status returns not_installed when not installed" {
    export DOCKER_MOCK_STATE="not_exists"
    setup_searxng_test_env
    
    run searxng::get_status
    [ "$status" -eq 0 ]
    [[ "$output" == "not_installed" ]]
}

@test "searxng::get_status returns stopped when installed but not running" {
    export DOCKER_MOCK_STATE="exists_stopped"
    setup_searxng_test_env
    
    run searxng::get_status
    [ "$status" -eq 0 ]
    [[ "$output" == "stopped" ]]
}

@test "searxng::get_status returns healthy when running and healthy" {
    export DOCKER_MOCK_STATE="exists_running_healthy"
    setup_searxng_test_env
    
    run searxng::get_status
    [ "$status" -eq 0 ]
    [[ "$output" == "healthy" ]]
}

@test "searxng::get_status returns unhealthy when running but unhealthy" {
    export DOCKER_MOCK_STATE="exists_running_unhealthy"
    setup_searxng_test_env
    
    run searxng::get_status
    [ "$status" -eq 0 ]
    [[ "$output" == "unhealthy" ]]
}

# ============================================================================
# Health Wait Tests
# ============================================================================

@test "searxng::wait_for_health returns quickly when already healthy" {
    export DOCKER_MOCK_STATE="exists_running_healthy"
    setup_searxng_test_env
    
    # Mock sleep to avoid actual waiting
    sleep() { return 0; }
    
    run searxng::wait_for_health 10
    [ "$status" -eq 0 ]
}

@test "searxng::wait_for_health times out when never becomes healthy" {
    export DOCKER_MOCK_STATE="exists_running_unhealthy"
    setup_searxng_test_env
    
    # Mock sleep to avoid actual waiting
    sleep() { return 0; }
    
    run searxng::wait_for_health 1
    [ "$status" -eq 1 ]
}

@test "searxng::wait_for_health uses default timeout when not specified" {
    export DOCKER_MOCK_STATE="exists_running_healthy"
    setup_searxng_test_env
    
    # Mock sleep to avoid actual waiting
    sleep() { return 0; }
    
    run searxng::wait_for_health
    [ "$status" -eq 0 ]
}

# ============================================================================
# Version Tests
# ============================================================================

@test "searxng::get_version returns unknown when not running" {
    export DOCKER_MOCK_STATE="exists_stopped"
    setup_searxng_test_env
    
    run searxng::get_version
    [ "$status" -eq 1 ]
    [[ "$output" == "unknown" ]]
}

@test "searxng::get_version returns version when running" {
    export DOCKER_MOCK_STATE="exists_running_healthy"
    setup_searxng_test_env
    
    run searxng::get_version
    [ "$status" -eq 0 ]
    [[ "$output" == "1.0.0" ]]
}

# ============================================================================
# Logs Tests
# ============================================================================

@test "searxng::get_logs fails when not installed" {
    export DOCKER_MOCK_STATE="not_exists"
    setup_searxng_test_env
    
    run searxng::get_logs
    [ "$status" -eq 1 ]
}

@test "searxng::get_logs returns logs when installed" {
    export DOCKER_MOCK_STATE="exists_running_healthy"
    setup_searxng_test_env
    
    run searxng::get_logs
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Mock log output" ]]
}

@test "searxng::get_logs uses custom line count" {
    export DOCKER_MOCK_STATE="exists_running_healthy"
    setup_searxng_test_env
    
    run searxng::get_logs 100
    [ "$status" -eq 0 ]
}

# ============================================================================
# Data Directory Tests
# ============================================================================

@test "searxng::ensure_data_dir creates directory when it doesn't exist" {
    setup_searxng_test_env_no_defaults
    
    export SEARXNG_DATA_DIR="/tmp/searxng-test-$(date +%s)"
    
    # Ensure directory doesn't exist
    rm -rf "$SEARXNG_DATA_DIR"
    
    run searxng::ensure_data_dir
    [ "$status" -eq 0 ]
    [ -d "$SEARXNG_DATA_DIR" ]
    
    # Cleanup
    rm -rf "$SEARXNG_DATA_DIR"
}

@test "searxng::ensure_data_dir succeeds when directory already exists" {
    setup_searxng_test_env_no_defaults
    
    export SEARXNG_DATA_DIR="/tmp/searxng-test-$(date +%s)"
    
    # Create directory first
    mkdir -p "$SEARXNG_DATA_DIR"
    
    run searxng::ensure_data_dir
    [ "$status" -eq 0 ]
    [ -d "$SEARXNG_DATA_DIR" ]
    
    # Cleanup
    rm -rf "$SEARXNG_DATA_DIR"
}

# ============================================================================
# Cleanup Tests
# ============================================================================

@test "searxng::cleanup with containers level" {
    setup_searxng_test_env
    
    # Mock successful cleanup
    export DOCKER_MOCK_STATE="exists_running_healthy"
    
    run searxng::cleanup "containers"
    [ "$status" -eq 0 ]
}

@test "searxng::cleanup with all level" {
    setup_searxng_test_env_no_defaults
    
    export SEARXNG_DATA_DIR="/tmp/searxng-test-$(date +%s)"
    mkdir -p "$SEARXNG_DATA_DIR"
    
    run searxng::cleanup "all"
    [ "$status" -eq 0 ]
    [ ! -d "$SEARXNG_DATA_DIR" ]
}

@test "searxng::cleanup uses default level when not specified" {
    setup_searxng_test_env
    
    run searxng::cleanup
    [ "$status" -eq 0 ]
}

# ============================================================================
# Configuration Validation Tests
# ============================================================================

@test "searxng::validate_config fails when required variables missing" {
    setup_searxng_test_env_no_defaults
    
    # Unset required variable
    unset SEARXNG_PORT
    
    run searxng::validate_config
    [ "$status" -eq 1 ]
}

@test "searxng::validate_config fails with invalid port" {
    setup_searxng_test_env
    
    export SEARXNG_PORT="invalid"
    
    run searxng::validate_config
    [ "$status" -eq 1 ]
}

@test "searxng::validate_config fails with port conflict" {
    export SERVICE_RUNNING_MOCK="yes"
    export DOCKER_MOCK_STATE="not_exists"
    setup_searxng_test_env
    
    run searxng::validate_config
    [ "$status" -eq 1 ]
}

@test "searxng::validate_config succeeds with valid configuration" {
    export DOCKER_MOCK_STATE="exists_running_healthy"
    setup_searxng_test_env
    
    run searxng::validate_config
    [ "$status" -eq 0 ]
}

# ============================================================================
# Information Display Tests
# ============================================================================

@test "searxng::show_info displays basic information" {
    export DOCKER_MOCK_STATE="exists_running_healthy"
    setup_searxng_test_env
    
    run searxng::show_info
    [ "$status" -eq 0 ]
    [[ "$output" =~ "SearXNG Information:" ]]
    [[ "$output" =~ "Status:" ]]
    [[ "$output" =~ "Port:" ]]
    [[ "$output" =~ "Base URL:" ]]
}

@test "searxng::show_info displays version when running" {
    export DOCKER_MOCK_STATE="exists_running_healthy"
    setup_searxng_test_env
    
    run searxng::show_info
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Version: 1.0.0" ]]
}

@test "searxng::show_info shows access information when healthy" {
    export DOCKER_MOCK_STATE="exists_running_healthy"
    setup_searxng_test_env
    
    run searxng::show_info
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Access SearXNG at:" ]]
}

# ============================================================================
# Function Existence Tests
# ============================================================================

@test "all required functions are defined" {
    setup_searxng_test_env
    
    local required_functions=(
        "searxng::is_installed"
        "searxng::is_running"
        "searxng::is_healthy"
        "searxng::get_status"
        "searxng::wait_for_health"
        "searxng::get_version"
        "searxng::get_logs"
        "searxng::ensure_data_dir"
        "searxng::cleanup"
        "searxng::validate_config"
        "searxng::show_info"
    )
    
    for func in "${required_functions[@]}"; do
        run bash -c "declare -f $func"
        [ "$status" -eq 0 ]
    done
}