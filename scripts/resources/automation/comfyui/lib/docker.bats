#!/usr/bin/env bats
# Tests for ComfyUI docker.sh functions

# Setup for each test
setup() {
    # Load shared test infrastructure
    source "$(dirname "${BATS_TEST_FILENAME}")/../../../tests/bats-fixtures/common_setup.bash"
    
    # Setup standard mocks
    setup_standard_mocks
    
    # Set test environment
    export COMFYUI_CUSTOM_PORT="8188"
    export COMFYUI_CONTAINER_NAME="comfyui-test"
    export COMFYUI_BASE_URL="http://localhost:8188"
    export COMFYUI_IMAGE="comfyanonymous/comfyui:latest"
    export COMFYUI_GPU_TYPE="cuda"
    export COMFYUI_DATA_DIR="/tmp/comfyui-test"
    export COMFYUI_MODELS_DIR="/tmp/comfyui-test/models"
    export COMFYUI_OUTPUT_DIR="/tmp/comfyui-test/output"
    export YES="no"
    
    # Load dependencies
    SCRIPT_DIR="$(dirname "${BATS_TEST_FILENAME}")"
    COMFYUI_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Create test directories
    mkdir -p "$COMFYUI_DATA_DIR" "$COMFYUI_MODELS_DIR" "$COMFYUI_OUTPUT_DIR"
    
    # Mock system functions
    
    # Mock Docker functions
    
    # Mock nvidia-smi
    nvidia-smi() {
        echo "GPU 0: NVIDIA GeForce RTX 4090"
        return 0
    }
    
    # Mock log functions
    
    
    
    
    
    # Mock ComfyUI functions
    comfyui::container_exists() { return 0; }
    comfyui::is_running() { return 0; }
    comfyui::get_docker_image() {
        case "$COMFYUI_GPU_TYPE" in
            "cuda") echo "comfyanonymous/comfyui:latest-cuda" ;;
            "rocm") echo "comfyanonymous/comfyui:latest-rocm" ;;
            *) echo "comfyanonymous/comfyui:latest-cpu" ;;
        esac
    }
    
    # Load configuration and messages
    source "${COMFYUI_DIR}/config/defaults.sh"
    source "${COMFYUI_DIR}/config/messages.sh"
    comfyui::export_config
    comfyui::export_messages
    
    # Load the functions to test
    source "${COMFYUI_DIR}/lib/docker.sh"
}

# Cleanup after each test
teardown() {
    rm -rf "$COMFYUI_DATA_DIR"
}

# Test image pulling - CUDA version
@test "comfyui::pull_image pulls CUDA image successfully" {
    export COMFYUI_GPU_TYPE="cuda"
    
    result=$(comfyui::pull_image)
    
    [[ "$result" =~ "INFO:" ]]
    [[ "$result" =~ "Pulling" ]]
    [[ "$result" =~ "DOCKER_PULL:" ]]
    [[ "$result" =~ "cuda" ]]
}

# Test image pulling - CPU version
@test "comfyui::pull_image pulls CPU image successfully" {
    export COMFYUI_GPU_TYPE="cpu"
    
    result=$(comfyui::pull_image)
    
    [[ "$result" =~ "INFO:" ]]
    [[ "$result" =~ "Pulling" ]]
    [[ "$result" =~ "DOCKER_PULL:" ]]
    [[ "$result" =~ "cpu" ]]
}

# Test image pulling - ROCm version
@test "comfyui::pull_image pulls ROCm image successfully" {
    export COMFYUI_GPU_TYPE="rocm"
    
    result=$(comfyui::pull_image)
    
    [[ "$result" =~ "INFO:" ]]
    [[ "$result" =~ "Pulling" ]]
    [[ "$result" =~ "DOCKER_PULL:" ]]
    [[ "$result" =~ "rocm" ]]
}

# Test image pulling failure
@test "comfyui::pull_image handles Docker pull failure" {
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
    
    run comfyui::pull_image
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR:" ]]
}

# Test container start with CUDA
@test "comfyui::start_container starts CUDA container successfully" {
    export COMFYUI_GPU_TYPE="cuda"
    
    result=$(comfyui::start_container)
    
    [[ "$result" =~ "INFO:" ]]
    [[ "$result" =~ "Starting" ]]
    [[ "$result" =~ "DOCKER_RUN:" ]]
    [[ "$result" =~ "--gpus" ]] || [[ "$result" =~ "nvidia" ]]
}

