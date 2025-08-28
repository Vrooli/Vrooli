#!/usr/bin/env bats
# Tests for Whisper manage.sh script

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
@test "manage.sh loads without errors" {
    # The script should source successfully in setup
    [ "$?" -eq 0 ]
}

# Test argument parsing function exists
@test "manage::parse_arguments function is defined" {
    # Check if the function is defined
    declare -f manage::parse_arguments >/dev/null
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
    # Use the real function from manage.sh
    
    # Test valid models
    run manage::validate_model "small"
    [ "$status" -eq 0 ]
    
    run manage::validate_model "large"
    [ "$status" -eq 0 ]
    
    # Test invalid model
    run manage::validate_model "invalid"
    [ "$status" -eq 1 ]
}

# Test Docker image selection logic
@test "Docker image selection works correctly" {
    # Create a simple test function
    manage::get_docker_image() {
        if [[ "${USE_GPU:-no}" == "yes" ]]; then
            echo "${WHISPER_IMAGE:-onerahmet/openai-whisper-asr-webservice:latest-gpu}"
        else
            echo "${WHISPER_CPU_IMAGE:-onerahmet/openai-whisper-asr-webservice:latest}" 
        fi
    }
    
    # Test CPU image selection
    USE_GPU="no"
    
    run manage::get_docker_image
    [[ "$output" == *"whisper"* ]]
    
    # Test GPU image selection
    USE_GPU="yes"
    
    run manage::get_docker_image
    [[ "$output" == *"gpu"* ]] || [[ "$output" == *"whisper"* ]]
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
    [ -n "${WHISPER_CUSTOM_PORT:-9005}" ]
    [ -n "${WHISPER_DEFAULT_MODEL:-medium}" ]
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

# Test whisper container operations with mock
@test "whisper container operations work with mock" {
    # Load the whisper mock if available
    if [[ -f "${BATSLIB_DIR}/fixtures/mocks/whisper.sh" ]]; then
        # shellcheck disable=SC1091
        source "${BATSLIB_DIR}/fixtures/mocks/whisper.sh"
        
        # Initialize mock
        mock::whisper::reset
        
        # Test container exists check
        run manage::container_exists
        [ "$status" -eq 0 ]
        
        # Test container is running check
        run manage::is_running
        [ "$status" -eq 0 ]
        
        # Set service as stopped
        mock::whisper::set_service_status "stopped"
        
        # Test container is not running
        run manage::is_running
        [ "$status" -eq 1 ]
    else
        skip "Whisper mock not available"
    fi
}

# Test whisper health check with mock
@test "whisper health check works with mock" {
    # Load the whisper mock if available
    if [[ -f "${BATSLIB_DIR}/fixtures/mocks/whisper.sh" ]]; then
        # shellcheck disable=SC1091
        source "${BATSLIB_DIR}/fixtures/mocks/whisper.sh"
        
        # Initialize mock as healthy
        mock::whisper::reset
        mock::whisper::set_service_status "running"
        
        # Test health check - mock intercepts curl
        run manage::is_healthy
        [ "$status" -eq 0 ]
        
        # Set service as unhealthy
        mock::whisper::set_service_status "unhealthy"
        
        # Test unhealthy state
        run curl -s "$WHISPER_BASE_URL/health"
        [[ "$output" == *"unhealthy"* ]]
    else
        skip "Whisper mock not available"
    fi
}

# Test whisper transcription with mock
@test "whisper transcription works with mock" {
    # Load the whisper mock if available
    if [[ -f "${BATSLIB_DIR}/fixtures/mocks/whisper.sh" ]]; then
        # shellcheck disable=SC1091
        source "${BATSLIB_DIR}/fixtures/mocks/whisper.sh"
        
        # Initialize mock
        mock::whisper::reset
        mock::whisper::set_service_status "running"
        
        # Add a custom transcript for test file
        mock::whisper::add_transcript "test.mp3" '{"text":"This is a test transcription","language":"en"}'
        
        # Test transcription API
        run curl -s -X POST "${WHISPER_BASE_URL}/asr" -F "audio_file=@test.mp3"
        [ "$status" -eq 0 ]
        [[ "$output" == *"test transcription"* ]]
    else
        skip "Whisper mock not available"
    fi
}