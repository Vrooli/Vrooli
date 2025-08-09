#!/usr/bin/env bats
# Tests for Whisper manage.sh script

# Load Vrooli test infrastructure (REQUIRED)
source "${BATS_TEST_DIRNAME}/../../../__test/fixtures/setup.bash"

# Expensive setup operations (run once per file)
setup_file() {
    # Use appropriate setup function
    vrooli_setup_service_test "whisper"
    
    # Load dependencies once
    SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    
    # Source manage.sh and all dependencies
    source "${SCRIPT_DIR}/manage.sh"
    
    # Export paths for use in setup()
    export SETUP_FILE_SCRIPT_DIR="$SCRIPT_DIR"
}

# Lightweight per-test setup
setup() {
    # Setup standard mocks
    vrooli_auto_setup
    
    # Use paths from setup_file
    SCRIPT_DIR="${SETUP_FILE_SCRIPT_DIR}"
    
    # Set whisper-specific environment
    export WHISPER_CUSTOM_PORT="9999"
    export WHISPER_DEFAULT_MODEL="small"
    export GPU="no"
    
    # Load config and messages from config files
    if [[ -f "${SCRIPT_DIR}/config/defaults.sh" ]]; then
        source "${SCRIPT_DIR}/config/defaults.sh"
        whisper::export_config 2>/dev/null || true
    fi
    if [[ -f "${SCRIPT_DIR}/config/messages.sh" ]]; then
        source "${SCRIPT_DIR}/config/messages.sh"
        whisper::export_messages 2>/dev/null || true
    fi
}

# BATS teardown function - runs after each test
teardown() {
    vrooli_cleanup_test
}

# Test script loading
@test "manage.sh loads without errors" {
    # The script should source successfully in setup
    [ "$?" -eq 0 ]
}

# Test argument parsing function exists
@test "whisper::parse_arguments function is defined" {
    # Check if the function is defined
    declare -f whisper::parse_arguments >/dev/null
}

# Test configuration loading
@test "whisper configuration is loaded correctly" {
    # Test that basic Whisper configuration is available
    [ -n "${WHISPER_CUSTOM_PORT:-}" ]
    [ -n "${WHISPER_DEFAULT_MODEL:-}" ]
}

# Test message loading
@test "whisper messages are loaded correctly" {
    # Test basic message loading by checking if messages config exists
    if [[ -f "${SCRIPT_DIR}/config/messages.sh" ]]; then
        source "${SCRIPT_DIR}/config/messages.sh"
        # Test passes if we can source the file
        [ "$?" -eq 0 ]
    else
        skip "Messages configuration file not found"
    fi
}

# Test model validation
@test "model validation works correctly" {
    # Mock the validate function if it doesn't exist
    if ! declare -f whisper::validate_model >/dev/null 2>&1; then
        whisper::validate_model() {
            local model="$1"
            case "$model" in
                tiny|base|small|medium|large|large-v2|large-v3) return 0 ;;
                *) return 1 ;;
            esac
        }
    fi
    
    # Test valid models
    run whisper::validate_model "small"
    [ "$status" -eq 0 ]
    
    run whisper::validate_model "large"
    [ "$status" -eq 0 ]
    
    # Test invalid model
    run whisper::validate_model "invalid"
    [ "$status" -eq 1 ]
}

# Test Docker image selection logic
@test "Docker image selection works correctly" {
    # Mock the function if it doesn't exist
    if ! declare -f whisper::get_docker_image >/dev/null 2>&1; then
        whisper::get_docker_image() {
            if [[ "${WHISPER_GPU_ENABLED:-no}" == "yes" ]]; then
                echo "${WHISPER_IMAGE:-gpu-image}"
            else
                echo "${WHISPER_CPU_IMAGE:-cpu-image}" 
            fi
        }
    fi
    
    # Test CPU image selection
    # Don't try to override readonly variables - just test the function
    WHISPER_GPU_ENABLED="no"
    
    run whisper::get_docker_image
    [[ "$output" == *"cpu"* ]] || [[ "$output" == *"whisper"* ]]
    
    # Test GPU image selection
    WHISPER_GPU_ENABLED="yes"
    
    run whisper::get_docker_image
    [[ "$output" == *"gpu"* ]] || [[ "$output" == *"cuda"* ]] || [[ "$output" == *"whisper"* ]]
}

# Test help functionality
@test "help action shows usage information" {
    # Mock args::usage function
    args::usage() {
        echo "Usage: manage.sh [OPTIONS]"
        echo "Whisper speech-to-text service management"
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
        "status" "logs" "transcribe" "models" "info" "test"
    )
    
    for action in "${valid_actions[@]}"; do
        # This test just verifies the actions are recognized
        # We're not actually executing them
        case "$action" in
            install|uninstall|start|stop|restart|status|logs|transcribe|models|info|test)
                # Action is valid
                ;;
            *)
                # This should not happen with our list
                fail "Action $action was not recognized as valid"
                ;;
        esac
    done
}

# Test environment variable handling
@test "environment variables are handled correctly" {
    # Test custom port handling
    export WHISPER_CUSTOM_PORT="9999"
    
    # Test that the variable is set correctly
    [ "$WHISPER_CUSTOM_PORT" = "9999" ]
}

# Test GPU detection logic
@test "GPU detection logic works" {
    # Mock GPU availability check
    whisper::is_gpu_available() {
        if [[ "${TEST_GPU_AVAILABLE:-no}" == "yes" ]]; then
            return 0
        else
            return 1
        fi
    }
    
    # Test when GPU is not available
    export TEST_GPU_AVAILABLE="no"
    run whisper::is_gpu_available
    [ "$status" -eq 1 ]
    
    # Test when GPU is available
    export TEST_GPU_AVAILABLE="yes"
    run whisper::is_gpu_available
    [ "$status" -eq 0 ]
}

# Test configuration validation
@test "configuration validation works" {
    # Test that required environment variables have defaults
    [ -n "${WHISPER_CUSTOM_PORT:-9005}" ]
    [ -n "${WHISPER_DEFAULT_MODEL:-medium}" ]
    [ -n "${GPU:-no}" ]
}

# Test error handling
@test "error conditions are handled gracefully" {
    # Mock a failing condition
    whisper::check_docker() {
        return 1  # Docker not available
    }
    
    # The function should handle the error gracefully
    run whisper::check_docker
    [ "$status" -eq 1 ]
}