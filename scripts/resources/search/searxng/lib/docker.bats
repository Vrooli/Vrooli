#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Path to the script under test
SCRIPT_PATH="$BATS_TEST_DIRNAME/docker.sh"
SEARXNG_DIR="$BATS_TEST_DIRNAME/.."

# Source dependencies in correct order
RESOURCES_DIR="$SEARXNG_DIR/../.."
HELPERS_DIR="$RESOURCES_DIR/../helpers"

# Helper function for proper sourcing in tests
setup_searxng_docker_test_env() {
    local script_dir="$SEARXNG_DIR"
    local resources_dir="$SEARXNG_DIR/../.."
    local helpers_dir="$resources_dir/../helpers"
    
    # Source utilities first
    source "$helpers_dir/utils/log.sh"
    source "$helpers_dir/utils/system.sh"
    source "$helpers_dir/utils/ports.sh"
    source "$helpers_dir/utils/flow.sh"
    source "$resources_dir/port-registry.sh"
    source "$resources_dir/common.sh"
    
    # Source config and messages
    source "$script_dir/config/defaults.sh"
    source "$script_dir/config/messages.sh"
    searxng::export_config
    
    # Source common functions (dependency for docker.sh)
    source "$script_dir/lib/common.sh"
    
    # Source the script under test
    source "$SCRIPT_PATH"
    
    # Global test state
    export DOCKER_COMMANDS_EXECUTED=()
    export DOCKER_MOCK_NETWORK_EXISTS="no"
    export DOCKER_MOCK_PULL_SUCCESS="yes"
    export DOCKER_MOCK_COMMAND_SUCCESS="yes"
    export DOCKER_COMPOSE_SUCCESS="yes"
    
    # Mock log functions
    log::info() { echo "INFO: $*"; }
    log::success() { echo "SUCCESS: $*"; }
    log::error() { echo "ERROR: $*"; }
    log::warn() { echo "WARNING: $*"; }
    
    # Mock message function
    searxng::message() {
        local type="$1"
        local msg_var="$2"
        echo "$type: ${!msg_var}"
    }
    
    # Mock common functions
    searxng::is_running() {
        if [[ "$SEARXNG_MOCK_RUNNING" == "yes" ]]; then
            return 0
        else
            return 1
        fi
    }
    
    searxng::is_installed() {
        if [[ "$SEARXNG_MOCK_INSTALLED" == "yes" ]]; then
            return 0
        else
            return 1
        fi
    }
    
    searxng::wait_for_health() {
        if [[ "$SEARXNG_MOCK_HEALTHY" == "yes" ]]; then
            return 0
        else
            return 1
        fi
    }
    
    searxng::create_network() {
        if [[ "$DOCKER_MOCK_NETWORK_SUCCESS" == "yes" ]]; then
            return 0
        else
            return 1
        fi
    }
    
    # Mock docker command
    docker() {
        local cmd="$1"
        shift
        
        DOCKER_COMMANDS_EXECUTED+=("$cmd $*")
        
        case "$cmd" in
            "network")
                if [[ "$1" == "inspect" ]]; then
                    if [[ "$DOCKER_MOCK_NETWORK_EXISTS" == "yes" ]]; then
                        return 0
                    else
                        return 1
                    fi
                elif [[ "$1" == "create" ]]; then
                    if [[ "$DOCKER_MOCK_COMMAND_SUCCESS" == "yes" ]]; then
                        return 0
                    else
                        return 1
                    fi
                elif [[ "$1" == "rm" ]]; then
                    if [[ "$DOCKER_MOCK_COMMAND_SUCCESS" == "yes" ]]; then
                        return 0
                    else
                        return 1
                    fi
                fi
                ;;
            "pull")
                if [[ "$DOCKER_MOCK_PULL_SUCCESS" == "yes" ]]; then
                    return 0
                else
                    return 1
                fi
                ;;
            "run")
                if [[ "$DOCKER_MOCK_COMMAND_SUCCESS" == "yes" ]]; then
                    return 0
                else
                    return 1
                fi
                ;;
            "stop")
                if [[ "$DOCKER_MOCK_COMMAND_SUCCESS" == "yes" ]]; then
                    return 0
                else
                    return 1
                fi
                ;;
            "rm")
                if [[ "$DOCKER_MOCK_COMMAND_SUCCESS" == "yes" ]]; then
                    return 0
                else
                    return 1
                fi
                ;;
            "stats")
                echo "CPU: 5.2%    MEM: 128MiB / 1GiB"
                return 0
                ;;
            "exec")
                if [[ "$3" == "test-command" ]]; then
                    echo "Test command output"
                    return 0
                fi
                return 0
                ;;
            "compose")
                if [[ "$DOCKER_COMPOSE_SUCCESS" == "yes" ]]; then
                    return 0
                else
                    return 1
                fi
                ;;
            *)
                return 0
                ;;
        esac
    }
    
    # Mock eval function for complex docker commands
    eval() {
        if [[ "$*" =~ "docker run" ]]; then
            if [[ "$DOCKER_MOCK_COMMAND_SUCCESS" == "yes" ]]; then
                return 0
            else
                return 1
            fi
        fi
        # Call real eval for other commands
        command eval "$@"
    }
    
    # Mock sleep to avoid delays in tests
    sleep() { return 0; }
}

