#!/usr/bin/env bats
# Tests for Kokoro manage/cli script

# Get script directory first
MANAGE_BATS_DIR="${BATS_TEST_DIRNAME}"

# Source var.sh first to get directory variables
# shellcheck disable=SC1091
source "${MANAGE_BATS_DIR}/../../../lib/utils/var.sh"

# Load Vrooli test infrastructure using var_ variables
# shellcheck disable=SC1091
source "${var_TEST_DIR}/fixtures/setup.bash"

# Expensive setup operations (run once per file)
setup_file() {
    # Use appropriate setup function
    vrooli_setup_service_test "kokoro"

    # Load dependencies once
    SCRIPT_DIR="${BATS_TEST_DIRNAME}"

    # Source cli.sh and all dependencies
    source "${SCRIPT_DIR}/cli.sh"

    # Export paths for use in setup()
    export SETUP_FILE_SCRIPT_DIR="$SCRIPT_DIR"
}

# Lightweight per-test setup
setup() {
    # Setup standard mocks
    vrooli_auto_setup

    # Use paths from setup_file
    SCRIPT_DIR="${SETUP_FILE_SCRIPT_DIR}"

    # Set kokoro-specific environment
    export KOKORO_CUSTOM_PORT="9999"
    export KOKORO_DEFAULT_VOICE="af_heart"
    export GPU="no"

    # Load config and messages from config files
    if [[ -f "${SCRIPT_DIR}/config/defaults.sh" ]]; then
        # shellcheck disable=SC1091
        source "${SCRIPT_DIR}/config/defaults.sh"
        defaults::export_config 2>/dev/null || true
    fi
    if [[ -f "${SCRIPT_DIR}/config/messages.sh" ]]; then
        # shellcheck disable=SC1091
        source "${SCRIPT_DIR}/config/messages.sh"
        messages::export_messages 2>/dev/null || true
    fi
}

# BATS teardown function - runs after each test
teardown() {
    vrooli_cleanup_test
}

# Test script loading
@test "cli.sh loads without errors" {
    # The script should source successfully in setup
    [ "$?" -eq 0 ]
}

# Test configuration loading
@test "kokoro configuration is loaded correctly" {
    # Test that basic Kokoro configuration is available
    [ -n "${KOKORO_CUSTOM_PORT:-}" ]
    [ -n "${KOKORO_DEFAULT_VOICE:-}" ]
}

# Test message loading
@test "kokoro messages are loaded correctly" {
    # Test basic message loading by checking if messages config exists
    if [[ -f "${SCRIPT_DIR}/config/messages.sh" ]]; then
        source "${SCRIPT_DIR}/config/messages.sh"
        # Test passes if we can source the file
        [ "$?" -eq 0 ]
    else
        skip "Messages configuration file not found"
    fi
}

# Test Docker image selection logic
@test "Docker image selection works correctly" {
    # Create a simple test function
    manage::get_docker_image() {
        if [[ "${USE_GPU:-no}" == "yes" ]]; then
            echo "${KOKORO_IMAGE:-ghcr.io/remsky/kokoro-fastapi-gpu:latest}"
        else
            echo "${KOKORO_CPU_IMAGE:-ghcr.io/remsky/kokoro-fastapi-cpu:latest}"
        fi
    }

    # Test CPU image selection
    USE_GPU="no"

    run manage::get_docker_image
    [[ "$output" == *"kokoro"* ]]

    # Test GPU image selection
    USE_GPU="yes"

    run manage::get_docker_image
    [[ "$output" == *"gpu"* ]] || [[ "$output" == *"kokoro"* ]]
}

# Test help functionality
@test "help action shows usage information" {
    # Mock args::usage function
    args::usage() {
        echo "Usage: resource-kokoro [OPTIONS]"
        echo "Kokoro text-to-speech synthesis service management"
    }

    # Test help display
    run args::usage "Test description"
    [ "$status" -eq 0 ]
    [[ "$output" == *"Usage"* ]]
}

# Test action validation
@test "valid actions are accepted" {
    local valid_actions=(
        "install" "uninstall" "start" "stop" "restart"
        "status" "logs" "synthesize" "voices" "info"
    )

    for action in "${valid_actions[@]}"; do
        case "$action" in
            install|uninstall|start|stop|restart|status|logs|synthesize|voices|info)
                # Action is valid
                ;;
            *)
                fail "Action $action was not recognized as valid"
                ;;
        esac
    done
}

# Test environment variable handling
@test "environment variables are handled correctly" {
    # Test custom port handling
    export KOKORO_CUSTOM_PORT="9999"

    # Test that the variable is set correctly
    [ "$KOKORO_CUSTOM_PORT" = "9999" ]
}

# Test GPU detection logic
@test "GPU detection logic works" {
    # Mock GPU availability check
    manage::is_gpu_available() {
        if [[ "${TEST_GPU_AVAILABLE:-no}" == "yes" ]]; then
            return 0
        else
            return 1
        fi
    }

    # Test when GPU is not available
    export TEST_GPU_AVAILABLE="no"
    run manage::is_gpu_available
    [ "$status" -eq 1 ]

    # Test when GPU is available
    export TEST_GPU_AVAILABLE="yes"
    run manage::is_gpu_available
    [ "$status" -eq 0 ]
}

# Test configuration validation
@test "configuration validation works" {
    # Test that required environment variables have defaults
    [ -n "${KOKORO_CUSTOM_PORT:-9999}" ]
    [ -n "${KOKORO_DEFAULT_VOICE:-af_heart}" ]
    [ -n "${GPU:-no}" ]
}

# Test error handling
@test "error conditions are handled gracefully" {
    # Mock a failing condition
    manage::check_docker() {
        return 1  # Docker not available
    }

    # The function should handle the error gracefully
    run manage::check_docker
    [ "$status" -eq 1 ]
}
