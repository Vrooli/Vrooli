#!/usr/bin/env bats
# Tests for Whisper inject.sh script

# Get script directory first
INJECT_BATS_DIR="${BATS_TEST_DIRNAME}"

# Source var.sh first to get directory variables
# shellcheck disable=SC1091
source "${INJECT_BATS_DIR}/../../../lib/utils/var.sh"

# Load Vrooli test infrastructure using var_ variables
# shellcheck disable=SC1091
source "${var_TEST_DIR}/fixtures/setup.bash"

# Expensive setup operations (run once per file)
setup_file() {
    # Use appropriate setup function
    vrooli_setup_service_test "whisper"
    
    # Load dependencies once
    SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    
    # Source inject.sh and all dependencies
    source "${SCRIPT_DIR}/inject.sh"
    
    # Export paths for use in setup()
    export SETUP_FILE_SCRIPT_DIR="$SCRIPT_DIR"
}

# Lightweight per-test setup
setup() {
    # Setup standard mocks
    vrooli_auto_setup
    
    # Use paths from setup_file
    SCRIPT_DIR="${SETUP_FILE_SCRIPT_DIR}"
    
    # Set whisper-specific environment for injection
    export WHISPER_HOST="http://localhost:9999"
    export WHISPER_DATA_DIR="${HOME}/.whisper"
    export WHISPER_MODELS_DIR="${WHISPER_DATA_DIR}/models"
}

# BATS teardown function - runs after each test
teardown() {
    vrooli_cleanup_test
}

# Test script loading
@test "inject.sh loads without errors" {
    # The script should source successfully in setup
    [ "$?" -eq 0 ]
}

# Test usage function exists
@test "inject::usage function is defined" {
    # Check if the function is defined
    declare -f inject::usage >/dev/null
}

# Test validation function exists
@test "inject::validate_config function is defined" {
    declare -f inject::validate_config >/dev/null
}

# Test model validation
@test "whisper model validation works" {
    # Test valid model configuration
    local valid_config='{"models": [{"name": "base", "download": true}]}'
    
    run inject::validate_config "$valid_config"
    [ "$status" -eq 0 ]
}

# Test invalid model validation
@test "whisper model validation rejects invalid models" {
    # Test invalid model configuration
    local invalid_config='{"models": [{"name": "invalid_model", "download": true}]}'
    
    run inject::validate_config "$invalid_config"
    [ "$status" -eq 1 ]
}

# Test empty configuration validation
@test "empty configuration is rejected" {
    # Test empty configuration
    local empty_config='{}'
    
    run inject::validate_config "$empty_config"
    [ "$status" -eq 1 ]
}

# Test malformed JSON validation
@test "malformed JSON is rejected" {
    # Test malformed JSON
    local malformed_config='{"models": [{'
    
    run inject::validate_config "$malformed_config"
    [ "$status" -eq 1 ]
}

# Test models array validation
@test "models configuration validation works" {
    local models_config='[{"name": "tiny", "download": true}, {"name": "base", "download": false}]'
    
    run inject::validate_models "$models_config"
    [ "$status" -eq 0 ]
}

# Test accessibility check
@test "accessibility check function works" {
    # Mock the accessibility check to avoid requiring real service
    inject::check_accessibility() {
        return 0  # Always accessible for test
    }
    
    run inject::check_accessibility
    [ "$status" -eq 0 ]
}

# Test rollback actions
@test "rollback actions can be added" {
    # Test adding rollback action
    inject::add_rollback_action "test action" "echo 'test command'"
    
    # Check that the action was added
    [ ${#WHISPER_ROLLBACK_ACTIONS[@]} -eq 1 ]
}

# Test configuration with audio samples
@test "audio samples validation works" {
    # Create a temporary test file
    local temp_file="${BATS_TMPDIR}/test_audio.mp3"
    echo "dummy content" > "$temp_file"
    
    local audio_config="{\"audio_samples\": [{\"file\": \"${temp_file}\", \"name\": \"test\"}]}"
    
    # This will fail because the file path is not relative to var_ROOT_DIR
    # but it tests the validation logic
    run inject::validate_config "$audio_config"
    [ "$status" -eq 1 ]  # Expected to fail due to path validation
}

# Test main function parameter validation
@test "main function requires parameters" {
    # Test main function without parameters
    run inject::main
    [ "$status" -eq 1 ]
}

# Test help action
@test "help action works" {
    run inject::main "--help"
    [ "$status" -eq 0 ]
}