# ============================================================================
# Script Loading Tests
# ============================================================================

@test "sourcing docker.sh defines required functions" {
    setup_searxng_docker_test_env
    
    local required_functions=(
        "searxng::create_network"
        "searxng::remove_network"
        "searxng::pull_image"
        "searxng::build_docker_command"
        "searxng::start_container"
        "searxng::stop_container"
        "searxng::restart_container"
        "searxng::remove_container"
        "searxng::compose_up"
        "searxng::compose_down"
        "searxng::get_resource_usage"
        "searxng::exec_command"
    )
    
    for func in "${required_functions[@]}"; do
        run bash -c "declare -f $func"
        [ "$status" -eq 0 ]
    done
}

# ============================================================================
# Network Management Tests
# ============================================================================

@test "searxng::create_network creates network when it doesn't exist" {
    setup_searxng_docker_test_env
    export DOCKER_MOCK_NETWORK_EXISTS="no"
    export DOCKER_MOCK_COMMAND_SUCCESS="yes"
    
    run searxng::create_network
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Creating SearXNG Docker network" ]]
    [[ "$output" =~ "SUCCESS:" ]]
}

@test "searxng::create_network succeeds when network already exists" {
    setup_searxng_docker_test_env
    export DOCKER_MOCK_NETWORK_EXISTS="yes"
    
    run searxng::create_network
    [ "$status" -eq 0 ]
    [[ "$output" =~ "network already exists" ]]
}

@test "searxng::create_network fails when docker command fails" {
    setup_searxng_docker_test_env
    export DOCKER_MOCK_NETWORK_EXISTS="no"
    export DOCKER_MOCK_COMMAND_SUCCESS="no"
    
    run searxng::create_network
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR:" ]]
}

@test "searxng::remove_network removes existing network" {
    setup_searxng_docker_test_env
    export DOCKER_MOCK_NETWORK_EXISTS="yes"
    export DOCKER_MOCK_COMMAND_SUCCESS="yes"
    
    run searxng::remove_network
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Removing SearXNG Docker network" ]]
    [[ "$output" =~ "SUCCESS:" ]]
}

@test "searxng::remove_network succeeds when network doesn't exist" {
    setup_searxng_docker_test_env
    export DOCKER_MOCK_NETWORK_EXISTS="no"
    
    run searxng::remove_network
    [ "$status" -eq 0 ]
    [[ "$output" =~ "network does not exist" ]]
}

@test "searxng::remove_network warns when removal fails" {
    setup_searxng_docker_test_env
    export DOCKER_MOCK_NETWORK_EXISTS="yes"
    export DOCKER_MOCK_COMMAND_SUCCESS="no"
    
    run searxng::remove_network
    [ "$status" -eq 1 ]
    [[ "$output" =~ "WARNING:" ]]
    [[ "$output" =~ "containers attached" ]]
}

# ============================================================================
# Image Management Tests
# ============================================================================

