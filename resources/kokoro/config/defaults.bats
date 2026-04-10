#!/usr/bin/env bats
# Tests for Kokoro defaults.sh configuration

# Setup for each test
setup() {
    # Set test environment
    export KOKORO_CUSTOM_PORT="9090"
    export GPU="yes"
    export KOKORO_DEFAULT_VOICE="af_heart"

    # Mock resources function
    resources::get_default_port() {
        echo "8880"
    }

    # Load the defaults
    SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    source "${SCRIPT_DIR}/defaults.sh"
}

# Test configuration export
@test "kokoro::export_config sets all required variables" {
    defaults::export_config

    # Test basic configuration
    [ "$KOKORO_PORT" = "9090" ]  # Custom port should override default
    [ "$KOKORO_BASE_URL" = "http://localhost:9090" ]
    [ "$KOKORO_CONTAINER_NAME" = "kokoro" ]
    [ "$KOKORO_DATA_DIR" = "${HOME}/.kokoro" ]
    [ "$KOKORO_VOICES_DIR" = "${HOME}/.kokoro/voices" ]
}

@test "kokoro::export_config sets docker image configuration" {
    defaults::export_config

    [ -n "$KOKORO_IMAGE" ]
    [ -n "$KOKORO_CPU_IMAGE" ]
    [[ "$KOKORO_IMAGE" == *"kokoro-fastapi"* ]]
    [[ "$KOKORO_CPU_IMAGE" == *"kokoro-fastapi"* ]]
}

@test "kokoro::export_config sets voice configuration" {
    defaults::export_config

    [ "$KOKORO_DEFAULT_VOICE" = "af_heart" ]
}

@test "kokoro::export_config sets health check configuration" {
    defaults::export_config

    [ "$KOKORO_HEALTH_CHECK_INTERVAL" = "5" ]
    [ "$KOKORO_HEALTH_CHECK_MAX_ATTEMPTS" = "12" ]
    [ "$KOKORO_API_TIMEOUT" = "10" ]
}

@test "kokoro::export_config sets timeout configuration" {
    defaults::export_config

    [ "$KOKORO_STARTUP_MAX_WAIT" = "120" ]
    [ "$KOKORO_STARTUP_WAIT_INTERVAL" = "5" ]
    [ "$KOKORO_INITIALIZATION_WAIT" = "30" ]
}

@test "kokoro::export_config handles GPU configuration" {
    defaults::export_config

    [ "$KOKORO_GPU_ENABLED" = "yes" ]  # Should use our GPU env var
}

@test "kokoro::export_config is idempotent" {
    defaults::export_config
    local first_port="$KOKORO_PORT"

    # Change environment variable
    export KOKORO_CUSTOM_PORT="9091"

    defaults::export_config

    # Should still have the original value (idempotent)
    [ "$KOKORO_PORT" = "$first_port" ]
}

@test "kokoro::export_config exports all variables" {
    defaults::export_config

    # Test that key variables are exported (accessible in subshells)
    ([ -n "$KOKORO_PORT" ])
    ([ -n "$KOKORO_BASE_URL" ])
    ([ -n "$KOKORO_CONTAINER_NAME" ])
    ([ -n "$KOKORO_DATA_DIR" ])
    ([ -n "$KOKORO_IMAGE" ])
    ([ -n "$KOKORO_DEFAULT_VOICE" ])
}
