#!/usr/bin/env bats
# Tests for Whisper messages.sh configuration

# Setup for each test
setup() {
    # Set test environment
    export WHISPER_PORT="8090"
    
    # Load the messages
    SCRIPT_DIR="$(dirname "${BATS_TEST_FILENAME}")"
    source "${SCRIPT_DIR}/messages.sh"
}

# Test message export
@test "whisper::export_messages sets success messages" {
    whisper::export_messages
    
    [ "$MSG_INSTALL_SUCCESS" = "✅ Whisper installed successfully" ]
    [ "$MSG_START_SUCCESS" = "✅ Whisper started successfully" ]
    [ "$MSG_STOP_SUCCESS" = "✅ Whisper stopped successfully" ]
    [ "$MSG_RESTART_SUCCESS" = "✅ Whisper restarted successfully" ]
    [ "$MSG_UNINSTALL_SUCCESS" = "✅ Whisper uninstalled successfully" ]
    [ "$MSG_HEALTHY" = "✅ Whisper API is healthy" ]
    [ "$MSG_RUNNING" = "✅ Whisper container is running" ]
}

@test "whisper::export_messages sets error messages" {
    whisper::export_messages
    
    [ "$MSG_DOCKER_NOT_FOUND" = "Docker is not installed" ]
    [ "$MSG_DOCKER_NOT_RUNNING" = "Docker daemon is not running" ]
    [ "$MSG_INSTALL_FAILED" = "❌ Whisper installation failed" ]
    [ "$MSG_NOT_INSTALLED" = "❌ Whisper is not installed" ]
    [ "$MSG_NOT_RUNNING" = "Whisper is not running" ]
    [ "$MSG_TRANSCRIPTION_FAILED" = "❌ Audio transcription failed" ]
    [ "$MSG_FILE_NOT_FOUND" = "❌ Audio file not found" ]
}

@test "whisper::export_messages sets info messages" {
    whisper::export_messages
    
    [ "$MSG_CHECKING_STATUS" = "🔍 Checking Whisper status..." ]
    [ "$MSG_PULLING_IMAGE" = "📥 Pulling Whisper image..." ]
    [ "$MSG_STARTING_CONTAINER" = "Starting Whisper container..." ]
    [ "$MSG_TRANSCRIBING" = "🎤 Transcribing audio file..." ]
    [ "$MSG_LOADING_MODEL" = "📥 Loading Whisper model..." ]
}

@test "whisper::export_messages sets warning messages" {
    whisper::export_messages
    
    [ "$MSG_ALREADY_INSTALLED" = "Whisper is already installed and running" ]
    [[ "$MSG_ALREADY_RUNNING" == *"8090"* ]]  # Should contain the port
    [ "$MSG_UNINSTALL_WARNING" = "This will remove Whisper and all transcription data" ]
    [ "$MSG_MODEL_LOADING_SLOW" = "⚠️  Model loading is taking longer than expected" ]
}

@test "whisper::export_messages sets usage messages" {
    whisper::export_messages
    
    [ "$MSG_USAGE_TRANSCRIBE" = "🎤 Testing Whisper Transcription API" ]
    [ "$MSG_USAGE_TRANSLATE" = "🌐 Testing Whisper Translation API" ]
    [ "$MSG_USAGE_MODELS" = "🧠 Checking Available Models" ]
    [ "$MSG_USAGE_HEALTH" = "🏥 Checking Whisper Health" ]
    [ "$MSG_USAGE_ALL" = "🎭 Running All Whisper Usage Examples" ]
}

@test "whisper::export_messages sets docker hint messages" {
    whisper::export_messages
    
    [[ "$MSG_DOCKER_INSTALL_HINT" == *"https://docs.docker.com/get-docker/"* ]]
    [[ "$MSG_DOCKER_START_HINT" == *"sudo systemctl start docker"* ]]
    [[ "$MSG_DOCKER_PERMISSIONS_HINT" == *"sudo usermod -aG docker"* ]]
}

@test "whisper::export_messages is idempotent" {
    whisper::export_messages
    local first_message="$MSG_INSTALL_SUCCESS"
    
    whisper::export_messages
    
    # Should still have the same value (idempotent)
    [ "$MSG_INSTALL_SUCCESS" = "$first_message" ]
}

@test "whisper::export_messages exports all variables" {
    whisper::export_messages
    
    # Test that key variables are exported (accessible in subshells)
    ([ -n "$MSG_INSTALL_SUCCESS" ])
    ([ -n "$MSG_DOCKER_NOT_FOUND" ])
    ([ -n "$MSG_CHECKING_STATUS" ])
    ([ -n "$MSG_ALREADY_INSTALLED" ])
    ([ -n "$MSG_USAGE_TRANSCRIBE" ])
}
