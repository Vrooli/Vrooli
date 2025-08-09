#!/usr/bin/env bats
# Tests for Whisper common.sh functions

# Load Vrooli test infrastructure (REQUIRED)
source "${BATS_TEST_DIRNAME}/../../../../__test/fixtures/setup.bash"

# Expensive setup operations (run once per file)
setup_file() {
    # Use appropriate setup function
    vrooli_setup_service_test "whisper"
    
    # Load dependencies once
    SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    WHISPER_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Source library files
    source "${SCRIPT_DIR}/common.sh"
    source "${WHISPER_DIR}/config/defaults.sh"
    source "${WHISPER_DIR}/config/messages.sh"
    
    # Export paths for use in setup()
    export SETUP_FILE_SCRIPT_DIR="$SCRIPT_DIR"
    export SETUP_FILE_WHISPER_DIR="$WHISPER_DIR"
}

# Lightweight per-test setup
setup() {
    # Setup standard mocks
    vrooli_auto_setup
    
    # Set test environment
    export WHISPER_CONTAINER_NAME="whisper-test"
    export WHISPER_PORT="9090"
    export WHISPER_BASE_URL="http://localhost:9090"
    export WHISPER_DATA_DIR="/tmp/whisper-test"
    export WHISPER_MODELS_DIR="/tmp/whisper-test/models"
    export WHISPER_UPLOADS_DIR="/tmp/whisper-test/uploads"
    export WHISPER_API_TIMEOUT="5"
    export WHISPER_STARTUP_MAX_WAIT="60"
    export WHISPER_STARTUP_WAIT_INTERVAL="2"
    export WHISPER_GPU_ENABLED="no"
    export WHISPER_IMAGE="test-image:latest"
    export WHISPER_CPU_IMAGE="test-cpu-image:latest"
    
    # Mock message variables
    export MSG_DOCKER_NOT_FOUND="Docker is not installed"
    export MSG_CREATING_DIRS="Creating Whisper data directory..."
    export MSG_CREATE_DIRS_FAILED="Failed to create Whisper directories"
    export MSG_DIRECTORIES_CREATED="Whisper directories created"
    export MSG_INVALID_MODEL="❌ Invalid model size specified"
    export MSG_PORT_IN_USE="Port 9090 is already in use"
    export MSG_GPU_NOT_AVAILABLE="⚠️  GPU not available, falling back to CPU"
    
    # Use paths from setup_file
    SCRIPT_DIR="${SETUP_FILE_SCRIPT_DIR}"
    WHISPER_DIR="${SETUP_FILE_WHISPER_DIR}"
    
    # Export config functions
    whisper::export_config
    whisper::export_messages
}

# BATS teardown function - runs after each test
teardown() {
    vrooli_cleanup_test
    
    # Clean up test directory
    rm -rf "$WHISPER_DATA_DIR"
}

# Test Docker checking
@test "whisper::check_docker succeeds when Docker is available" {
    # Mock docker commands to succeed
    docker() {
        case "$1" in
            "info") return 0 ;;
            "ps") return 0 ;;
            *) return 0 ;;
        esac
    }
    
    run whisper::check_docker
    [ "$status" -eq 0 ]
}

@test "whisper::check_docker fails when Docker is not installed" {
    # Mock docker command not found
    system::is_command() {
        case "$1" in
            "docker") return 1 ;;
            *) return 1 ;;
        esac
    }
    
    run whisper::check_docker
    [ "$status" -eq 1 ]
    [[ "$output" == *"Docker is not installed"* ]]
}

# Test container existence checking
@test "whisper::container_exists returns true when container exists" {
    # Mock docker ps to show our container
    docker() {
        if [[ "$1" == "ps" && "$*" == *"whisper-test"* ]]; then
            echo "whisper-test"
        fi
    }
    
    run whisper::container_exists
    [ "$status" -eq 0 ]
}

@test "whisper::container_exists returns false when container doesn't exist" {
    # Mock docker ps to show no containers
    docker() {
        if [[ "$1" == "ps" ]]; then
            echo ""
        fi
    }
    
    run whisper::container_exists
    [ "$status" -eq 1 ]
}

# Test model validation
@test "whisper::validate_model accepts valid model sizes" {
    local valid_models=("tiny" "base" "small" "medium" "large" "large-v2" "large-v3")
    
    for model in "${valid_models[@]}"; do
        run whisper::validate_model "$model"
        [ "$status" -eq 0 ]
    done
}

@test "whisper::validate_model rejects invalid model sizes" {
    run whisper::validate_model "invalid"
    [ "$status" -eq 1 ]
    [[ "$output" == *"Invalid model size specified"* ]]
}

# Test model size retrieval
@test "whisper::get_model_size returns correct sizes" {
    export WHISPER_MODEL_SIZE_TINY="0.04"
    export WHISPER_MODEL_SIZE_BASE="0.07"
    export WHISPER_MODEL_SIZE_LARGE="1.55"
    
    run whisper::get_model_size "tiny"
    [ "$output" = "0.04" ]
    
    run whisper::get_model_size "base"
    [ "$output" = "0.07" ]
    
    run whisper::get_model_size "large"
    [ "$output" = "1.55" ]
    
    run whisper::get_model_size "invalid"
    [ "$output" = "unknown" ]
}

# Test directory creation
@test "whisper::create_directories creates all required directories" {
    run whisper::create_directories
    [ "$status" -eq 0 ]
    [ -d "$WHISPER_DATA_DIR" ]
    [ -d "$WHISPER_MODELS_DIR" ]
    [ -d "$WHISPER_UPLOADS_DIR" ]
}

@test "whisper::create_directories handles creation failure" {
    # Mock mkdir to fail
    mkdir() {
        return 1
    }
    
    run whisper::create_directories
    [ "$status" -eq 1 ]
}

# Test port availability
@test "whisper::is_port_available succeeds when port is free" {
    run whisper::is_port_available "9090"
    [ "$status" -eq 0 ]
}

@test "whisper::is_port_available fails when port is in use" {
    # Mock port as in use
    system::is_port_in_use() {
        return 0  # Port is in use
    }
    
    run whisper::is_port_available "9090"
    [ "$status" -eq 1 ]
}

# Test GPU availability
@test "whisper::is_gpu_available returns false when nvidia-smi not available" {
    run whisper::is_gpu_available
    [ "$status" -eq 1 ]
}

@test "whisper::is_gpu_available returns true when GPU is available" {
    # Mock nvidia-smi to be available and working
    system::is_command() {
        case "$1" in
            "nvidia-smi") return 0 ;;
            *) return 1 ;;
        esac
    }
    
    nvidia-smi() {
        return 0
    }
    
    run whisper::is_gpu_available
    [ "$status" -eq 0 ]
}

# Test Docker image selection
@test "whisper::get_docker_image returns CPU image by default" {
    run whisper::get_docker_image
    [ "$output" = "$WHISPER_CPU_IMAGE" ]
}

@test "whisper::get_docker_image returns GPU image when GPU is enabled and available" {
    export WHISPER_GPU_ENABLED="yes"
    
    # Mock GPU as available
    whisper::is_gpu_available() {
        return 0
    }
    
    run whisper::get_docker_image
    [ "$output" = "$WHISPER_IMAGE" ]
}