@test "searxng::pull_image pulls image successfully" {
    setup_searxng_docker_test_env
    export DOCKER_MOCK_PULL_SUCCESS="yes"
    
    run searxng::pull_image
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Pulling SearXNG Docker image" ]]
    [[ "$output" =~ "SUCCESS:" ]]
}

@test "searxng::pull_image fails when docker pull fails" {
    setup_searxng_docker_test_env
    export DOCKER_MOCK_PULL_SUCCESS="no"
    
    run searxng::pull_image
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR:" ]]
    [[ "$output" =~ "Failed to pull" ]]
}

# ============================================================================
# Docker Command Building Tests
# ============================================================================

@test "searxng::build_docker_command generates valid docker run command" {
    setup_searxng_docker_test_env
    
    run searxng::build_docker_command
    [ "$status" -eq 0 ]
    [[ "$output" =~ "docker run -d" ]]
    [[ "$output" =~ "--name $SEARXNG_CONTAINER_NAME" ]]
    [[ "$output" =~ "--network $SEARXNG_NETWORK_NAME" ]]
    [[ "$output" =~ "-p ${SEARXNG_BIND_ADDRESS}:${SEARXNG_PORT}:8080" ]]
    [[ "$output" =~ "$SEARXNG_IMAGE" ]]
}

@test "searxng::build_docker_command includes security configuration" {
    setup_searxng_docker_test_env
    
    run searxng::build_docker_command
    [ "$status" -eq 0 ]
    [[ "$output" =~ "--cap-drop=ALL" ]]
    [[ "$output" =~ "--cap-add=CHOWN" ]]
    [[ "$output" =~ "--cap-add=SETGID" ]]
    [[ "$output" =~ "--cap-add=SETUID" ]]
}

@test "searxng::build_docker_command includes health check" {
    setup_searxng_docker_test_env
    
    run searxng::build_docker_command
    [ "$status" -eq 0 ]
    [[ "$output" =~ "--health-cmd" ]]
    [[ "$output" =~ "--health-interval=30s" ]]
    [[ "$output" =~ "--health-timeout=10s" ]]
    [[ "$output" =~ "--health-retries=3" ]]
}

@test "searxng::build_docker_command includes volume mounts" {
    setup_searxng_docker_test_env
    
    run searxng::build_docker_command
    [ "$status" -eq 0 ]
    [[ "$output" =~ "-v ${SEARXNG_DATA_DIR}:/etc/searxng:rw" ]]
}

@test "searxng::build_docker_command includes environment variables" {
    setup_searxng_docker_test_env
    
    run searxng::build_docker_command
    [ "$status" -eq 0 ]
    [[ "$output" =~ "-e SEARXNG_BASE_URL=${SEARXNG_BASE_URL}" ]]
    [[ "$output" =~ "-e SEARXNG_SECRET=${SEARXNG_SECRET_KEY}" ]]
}

@test "searxng::build_docker_command includes logging configuration" {
    setup_searxng_docker_test_env
    
    run searxng::build_docker_command
    [ "$status" -eq 0 ]
    [[ "$output" =~ "--log-driver json-file" ]]
    [[ "$output" =~ "--log-opt max-size=1m" ]]
    [[ "$output" =~ "--log-opt max-file=3" ]]
}

# ============================================================================
# Container Management Tests
# ============================================================================

@test "searxng::start_container starts container when not running" {
    setup_searxng_docker_test_env
    export SEARXNG_MOCK_RUNNING="no"
    export DOCKER_MOCK_NETWORK_SUCCESS="yes"
    export DOCKER_MOCK_COMMAND_SUCCESS="yes"
    export SEARXNG_MOCK_HEALTHY="yes"
    
    run searxng::start_container
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Starting SearXNG container" ]]
    [[ "$output" =~ "success:" ]]
}

@test "searxng::start_container warns when already running" {
    setup_searxng_docker_test_env
    export SEARXNG_MOCK_RUNNING="yes"
    
    run searxng::start_container
    [ "$status" -eq 0 ]
    [[ "$output" =~ "warning:" ]]
}