# Test container start with CPU
@test "comfyui::start_container starts CPU container successfully" {
    export COMFYUI_GPU_TYPE="cpu"
    
    result=$(comfyui::start_container)
    
    [[ "$result" =~ "INFO:" ]]
    [[ "$result" =~ "Starting" ]]
    [[ "$result" =~ "DOCKER_RUN:" ]]
    [[ ! "$result" =~ "--gpus" ]]
}

# Test container start with ROCm
@test "comfyui::start_container starts ROCm container successfully" {
    export COMFYUI_GPU_TYPE="rocm"
    
    result=$(comfyui::start_container)
    
    [[ "$result" =~ "INFO:" ]]
    [[ "$result" =~ "Starting" ]]
    [[ "$result" =~ "DOCKER_RUN:" ]]
    [[ "$result" =~ "rocm" ]] || [[ "$result" =~ "--device" ]]
}

# Test container stop
@test "comfyui::stop_container stops container successfully" {
    result=$(comfyui::stop_container)
    
    [[ "$result" =~ "INFO:" ]]
    [[ "$result" =~ "Stopping" ]]
    [[ "$result" =~ "DOCKER_STOP:" ]]
}

# Test container stop with non-running container
@test "comfyui::stop_container handles non-running container" {
    # Override running check to return false
    comfyui::is_running() { return 1; }
    
    result=$(comfyui::stop_container)
    
    [[ "$result" =~ "not running" ]] || [[ "$result" =~ "already stopped" ]]
}

# Test container restart
@test "comfyui::restart_container restarts container successfully" {
    result=$(comfyui::restart_container)
    
    [[ "$result" =~ "INFO:" ]]
    [[ "$result" =~ "Restarting" ]]
    [[ "$result" =~ "DOCKER_RESTART:" ]]
}

# Test container removal
@test "comfyui::remove_container removes container successfully" {
    result=$(comfyui::remove_container)
    
    [[ "$result" =~ "INFO:" ]]
    [[ "$result" =~ "Removing" ]]
    [[ "$result" =~ "DOCKER_RM:" ]]
}

# Test container removal with non-existent container
@test "comfyui::remove_container handles non-existent container" {
    # Override container check to return false
    comfyui::container_exists() { return 1; }
    
    result=$(comfyui::remove_container)
    
    [[ "$result" =~ "not found" ]] || [[ "$result" =~ "does not exist" ]]
}

# Test container logs
@test "comfyui::get_logs retrieves container logs" {
    result=$(comfyui::get_logs)
    
    [[ "$result" =~ "Mock ComfyUI container logs" ]]
}

# Test container logs with follow option
@test "comfyui::get_logs handles follow option" {
    # Mock docker logs with follow
    docker() {
        case "$1" in
            "logs")
                if [[ "$*" =~ "-f" ]]; then
                    echo "Following ComfyUI logs..."
                else
                    echo "Static ComfyUI logs"
                fi
                ;;
            *) return 0 ;;
        esac
    }
    
    result=$(comfyui::get_logs "yes")
    [[ "$result" =~ "Following ComfyUI logs" ]]
    
    result=$(comfyui::get_logs "no")
    [[ "$result" =~ "Static ComfyUI logs" ]]
}

# Test container inspection
@test "comfyui::inspect_container returns container details" {
    result=$(comfyui::inspect_container)
    
    [[ "$result" =~ "State" ]]
    [[ "$result" =~ "Running" ]]
    [[ "$result" =~ "Config" ]]
}

# Test container inspection with non-existent container
@test "comfyui::inspect_container handles non-existent container" {
    # Override container check to return false
    comfyui::container_exists() { return 1; }
    
    run comfyui::inspect_container
    [ "$status" -eq 1 ]
}

# Test GPU support check
@test "comfyui::check_gpu_support checks GPU availability" {
    result=$(comfyui::check_gpu_support)
    
    [[ "$result" =~ "GPU support:" ]]
    [[ "$result" =~ "NVIDIA" ]] || [[ "$result" =~ "detected" ]]
}

