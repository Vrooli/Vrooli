#!/usr/bin/env bats
# Tests for Whisper docker.sh functions

# Setup for each test
setup() {
    # Load shared test infrastructure
    source "$(dirname "${BATS_TEST_FILENAME}")/../../../tests/bats-fixtures/common_setup.bash"
    
    # Setup standard mocks
    setup_standard_mocks
    
    # Set test environment
    export WHISPER_CUSTOM_PORT="8090"
    export WHISPER_CONTAINER_NAME="whisper-test"
    export WHISPER_GPU_ENABLED="no"
    export WHISPER_MODEL_SIZE="base"
    export WHISPER_DATA_DIR="/tmp/whisper-test"
    export YES="no"
    
    # Load dependencies
    SCRIPT_DIR="$(dirname "${BATS_TEST_FILENAME}")"
    WHISPER_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Mock system functions
    
    system::is_port_in_use() {
        return 1  # Port available
    }
    
    # Mock Docker functions
    
    # Mock log functions
    
    
    
    
    
    # Mock common functions
    whisper::get_docker_image() {
        if [[ "$WHISPER_GPU_ENABLED" == "yes" ]]; then
            echo "openai/whisper:gpu"
        else
            echo "openai/whisper:cpu"
        fi
    }
    
    whisper::container_exists() { return 0; }
    whisper::is_running() { return 0; }
    
    # Load configuration and messages
    source "${WHISPER_DIR}/config/defaults.sh"
    source "${WHISPER_DIR}/config/messages.sh"
    whisper::export_config
    whisper::export_messages
    
    # Load the functions to test
    source "${WHISPER_DIR}/lib/docker.sh"
}

# Test image pulling - CPU version
@test "whisper::pull_image pulls CPU image successfully" {
    export WHISPER_GPU_ENABLED="no"
    
    result=$(whisper::pull_image "no")
    
    [[ "$result" =~ "INFO:" ]]
    [[ "$result" =~ "Pulling" ]]
    [[ "$result" =~ "DOCKER_PULL:" ]]
    [[ "$result" =~ "openai/whisper:cpu" ]]
}

# Test image pulling - GPU version
@test "whisper::pull_image pulls GPU image successfully" {
    export WHISPER_GPU_ENABLED="yes"
    
    result=$(whisper::pull_image "yes")
    
    [[ "$result" =~ "INFO:" ]]
    [[ "$result" =~ "Pulling" ]]
    [[ "$result" =~ "DOCKER_PULL:" ]]
    [[ "$result" =~ "openai/whisper:gpu" ]]
}

# Test image pulling with default parameter
@test "whisper::pull_image uses default GPU setting" {
    export WHISPER_GPU_ENABLED="yes"
    
    result=$(whisper::pull_image)
    
    [[ "$result" =~ "openai/whisper:gpu" ]]
}

# Test image pulling failure
@test "whisper::pull_image handles Docker pull failure" {
    # Override docker to fail on pull
    docker() {
        case "$1" in
            "pull")
                echo "ERROR: Pull failed"
                return 1
                ;;
            *) return 0 ;;
        esac
    }
    
    run whisper::pull_image "no"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR: Failed to pull Docker image" ]]
}

# Test container start
@test "whisper::start_container starts container successfully" {
    result=$(whisper::start_container)
    
    [[ "$result" =~ "INFO:" ]]
    [[ "$result" =~ "Starting" ]]
    [[ "$result" =~ "DOCKER_RUN:" ]]
}

# Test container start with GPU enabled
@test "whisper::start_container starts GPU container" {
    export WHISPER_GPU_ENABLED="yes"
    
    result=$(whisper::start_container)
    
    [[ "$result" =~ "DOCKER_RUN:" ]]
    [[ "$result" =~ "--gpus" ]] || [[ "$result" =~ "nvidia-docker" ]]
}

# Test container start with custom parameters
@test "whisper::start_container uses custom configuration" {
    export WHISPER_MODEL_SIZE="large"
    export WHISPER_CUSTOM_PORT="8091"
    
    result=$(whisper::start_container)
    
    [[ "$result" =~ "DOCKER_RUN:" ]]
    [[ "$result" =~ "8091" ]]
    [[ "$result" =~ "large" ]]
}

# Test container stop
@test "whisper::stop_container stops container successfully" {
    result=$(whisper::stop_container)
    
    [[ "$result" =~ "INFO:" ]]
    [[ "$result" =~ "Stopping" ]]
    [[ "$result" =~ "DOCKER_STOP:" ]]
}

# Test container stop with non-running container
@test "whisper::stop_container handles non-running container" {
    # Override running check to return false
    whisper::is_running() { return 1; }
    
    result=$(whisper::stop_container)
    
    [[ "$result" =~ "not running" ]] || [[ "$result" =~ "already stopped" ]]
}