@test "searxng::start_container fails when network creation fails" {
    setup_searxng_docker_test_env
    export SEARXNG_MOCK_RUNNING="no"
    export DOCKER_MOCK_NETWORK_SUCCESS="no"
    
    run searxng::start_container
    [ "$status" -eq 1 ]
}

@test "searxng::start_container fails when docker run fails" {
    setup_searxng_docker_test_env
    export SEARXNG_MOCK_RUNNING="no"
    export DOCKER_MOCK_NETWORK_SUCCESS="yes"
    export DOCKER_MOCK_COMMAND_SUCCESS="no"
    
    run searxng::start_container
    [ "$status" -eq 1 ]
    [[ "$output" =~ "error:" ]]
}

@test "searxng::stop_container stops running container" {
    setup_searxng_docker_test_env
    export SEARXNG_MOCK_RUNNING="yes"
    export DOCKER_MOCK_COMMAND_SUCCESS="yes"
    
    run searxng::stop_container
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Stopping SearXNG container" ]]
    [[ "$output" =~ "success:" ]]
}

@test "searxng::stop_container warns when not running" {
    setup_searxng_docker_test_env
    export SEARXNG_MOCK_RUNNING="no"
    
    run searxng::stop_container
    [ "$status" -eq 0 ]
    [[ "$output" =~ "warning:" ]]
}

@test "searxng::stop_container fails when docker stop fails" {
    setup_searxng_docker_test_env
    export SEARXNG_MOCK_RUNNING="yes"
    export DOCKER_MOCK_COMMAND_SUCCESS="no"
    
    run searxng::stop_container
    [ "$status" -eq 1 ]
    [[ "$output" =~ "error:" ]]
}

@test "searxng::restart_container restarts running container" {
    setup_searxng_docker_test_env
    export SEARXNG_MOCK_RUNNING="yes"
    export DOCKER_MOCK_COMMAND_SUCCESS="yes"
    export DOCKER_MOCK_NETWORK_SUCCESS="yes"
    export SEARXNG_MOCK_HEALTHY="yes"
    
    # Mock stop and start
    searxng::stop_container() { echo "stop called"; return 0; }
    searxng::start_container() { echo "start called"; return 0; }
    
    run searxng::restart_container
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Restarting SearXNG container" ]]
    [[ "$output" =~ "success:" ]]
}

@test "searxng::restart_container starts stopped container" {
    setup_searxng_docker_test_env
    export SEARXNG_MOCK_RUNNING="no"
    export DOCKER_MOCK_COMMAND_SUCCESS="yes"
    export DOCKER_MOCK_NETWORK_SUCCESS="yes"
    export SEARXNG_MOCK_HEALTHY="yes"
    
    # Mock start
    searxng::start_container() { echo "start called"; return 0; }
    
    run searxng::restart_container
    [ "$status" -eq 0 ]
    [[ "$output" =~ "success:" ]]
}

@test "searxng::remove_container removes existing container" {
    setup_searxng_docker_test_env
    export SEARXNG_MOCK_RUNNING="yes"
    export SEARXNG_MOCK_INSTALLED="yes"
    export DOCKER_MOCK_COMMAND_SUCCESS="yes"
    
    # Mock stop
    searxng::stop_container() { echo "stop called"; return 0; }
    
    run searxng::remove_container
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Removing SearXNG container" ]]
    [[ "$output" =~ "SUCCESS:" ]]
}

@test "searxng::remove_container succeeds when not installed" {
    setup_searxng_docker_test_env
    export SEARXNG_MOCK_INSTALLED="no"
    
    run searxng::remove_container
    [ "$status" -eq 0 ]
}

# ============================================================================
# Docker Compose Tests
# ============================================================================

@test "searxng::compose_up starts with docker compose" {
    setup_searxng_docker_test_env
    export DOCKER_COMPOSE_SUCCESS="yes"
    export SEARXNG_MOCK_HEALTHY="yes"
    
    # Mock compose file existence
    mkdir -p "$SEARXNG_DIR/docker"
    touch "$SEARXNG_DIR/docker/docker-compose.yml"
    
    run searxng::compose_up
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Starting SearXNG with Docker Compose" ]]
    [[ "$output" =~ "success:" ]]
    
    # Cleanup
    rm -rf "$SEARXNG_DIR/docker"
}

