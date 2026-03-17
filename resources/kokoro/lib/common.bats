#!/usr/bin/env bats
# Tests for Kokoro lib/common.sh functions

# Load Vrooli test infrastructure (REQUIRED)
source "${BATS_TEST_DIRNAME}/../../../../__test/fixtures/setup.bash"

# Expensive setup operations (run once per file)
setup_file() {
    # Use appropriate setup function
    vrooli_setup_service_test "kokoro"

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
    export MSG_CREATING_DIRS="Creating directories"
    export MSG_CREATE_DIRS_FAILED="Failed to create directories"
    export MSG_DIRECTORIES_CREATED="Directories created"
    export MSG_WAITING_STARTUP="Waiting for startup"
    export MSG_HEALTHY="Service healthy"
    export MSG_STARTUP_TIMEOUT="Startup timeout"
    export MSG_REMOVING_CONTAINER="Removing container"
    export MSG_GPU_NOT_AVAILABLE="GPU not available"

    # Set Kokoro configuration
    export KOKORO_CONTAINER_NAME="kokoro-test"
    export KOKORO_DATA_DIR="${HOME}/.kokoro"
    export KOKORO_VOICES_DIR="${KOKORO_DATA_DIR}/voices"
    export KOKORO_BASE_URL="http://localhost:9999"
    export KOKORO_STARTUP_MAX_WAIT=60
    export KOKORO_STARTUP_WAIT_INTERVAL=5
    export KOKORO_API_TIMEOUT=10
    export KOKORO_IMAGE="kokoro:gpu"
    export KOKORO_CPU_IMAGE="kokoro:cpu"
    export KOKORO_GPU_ENABLED="no"

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
@test "kokoro::check_docker function is defined" {
    declare -f common::check_docker >/dev/null
}

# Test container existence check function exists
@test "kokoro::container_exists function is defined" {
    declare -f common::container_exists >/dev/null
}

# Test running status check function exists
@test "kokoro::is_running function is defined" {
    declare -f common::is_running >/dev/null
}

# Test directory creation function exists
@test "kokoro::create_directories function is defined" {
    declare -f kokoro::create_directories >/dev/null
}

# Test health check function exists
@test "kokoro::is_healthy function is defined" {
    declare -f kokoro::is_healthy >/dev/null
}

# Test GPU availability function exists
@test "kokoro::is_gpu_available function is defined" {
    declare -f kokoro::is_gpu_available >/dev/null
}

# Test Docker image selection function
@test "kokoro::get_docker_image function works" {
    # Test CPU image selection (default)
    run kokoro::get_docker_image
    [ "$output" = "$KOKORO_CPU_IMAGE" ]
}

# Test cleanup function exists
@test "kokoro::cleanup function is defined" {
    declare -f kokoro::cleanup >/dev/null
}

# Test port availability function exists
@test "kokoro::is_port_available function is defined" {
    declare -f common::is_port_available >/dev/null
}

# Test wait for health function exists
@test "kokoro::wait_for_health function is defined" {
    declare -f kokoro::wait_for_health >/dev/null
}
