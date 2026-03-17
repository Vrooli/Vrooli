#!/usr/bin/env bats
# Tests for Kokoro API functions

# Source trash module for safe test cleanup
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/resources/kokoro/lib"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Load Vrooli test infrastructure (REQUIRED)
source "${BATS_TEST_DIRNAME}/../../../../__test/fixtures/setup.bash"

# Expensive setup operations (run once per file)
setup_file() {
    # Use appropriate setup function
    vrooli_setup_service_test "kokoro"

    # Load dependencies once
    SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    KOKORO_DIR="$(dirname "$SCRIPT_DIR")"

    # Source necessary files
    source "${KOKORO_DIR}/config/defaults.sh"
    source "${KOKORO_DIR}/config/messages.sh"

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
    export KOKORO_CONTAINER_NAME="kokoro-test"
    export KOKORO_BASE_URL="http://localhost:9090"
    export KOKORO_API_TIMEOUT="10"
    export KOKORO_DEFAULT_VOICE="af_heart"

    # Export config functions
    defaults::export_config
    messages::export_messages

    # Mock health check
    kokoro::is_healthy() { return 0; }
}

# BATS teardown function - runs after each test
teardown() {
    vrooli_cleanup_test

    # Clean up any test files
    trash::safe_remove "/tmp/kokoro_test_*" --test-cleanup
}

# === API Health Check Tests ===
@test "kokoro::test_api checks service health" {
    # Mock successful health check
    curl() {
        if [[ "$*" == *"/v1/audio/voices"* ]]; then
            echo '["af_heart","af_bella","am_adam"]'
            return 0
        fi
        return 1
    }

    run kokoro::test_api
    [ "$status" -eq 0 ]
    [[ "$output" == *"accessible"* ]]
}

@test "kokoro::test_api fails when service unhealthy" {
    # Mock failed health check
    kokoro::is_healthy() { return 1; }

    run kokoro::test_api
    [ "$status" -eq 1 ]
}

# === Synthesis API Tests ===
@test "kokoro::synthesize_text handles successful synthesis" {
    local test_file="/tmp/kokoro_test_output.mp3"

    # Mock successful synthesis
    curl() {
        if [[ "$*" == *"/v1/audio/speech"* ]]; then
            # Create a fake audio file
            local output_arg=""
            for arg in "$@"; do
                if [[ "$prev_arg" == "-o" ]]; then
                    output_arg="$arg"
                    break
                fi
                prev_arg="$arg"
            done
            if [[ -n "$output_arg" ]]; then
                echo "fake audio data" > "$output_arg"
            fi
            echo "200"
            return 0
        fi
        return 1
    }

    run kokoro::synthesize_text "Hello world" "$test_file"
    [ "$status" -eq 0 ]
    [[ "$output" == *"completed"* ]] || [[ "$output" == *"Synthesis"* ]]
}

@test "kokoro::synthesize_text validates text input" {
    run kokoro::synthesize_text ""
    [ "$status" -eq 1 ]
    [[ "$output" == *"No text"* ]] || [[ "$output" == *"text"* ]]
}

# === Voice Listing Tests ===
@test "kokoro::list_voices returns available voices" {
    # Mock voice list response
    curl() {
        if [[ "$*" == *"/v1/audio/voices"* ]]; then
            echo '["af_heart","af_bella","am_adam","bf_emma"]'
            return 0
        fi
        return 1
    }

    run kokoro::list_voices
    [ "$status" -eq 0 ]
    [[ "$output" == *"af_heart"* ]] || [[ "$output" == *"voice"* ]]
}

# === API Info Tests ===
@test "kokoro::get_api_info returns API information" {
    run kokoro::get_api_info
    [ "$status" -eq 0 ]
    [[ "$output" == *"Kokoro"* ]]
    [[ "$output" == *"/v1/audio/speech"* ]]
    [[ "$output" == *"/v1/audio/voices"* ]]
    [[ "$output" == *"mp3"* ]]
}

# === Error Handling Tests ===
@test "kokoro::test_api handles timeout gracefully" {
    # Mock timeout
    curl() {
        sleep 2  # Simulate slow response
        return 124  # Timeout exit code
    }
    export KOKORO_API_TIMEOUT="1"

    run kokoro::test_api
    [ "$status" -eq 1 ]
}