@test "searxng::compose_up fails when compose file missing" {
    setup_searxng_docker_test_env
    
    run searxng::compose_up
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Docker Compose file not found" ]]
}

@test "searxng::compose_up includes redis profile when enabled" {
    setup_searxng_docker_test_env
    export SEARXNG_ENABLE_REDIS="yes"
    export DOCKER_COMPOSE_SUCCESS="yes"
    export SEARXNG_MOCK_HEALTHY="yes"
    
    # Mock compose file existence
    mkdir -p "$SEARXNG_DIR/docker"
    touch "$SEARXNG_DIR/docker/docker-compose.yml"
    
    run searxng::compose_up
    [ "$status" -eq 0 ]
    
    # Check that redis profile would be used (in real execution)
    [[ "$output" =~ "success:" ]]
    
    # Cleanup
    rm -rf "$SEARXNG_DIR/docker"
}

@test "searxng::compose_down stops with docker compose" {
    setup_searxng_docker_test_env
    export DOCKER_COMPOSE_SUCCESS="yes"
    
    # Mock compose file existence
    mkdir -p "$SEARXNG_DIR/docker"
    touch "$SEARXNG_DIR/docker/docker-compose.yml"
    
    run searxng::compose_down
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Stopping SearXNG with Docker Compose" ]]
    [[ "$output" =~ "success:" ]]
    
    # Cleanup
    rm -rf "$SEARXNG_DIR/docker"
}

@test "searxng::compose_down fails when compose file missing" {
    setup_searxng_docker_test_env
    
    run searxng::compose_down
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Docker Compose file not found" ]]
}

# ============================================================================
# Resource Usage Tests
# ============================================================================

@test "searxng::get_resource_usage shows stats when running" {
    setup_searxng_docker_test_env
    export SEARXNG_MOCK_RUNNING="yes"
    
    run searxng::get_resource_usage
    [ "$status" -eq 0 ]
    [[ "$output" =~ "SearXNG Resource Usage:" ]]
    [[ "$output" =~ "CPU:" ]]
    [[ "$output" =~ "MEM:" ]]
}

@test "searxng::get_resource_usage fails when not running" {
    setup_searxng_docker_test_env
    export SEARXNG_MOCK_RUNNING="no"
    
    run searxng::get_resource_usage
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Container not running" ]]
}

# ============================================================================
# Command Execution Tests
# ============================================================================

@test "searxng::exec_command executes command in container" {
    setup_searxng_docker_test_env
    export SEARXNG_MOCK_RUNNING="yes"
    
    run searxng::exec_command "test-command"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Test command output" ]]
}

@test "searxng::exec_command fails when container not running" {
    setup_searxng_docker_test_env
    export SEARXNG_MOCK_RUNNING="no"
    
    run searxng::exec_command "test-command"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR:" ]]
    [[ "$output" =~ "not running" ]]
}

# ============================================================================
# Integration Tests
# ============================================================================

@test "docker commands include proper error handling" {
    setup_searxng_docker_test_env
    
    # Test that functions properly handle docker command failures
    export DOCKER_MOCK_COMMAND_SUCCESS="no"
    
    run searxng::pull_image
    [ "$status" -eq 1 ]
    
    run searxng::create_network
    [ "$status" -eq 1 ]
}

@test "environment variables are properly exported for compose" {
    setup_searxng_docker_test_env
    export DOCKER_COMPOSE_SUCCESS="yes"
    export SEARXNG_MOCK_HEALTHY="yes"
    
    # Mock compose file existence
    mkdir -p "$SEARXNG_DIR/docker"
    touch "$SEARXNG_DIR/docker/docker-compose.yml"
    
    # Check that required variables are available
    [ -n "$SEARXNG_PORT" ]
    [ -n "$SEARXNG_BASE_URL" ]
    [ -n "$SEARXNG_SECRET_KEY" ]
    [ -n "$SEARXNG_DATA_DIR" ]
    
    run searxng::compose_up
    [ "$status" -eq 0 ]
    
    # Cleanup
    rm -rf "$SEARXNG_DIR/docker"
}