# Test container restart
@test "whisper::restart_container restarts container successfully" {
    result=$(whisper::restart_container)
    
    [[ "$result" =~ "INFO:" ]]
    [[ "$result" =~ "Restarting" ]]
    [[ "$result" =~ "DOCKER_RESTART:" ]]
}

# Test container removal
@test "whisper::remove_container removes container successfully" {
    result=$(whisper::remove_container)
    
    [[ "$result" =~ "INFO:" ]]
    [[ "$result" =~ "Removing" ]]
    [[ "$result" =~ "DOCKER_RM:" ]]
}

# Test container removal with non-existent container
@test "whisper::remove_container handles non-existent container" {
    # Override container check to return false
    whisper::container_exists() { return 1; }
    
    result=$(whisper::remove_container)
    
    [[ "$result" =~ "not found" ]] || [[ "$result" =~ "does not exist" ]]
}

# Test container logs
@test "whisper::get_logs retrieves container logs" {
    result=$(whisper::get_logs)
    
    [[ "$result" =~ "Mock whisper container logs" ]]
}

# Test container logs with follow option
@test "whisper::get_logs handles follow option" {
    # Mock docker logs with follow
    docker() {
        case "$1" in
            "logs")
                if [[ "$*" =~ "-f" ]]; then
                    echo "Following whisper logs..."
                else
                    echo "Static whisper logs"
                fi
                ;;
            *) return 0 ;;
        esac
    }
    
    result=$(whisper::get_logs "yes")
    [[ "$result" =~ "Following whisper logs" ]]
    
    result=$(whisper::get_logs "no")
    [[ "$result" =~ "Static whisper logs" ]]
}

# Test container inspection
@test "whisper::inspect_container returns container details" {
    result=$(whisper::inspect_container)
    
    [[ "$result" =~ "State" ]]
    [[ "$result" =~ "Running" ]]
    [[ "$result" =~ "Config" ]]
}

# Test container inspection with non-existent container
@test "whisper::inspect_container handles non-existent container" {
    # Override container check to return false
    whisper::container_exists() { return 1; }
    
    run whisper::inspect_container
    [ "$status" -eq 1 ]
}

# Test GPU availability check
@test "whisper::check_gpu_support checks GPU availability" {
    # With GPU enabled
    export WHISPER_GPU_ENABLED="yes"
    
    result=$(whisper::check_gpu_support)
    
    [[ "$result" =~ "GPU" ]]
}

# Test GPU availability check without GPU
@test "whisper::check_gpu_support handles no GPU" {
    export WHISPER_GPU_ENABLED="no"
    
    result=$(whisper::check_gpu_support)
    
    [[ "$result" =~ "CPU" ]] || [[ "$result" =~ "no GPU" ]]
}

# Test Docker image selection
@test "whisper::get_docker_image selects correct image" {
    # CPU image
    export WHISPER_GPU_ENABLED="no"
    result=$(whisper::get_docker_image)
    [[ "$result" =~ "cpu" ]]
    
    # GPU image
    export WHISPER_GPU_ENABLED="yes"
    result=$(whisper::get_docker_image)
    [[ "$result" =~ "gpu" ]]
}

# Test container health check
@test "whisper::check_container_health verifies container health" {
    result=$(whisper::check_container_health)
    
    [[ "$result" =~ "healthy" ]] || [[ "$result" =~ "running" ]]
}

# Test container health check with unhealthy container
@test "whisper::check_container_health detects unhealthy container" {
    # Override inspect to show unhealthy state
    docker() {
        case "$1" in
            "inspect")
                echo '{"State":{"Running":false,"Health":{"Status":"unhealthy"}}}'
                ;;
            *) return 0 ;;
        esac
    }
    
    run whisper::check_container_health
    [ "$status" -eq 1 ]
}

# Test volume mounting
@test "whisper::setup_volumes configures container volumes" {
    result=$(whisper::setup_volumes)
    
    [[ "$result" =~ "volume" ]] || [[ "$result" =~ "mount" ]]
}

# Test network configuration
@test "whisper::setup_network configures container networking" {
    result=$(whisper::setup_network)
    
    [[ "$result" =~ "network" ]] || [[ "$result" =~ "port" ]]
}

# Test environment variables setup
@test "whisper::setup_environment configures container environment" {
    result=$(whisper::setup_environment)
    
    [[ "$result" =~ "environment" ]] || [[ "$result" =~ "WHISPER" ]]
}

# Test container resource limits
@test "whisper::set_resource_limits configures resource constraints" {
    result=$(whisper::set_resource_limits)
    
    [[ "$result" =~ "memory" ]] || [[ "$result" =~ "cpu" ]] || [[ "$result" =~ "resource" ]]
}

# Test container cleanup
@test "whisper::cleanup_container performs complete cleanup" {
    result=$(whisper::cleanup_container)
    
    [[ "$result" =~ "cleanup" ]] || [[ "$result" =~ "DOCKER_RM:" ]]
}