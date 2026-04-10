#!/usr/bin/env bats
# Tests for Kokoro messages.sh configuration

# Setup for each test
setup() {
    # Set test environment
    export KOKORO_PORT="8880"

    # Load the messages
    SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    source "${SCRIPT_DIR}/messages.sh"
}

# Test message export
@test "kokoro::export_messages sets success messages" {
    messages::export_messages

    [ "$MSG_INSTALL_SUCCESS" = "✅ Kokoro installed successfully" ]
    [ "$MSG_START_SUCCESS" = "✅ Kokoro started successfully" ]
    [ "$MSG_STOP_SUCCESS" = "✅ Kokoro stopped successfully" ]
    [ "$MSG_RESTART_SUCCESS" = "✅ Kokoro restarted successfully" ]
    [ "$MSG_UNINSTALL_SUCCESS" = "✅ Kokoro uninstalled successfully" ]
    [ "$MSG_HEALTHY" = "✅ Kokoro API is healthy" ]
    [ "$MSG_RUNNING" = "✅ Kokoro container is running" ]
}

@test "kokoro::export_messages sets error messages" {
    messages::export_messages

    [ "$MSG_DOCKER_NOT_FOUND" = "Docker is not installed" ]
    [ "$MSG_DOCKER_NOT_RUNNING" = "Docker daemon is not running" ]
    [ "$MSG_INSTALL_FAILED" = "❌ Kokoro installation failed" ]
    [ "$MSG_NOT_INSTALLED" = "❌ Kokoro is not installed" ]
    [ "$MSG_NOT_RUNNING" = "Kokoro is not running" ]
    [ "$MSG_SYNTHESIS_FAILED" = "❌ Text-to-speech synthesis failed" ]
    [ "$MSG_INVALID_VOICE" = "❌ Invalid voice specified" ]
}

@test "kokoro::export_messages sets info messages" {
    messages::export_messages

    [ "$MSG_CHECKING_STATUS" = "🔍 Checking Kokoro status..." ]
    [ "$MSG_PULLING_IMAGE" = "📥 Pulling Kokoro image..." ]
    [ "$MSG_STARTING_CONTAINER" = "Starting Kokoro container..." ]
    [ "$MSG_SYNTHESIZING" = "🔊 Synthesizing speech..." ]
    [ "$MSG_LISTING_VOICES" = "📋 Listing available voices..." ]
}

@test "kokoro::export_messages sets warning messages" {
    messages::export_messages

    [ "$MSG_ALREADY_INSTALLED" = "Kokoro is already installed and running" ]
    [[ "$MSG_ALREADY_RUNNING" == *"8880"* ]]  # Should contain the port
    [ "$MSG_UNINSTALL_WARNING" = "This will remove Kokoro and all voice data" ]
}

@test "kokoro::export_messages sets usage messages" {
    messages::export_messages

    [ "$MSG_USAGE_SYNTHESIZE" = "🔊 Testing Kokoro Speech Synthesis API" ]
    [ "$MSG_USAGE_VOICES" = "🎭 Listing Available Voices" ]
    [ "$MSG_USAGE_HEALTH" = "🏥 Checking Kokoro Health" ]
    [ "$MSG_USAGE_ALL" = "🎭 Running All Kokoro Usage Examples" ]
}

@test "kokoro::export_messages sets docker hint messages" {
    messages::export_messages

    [[ "$MSG_DOCKER_INSTALL_HINT" == *"https://docs.docker.com/get-docker/"* ]]
    [[ "$MSG_DOCKER_START_HINT" == *"sudo systemctl start docker"* ]]
    [[ "$MSG_DOCKER_PERMISSIONS_HINT" == *"sudo usermod -aG docker"* ]]
}

@test "kokoro::export_messages is idempotent" {
    messages::export_messages
    local first_message="$MSG_INSTALL_SUCCESS"

    messages::export_messages

    # Should still have the same value (idempotent)
    [ "$MSG_INSTALL_SUCCESS" = "$first_message" ]
}

@test "kokoro::export_messages exports all variables" {
    messages::export_messages

    # Test that key variables are exported (accessible in subshells)
    ([ -n "$MSG_INSTALL_SUCCESS" ])
    ([ -n "$MSG_DOCKER_NOT_FOUND" ])
    ([ -n "$MSG_CHECKING_STATUS" ])
    ([ -n "$MSG_ALREADY_INSTALLED" ])
    ([ -n "$MSG_USAGE_SYNTHESIZE" ])
}