# Test GPU support check without GPU
@test "comfyui::check_gpu_support handles no GPU" {
    # Override nvidia-smi to fail
    nvidia-smi() {
        return 1
    }
    
    # Override system check
    system::is_command() {
        case "$1" in
            "nvidia-smi") return 1 ;;
            *) return 0 ;;
        esac
    }
    
    result=$(comfyui::check_gpu_support)
    
    [[ "$result" =~ "No GPU" ]] || [[ "$result" =~ "CPU mode" ]]
}

# Test Docker image selection
@test "comfyui::get_docker_image selects correct image" {
    # CUDA image
    export COMFYUI_GPU_TYPE="cuda"
    result=$(comfyui::get_docker_image)
    [[ "$result" =~ "cuda" ]]
    
    # CPU image
    export COMFYUI_GPU_TYPE="cpu"
    result=$(comfyui::get_docker_image)
    [[ "$result" =~ "cpu" ]]
    
    # ROCm image
    export COMFYUI_GPU_TYPE="rocm"
    result=$(comfyui::get_docker_image)
    [[ "$result" =~ "rocm" ]]
}

# Test container health check
@test "comfyui::check_container_health verifies container health" {
    result=$(comfyui::check_container_health)
    
    [[ "$result" =~ "healthy" ]] || [[ "$result" =~ "running" ]]
}

# Test volume mounting
@test "comfyui::setup_volumes configures container volumes" {
    result=$(comfyui::setup_volumes)
    
    [[ "$result" =~ "volume" ]] || [[ "$result" =~ "mount" ]]
    [[ "$result" =~ "$COMFYUI_DATA_DIR" ]]
    [[ "$result" =~ "$COMFYUI_MODELS_DIR" ]]
    [[ "$result" =~ "$COMFYUI_OUTPUT_DIR" ]]
}

# Test network configuration
@test "comfyui::setup_network configures container networking" {
    result=$(comfyui::setup_network)
    
    [[ "$result" =~ "network" ]] || [[ "$result" =~ "port" ]]
    [[ "$result" =~ "8188" ]]
}

# Test environment variables setup
@test "comfyui::setup_environment configures container environment" {
    result=$(comfyui::setup_environment)
    
    [[ "$result" =~ "environment" ]] || [[ "$result" =~ "COMFYUI" ]]
}

# Test container resource limits
@test "comfyui::set_resource_limits configures resource constraints" {
    result=$(comfyui::set_resource_limits)
    
    [[ "$result" =~ "memory" ]] || [[ "$result" =~ "cpu" ]] || [[ "$result" =~ "resource" ]]
}

# Test container command execution
@test "comfyui::exec_command executes commands in container" {
    local command="python --version"
    
    result=$(comfyui::exec_command "$command")
    
    [[ "$result" =~ "DOCKER_EXEC:" ]]
    [[ "$result" =~ "python --version" ]]
}

# Test container stats
@test "comfyui::get_container_stats retrieves container statistics" {
    # Mock docker stats
    docker() {
        case "$1" in
            "stats")
                echo "CONTAINER       CPU %     MEM USAGE / LIMIT     MEM %     NET I/O             BLOCK I/O           PIDS"
                echo "comfyui-test    15.50%    2.5GiB / 8GiB         31.25%    1.2MB / 800KB       50MB / 25MB         35"
                ;;
            *) return 0 ;;
        esac
    }
    
    result=$(comfyui::get_container_stats)
    
    [[ "$result" =~ "CPU" ]]
    [[ "$result" =~ "MEM" ]]
    [[ "$result" =~ "15.50%" ]]
}

# Test container backup
@test "comfyui::backup_container creates container backup" {
    local backup_path="/tmp/comfyui_backup.tar"
    
    result=$(comfyui::backup_container "$backup_path")
    
    [[ "$result" =~ "backup" ]]
    [[ "$result" =~ "$backup_path" ]]
}

# Test container restore
@test "comfyui::restore_container restores from backup" {
    local backup_path="/tmp/comfyui_backup.tar"
    touch "$backup_path"
    
    result=$(comfyui::restore_container "$backup_path")
    
    [[ "$result" =~ "restore" ]]
    [[ "$result" =~ "$backup_path" ]]
    
    rm -f "$backup_path"
}

# Test container cleanup
@test "comfyui::cleanup_container performs complete cleanup" {
    result=$(comfyui::cleanup_container)
    
    [[ "$result" =~ "cleanup" ]] || [[ "$result" =~ "DOCKER_RM:" ]]
}