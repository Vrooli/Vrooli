#!/usr/bin/env bats
# Tests for Kokoro docker.sh functions

# Load Vrooli test infrastructure
source "${BATS_TEST_DIRNAME}/../../../../__test/fixtures/setup.bash"

# Expensive setup operations (run once per file)
setup_file() {
    # Use appropriate setup function
    vrooli_setup_service_test "kokoro"

    # Load dependencies once
    SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    KOKORO_DIR="$(dirname "$SCRIPT_DIR")"

    # Load configuration and messages once
    source "${KOKORO_DIR}/config/defaults.sh"
    source "${KOKORO_DIR}/config/messages.sh"

    # Load docker functions once
    source "${SCRIPT_DIR}/docker.sh"

    # Export paths for use in setup()
    export SETUP_FILE_SCRIPT_DIR="$SCRIPT_DIR"
    export SETUP_FILE_KOKORO_DIR="$KOKORO_DIR"
}

# Lightweight per-test setup
setup() {
    # Setup standard mocks
    vrooli_auto_setup

    # Use paths from setup_file
    SCRIPT_DIR="${SETUP_FILE_SCRIPT_DIR}"
    KOKORO_DIR="${SETUP_FILE_KOKORO_DIR}"

    # Set test environment
    export KOKORO_CUSTOM_PORT="8880"
    export KOKORO_CONTAINER_NAME="kokoro-test"
    export KOKORO_GPU_ENABLED="no"
    export KOKORO_DATA_DIR="/tmp/kokoro-test"
    export YES="no"

    # Load dependencies
    SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    KOKORO_DIR="$(dirname "$SCRIPT_DIR")"

    # Mock common functions
    kokoro::get_docker_image() {
        if [[ "$KOKORO_GPU_ENABLED" == "yes" ]]; then
            echo "ghcr.io/remsky/kokoro-fastapi-gpu:latest"
        else
            echo "ghcr.io/remsky/kokoro-fastapi-cpu:latest"
        fi
    }

    kokoro::container_exists() { return 0; }
    kokoro::is_running() { return 0; }

    # Load config and messages from config files
    if [[ -f "${KOKORO_DIR}/config/defaults.sh" ]]; then
        source "${KOKORO_DIR}/config/defaults.sh"
        defaults::export_config 2>/dev/null || true
    fi
    if [[ -f "${KOKORO_DIR}/config/messages.sh" ]]; then
        source "${KOKORO_DIR}/config/messages.sh"
        messages::export_messages 2>/dev/null || true
    fi

    # Load the functions to test
    source "${KOKORO_DIR}/lib/docker.sh"
}

# BATS teardown function - runs after each test
teardown() {
    vrooli_cleanup_test
}

# Test image pulling - CPU version
@test "kokoro::pull_image pulls CPU image successfully" {
    export KOKORO_GPU_ENABLED="no"

    result=$(kokoro::docker::pull_image "no")

    [[ "$result" =~ "INFO:" ]]
    [[ "$result" =~ "Pulling" ]]
    [[ "$result" =~ "DOCKER_PULL:" ]]
    [[ "$result" =~ "kokoro-fastapi-cpu" ]]
}

# Test image pulling - GPU version
@test "kokoro::pull_image pulls GPU image successfully" {
    export KOKORO_GPU_ENABLED="yes"

    result=$(kokoro::docker::pull_image "yes")

    [[ "$result" =~ "INFO:" ]]
    [[ "$result" =~ "Pulling" ]]
    [[ "$result" =~ "DOCKER_PULL:" ]]
    [[ "$result" =~ "kokoro-fastapi-gpu" ]]
}

# Test container start
@test "kokoro::start_container starts container successfully" {
    result=$(kokoro::docker::start_container)

    [[ "$result" =~ "INFO:" ]]
    [[ "$result" =~ "Starting" ]]
    [[ "$result" =~ "DOCKER_RUN:" ]]
}

# Test container start with GPU enabled
@test "kokoro::start_container starts GPU container" {
    export KOKORO_GPU_ENABLED="yes"

    result=$(kokoro::docker::start_container)

    [[ "$result" =~ "DOCKER_RUN:" ]]
    [[ "$result" =~ "--gpus" ]] || [[ "$result" =~ "nvidia-docker" ]]
}

# Test GPU availability check
@test "kokoro::check_gpu_support checks GPU availability" {
    export KOKORO_GPU_ENABLED="yes"

    result=$(kokoro::docker::check_gpu_support)

    [[ "$result" =~ "GPU" ]]
}

# Test GPU availability check without GPU
@test "kokoro::check_gpu_support handles no GPU" {
    export KOKORO_GPU_ENABLED="no"

    result=$(kokoro::docker::check_gpu_support)

    [[ "$result" =~ "CPU" ]] || [[ "$result" =~ "no GPU" ]] || [[ "$result" =~ "No GPU" ]]
}

# Test Docker image selection
@test "kokoro::get_docker_image selects correct image" {
    # CPU image
    export KOKORO_GPU_ENABLED="no"
    result=$(kokoro::get_docker_image)
    [[ "$result" =~ "cpu" ]]

    # GPU image
    export KOKORO_GPU_ENABLED="yes"
    result=$(kokoro::get_docker_image)
    [[ "$result" =~ "gpu" ]]
}

# Test container health check
@test "kokoro::check_container_health verifies container health" {
    result=$(kokoro::docker::check_container_health)

    [[ "$result" =~ "healthy" ]] || [[ "$result" =~ "running" ]]
}

# GPU auto-detection tests - run in subshells to get clean readonly state

@test "GPU auto-detection defaults to yes when GPU available" {
    local defaults_path="${KOKORO_DIR}/config/defaults.sh"
    result=$(
        unset KOKORO_GPU_ENABLED GPU
        # Mock nvidia-smi and docker to simulate GPU presence
        nvidia-smi() { return 0; }
        docker() {
            case "$1" in
                "info") echo "  Runtimes: nvidia runc"; return 0 ;;
                *) return 0 ;;
            esac
        }
        export -f nvidia-smi docker
        source "$defaults_path"
        defaults::export_config
        echo "$KOKORO_GPU_ENABLED"
    )
    [ "$result" = "yes" ]
}

@test "GPU auto-detection defaults to no when no GPU" {
    local defaults_path="${KOKORO_DIR}/config/defaults.sh"
    result=$(
        unset KOKORO_GPU_ENABLED GPU
        # Mock nvidia-smi to fail
        nvidia-smi() { return 1; }
        export -f nvidia-smi
        source "$defaults_path"
        defaults::export_config
        echo "$KOKORO_GPU_ENABLED"
    )
    [ "$result" = "no" ]
}

@test "KOKORO_GPU_ENABLED manual override is respected" {
    local defaults_path="${KOKORO_DIR}/config/defaults.sh"
    result=$(
        export KOKORO_GPU_ENABLED="no"
        # Mock GPU as available - should be ignored because of manual override
        nvidia-smi() { return 0; }
        docker() {
            case "$1" in
                "info") echo "  Runtimes: nvidia runc"; return 0 ;;
                *) return 0 ;;
            esac
        }
        export -f nvidia-smi docker
        source "$defaults_path"
        defaults::export_config
        echo "$KOKORO_GPU_ENABLED"
    )
    [ "$result" = "no" ]
}
