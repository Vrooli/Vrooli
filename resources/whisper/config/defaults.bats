#!/usr/bin/env bats
# Tests for Whisper defaults.sh configuration

# Setup for each test
setup() {
    # Set test environment
    export WHISPER_CUSTOM_PORT="9090"
    export GPU="yes"
    export WHISPER_DEFAULT_MODEL="medium"
    
    # Mock resources function
    resources::get_default_port() {
        echo "8090"
    }
    
    # Load the defaults
    SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    source "${SCRIPT_DIR}/defaults.sh"
}

# Test configuration export
@test "whisper::export_config sets all required variables" {
    whisper::export_config
    
    # Test basic configuration
    [ "$WHISPER_PORT" = "9090" ]  # Custom port should override default
    [ "$WHISPER_BASE_URL" = "http://localhost:9090" ]
    [ "$WHISPER_CONTAINER_NAME" = "whisper" ]
    [ "$WHISPER_DATA_DIR" = "${HOME}/.whisper" ]
    [ "$WHISPER_MODELS_DIR" = "${HOME}/.whisper/models" ]
    [ "$WHISPER_UPLOADS_DIR" = "${HOME}/.whisper/uploads" ]
}

@test "whisper::export_config sets docker image configuration" {
    whisper::export_config
    
    [ -n "$WHISPER_IMAGE" ]
    [ -n "$WHISPER_CPU_IMAGE" ]
    [[ "$WHISPER_IMAGE" == *"whisper-asr-webservice"* ]]
    [[ "$WHISPER_CPU_IMAGE" == *"whisper-asr-webservice"* ]]
}

@test "whisper::export_config sets model configuration" {
    whisper::export_config
    
    [ "$WHISPER_DEFAULT_MODEL" = "medium" ]  # Should use our custom value
    [ -n "$WHISPER_MODEL_SIZE_TINY" ]
    [ -n "$WHISPER_MODEL_SIZE_BASE" ]
    [ -n "$WHISPER_MODEL_SIZE_SMALL" ]
    [ -n "$WHISPER_MODEL_SIZE_MEDIUM" ]
    [ -n "$WHISPER_MODEL_SIZE_LARGE" ]
}

@test "whisper::export_config sets health check configuration" {
    whisper::export_config
    
    [ "$WHISPER_HEALTH_CHECK_INTERVAL" = "5" ]
    [ "$WHISPER_HEALTH_CHECK_MAX_ATTEMPTS" = "12" ]
    [ "$WHISPER_API_TIMEOUT" = "10" ]
}

@test "whisper::export_config sets timeout configuration" {
    whisper::export_config
    
    [ "$WHISPER_STARTUP_MAX_WAIT" = "120" ]
    [ "$WHISPER_STARTUP_WAIT_INTERVAL" = "5" ]
    [ "$WHISPER_INITIALIZATION_WAIT" = "30" ]
}

@test "whisper::export_config handles GPU configuration" {
    whisper::export_config
    
    [ "$WHISPER_GPU_ENABLED" = "yes" ]  # Should use our GPU env var
}

@test "whisper::export_config is idempotent" {
    whisper::export_config
    local first_port="$WHISPER_PORT"
    
    # Change environment variable
    export WHISPER_CUSTOM_PORT="9091"
    
    whisper::export_config
    
    # Should still have the original value (idempotent)
    [ "$WHISPER_PORT" = "$first_port" ]
}

@test "whisper::export_config exports all variables" {
    whisper::export_config
    
    # Test that key variables are exported (accessible in subshells)
    ([ -n "$WHISPER_PORT" ])
    ([ -n "$WHISPER_BASE_URL" ])
    ([ -n "$WHISPER_CONTAINER_NAME" ])
    ([ -n "$WHISPER_DATA_DIR" ])
    ([ -n "$WHISPER_IMAGE" ])
    ([ -n "$WHISPER_DEFAULT_MODEL" ])
}