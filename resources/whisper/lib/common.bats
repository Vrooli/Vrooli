#!/usr/bin/env bats
# Tests for Whisper lib/common.sh functions

# Load Vrooli test infrastructure (REQUIRED)
source "${BATS_TEST_DIRNAME}/../../../../__test/fixtures/setup.bash"

# Expensive setup operations (run once per file)  
setup_file() {
    # Use appropriate setup function
    vrooli_setup_service_test "whisper"
    
    # Load dependencies once
    SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    
    # Source var.sh and dependencies first
    source "${SCRIPT_DIR}/../../../../lib/utils/var.sh"
    source "${var_LOG_FILE}"
    source "${var_LIB_SYSTEM_DIR}/system_commands.sh"
    
    # Mock message variables that common.sh expects
    export MSG_DOCKER_NOT_FOUND="Docker not found"
    export MSG_DOCKER_INSTALL_HINT="Install Docker"
    export MSG_DOCKER_NOT_RUNNING="Docker not running"
    export MSG_DOCKER_START_HINT="Start Docker"
    export MSG_DOCKER_NO_PERMISSIONS="No Docker permissions"
    export MSG_DOCKER_PERMISSIONS_HINT="Fix Docker permissions"
    export MSG_DOCKER_LOGOUT_HINT="Logout and login"
    export MSG_PORT_IN_USE="Port in use"
    export MSG_INVALID_MODEL="Invalid model"
    export MSG_CREATING_DIRS="Creating directories"
    export MSG_CREATE_DIRS_FAILED="Failed to create directories"
    export MSG_DIRECTORIES_CREATED="Directories created"
    export MSG_WAITING_STARTUP="Waiting for startup"
    export MSG_HEALTHY="Service healthy"
    export MSG_STARTUP_TIMEOUT="Startup timeout"
    export MSG_REMOVING_CONTAINER="Removing container"
    export MSG_GPU_NOT_AVAILABLE="GPU not available"
    
    # Set Whisper configuration
    export WHISPER_CONTAINER_NAME="whisper-test"
    export WHISPER_DATA_DIR="${HOME}/.whisper"
    export WHISPER_MODELS_DIR="${WHISPER_DATA_DIR}/models"
    export WHISPER_UPLOADS_DIR="${WHISPER_DATA_DIR}/uploads"
    export WHISPER_BASE_URL="http://localhost:9999"
    export WHISPER_STARTUP_MAX_WAIT=60
    export WHISPER_STARTUP_WAIT_INTERVAL=5
    export WHISPER_API_TIMEOUT=10
    export WHISPER_IMAGE="whisper:gpu"
    export WHISPER_CPU_IMAGE="whisper:cpu"
    export WHISPER_GPU_ENABLED="no"
    
    # Source the common.sh library
    source "${SCRIPT_DIR}/../common.sh"
    
    # Export paths for use in setup()
    export SETUP_FILE_SCRIPT_DIR="$SCRIPT_DIR"
}

# Lightweight per-test setup
setup() {
    # Setup standard mocks
    vrooli_auto_setup
    
    # Use paths from setup_file
    SCRIPT_DIR="${SETUP_FILE_SCRIPT_DIR}"
}

# BATS teardown function - runs after each test
teardown() {
    vrooli_cleanup_test
}

# Test script loading
@test "common.sh loads without errors" {
    # The script should source successfully in setup_file
    [ "$?" -eq 0 ]
}

# Test Docker check function exists
@test "whisper::check_docker function is defined" {
    declare -f whisper::check_docker >/dev/null
}

# Test container existence check function exists
@test "whisper::container_exists function is defined" {
    declare -f whisper::container_exists >/dev/null
}

# Test running status check function exists
@test "whisper::is_running function is defined" {
    declare -f whisper::is_running >/dev/null
}

# Test model validation function
@test "whisper::validate_model function works" {
    # Test valid models
    run whisper::validate_model "tiny"
    [ "$status" -eq 0 ]
    
    run whisper::validate_model "base"
    [ "$status" -eq 0 ]
    
    run whisper::validate_model "large"
    [ "$status" -eq 0 ]
}

# Test invalid model validation
@test "whisper::validate_model rejects invalid models" {
    run whisper::validate_model "invalid"
    [ "$status" -eq 1 ]
    
    run whisper::validate_model ""
    [ "$status" -eq 1 ]
}

# Test model size function
@test "whisper::get_model_size function works" {
    run whisper::get_model_size "tiny"
    [[ "$output" =~ ^[0-9]+(\.[0-9]+)?$ ]]
    
    run whisper::get_model_size "invalid"
    [ "$output" = "unknown" ]
}

# Test directory creation function exists
@test "whisper::create_directories function is defined" {
    declare -f whisper::create_directories >/dev/null
}

# Test health check function exists
@test "whisper::is_healthy function is defined" {
    declare -f whisper::is_healthy >/dev/null
}

# Test GPU availability function exists
@test "whisper::is_gpu_available function is defined" {
    declare -f whisper::is_gpu_available >/dev/null
}

# Test Docker image selection function
@test "whisper::get_docker_image function works" {
    # Test CPU image selection (default)
    run whisper::get_docker_image
    [ "$output" = "$WHISPER_CPU_IMAGE" ]
}

# Test cleanup function exists
@test "whisper::cleanup function is defined" {
    declare -f whisper::cleanup >/dev/null
}

# Test port availability function exists
@test "whisper::is_port_available function is defined" {
    declare -f whisper::is_port_available >/dev/null
}

# Test wait for health function exists
@test "whisper::wait_for_health function is defined" {
    declare -f whisper::wait_for_health >/dev/null
}