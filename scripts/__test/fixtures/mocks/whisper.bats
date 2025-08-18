#!/usr/bin/env bats
# Comprehensive tests for Whisper mock system
# Tests the whisper.sh mock implementation for correctness and integration

# Source trash module for safe test cleanup
MOCK_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
source "${MOCK_DIR}/../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Test setup - load dependencies
setup() {
    # Set up test environment
    export MOCK_UTILS_VERBOSE=false
    export MOCK_VERIFICATION_ENABLED=true
    
    # Load the mock utilities first (required by whisper mock)
    source "${BATS_TEST_DIRNAME}/logs.sh"
    
    # Load verification system if available
    if [[ -f "${BATS_TEST_DIRNAME}/verification.sh" ]]; then
        source "${BATS_TEST_DIRNAME}/verification.sh"
    fi
    
    # Load the whisper mock
    source "${BATS_TEST_DIRNAME}/whisper.sh"
    
    # Initialize clean state for each test
    mock::whisper::reset
    
    # Create test log directory
    TEST_LOG_DIR=$(mktemp -d)
    mock::init_logging "$TEST_LOG_DIR"
    
    # Create test audio files
    TEST_AUDIO_DIR=$(mktemp -d)
    echo "fake audio data" > "$TEST_AUDIO_DIR/test.mp3"
    echo "fake wave data" > "$TEST_AUDIO_DIR/speech.wav"
    echo "" > "$TEST_AUDIO_DIR/silent.mp3"
    echo "corrupted" > "$TEST_AUDIO_DIR/corrupted.ogg"
}

# Wrapper for run command that reloads whisper state afterward
run_whisper_command() {
    run "$@"
    # Reload state from file after commands that might modify state
    if [[ -n "${WHISPER_MOCK_STATE_FILE}" && -f "$WHISPER_MOCK_STATE_FILE" ]]; then
        eval "$(cat "$WHISPER_MOCK_STATE_FILE")" 2>/dev/null || true
    fi
}

# Test cleanup
teardown() {
    # Clean up test files
    if [[ -n "${TEST_AUDIO_DIR:-}" && -d "$TEST_AUDIO_DIR" ]]; then
        trash::safe_remove "$TEST_AUDIO_DIR" --test-cleanup
    fi
    
    if [[ -n "${TEST_LOG_DIR:-}" && -d "$TEST_LOG_DIR" ]]; then
        trash::safe_remove "$TEST_LOG_DIR" --test-cleanup
    fi
    
    # Clean up environment
    unset TEST_AUDIO_DIR
    unset TEST_LOG_DIR
}

# =============================================================================
# Basic Mock State Management Tests
# =============================================================================

@test "whisper mock should initialize with default configuration" {
    # Check default configuration values
    local status=$(mock::whisper::get::service_status)
    local model=$(mock::whisper::get::current_model)
    
    [[ "$status" == "running" ]]
    [[ "$model" == "base" ]]
}

@test "whisper mock reset should clear all state" {
    # Set up some state
    mock::whisper::set_config "port" "9090"
    mock::whisper::set_model "large"
    mock::whisper::add_transcript "test.mp3" '{"text":"custom transcript"}'
    
    # Verify state exists
    [[ "$(mock::whisper::get::current_model)" == "large" ]]
    
    # Reset and verify clean state
    mock::whisper::reset
    
    local status=$(mock::whisper::get::service_status)
    local model=$(mock::whisper::get::current_model)
    [[ "$status" == "running" ]]
    [[ "$model" == "base" ]]
}

@test "whisper configuration should be modifiable" {
    mock::whisper::set_config "port" "8091"
    mock::whisper::set_config "version" "2.0.0"
    
    # Configuration changes should be reflected in responses
    run curl -s "http://localhost:8091/health"
    [[ "$output" =~ '"version":"2.0.0"' ]]
}

@test "whisper model should be changeable" {
    run mock::whisper::set_model "large"
    [[ "$status" -eq 0 ]]
    [[ "$(mock::whisper::get::current_model)" == "large" ]]
    
    # Invalid model should fail
    run mock::whisper::set_model "invalid_model"
    [[ "$status" -eq 1 ]]
    [[ "$output" =~ "Unknown Whisper model" ]]
}

# =============================================================================
# Service Status Management Tests  
# =============================================================================

@test "service status should affect endpoint responses" {
    # Healthy service
    mock::whisper::set_service_status "running"
    run curl -s "http://localhost:8090/health"
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ '"status":"healthy"' ]]
    
    # Unhealthy service
    mock::whisper::set_service_status "unhealthy"
    run curl -s "http://localhost:8090/health"
    [[ "$status" -eq 1 ]]  # Should return HTTP error
    [[ "$output" =~ '"status":"unhealthy"' ]]
}

@test "installing status should show progress information" {
    mock::whisper::set_service_status "installing"
    mock::whisper::set_config "progress" "75"
    mock::whisper::set_config "current_step" "Loading model files"
    
    run curl -s "http://localhost:8090/health"
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ '"status":"installing"' ]]
    [[ "$output" =~ '"progress":75' ]]
    [[ "$output" =~ '"current_step":"Loading model files"' ]]
}

@test "stopped service should have unreachable endpoints" {
    mock::whisper::set_service_status "stopped"
    
    run curl -s "http://localhost:8090/health"
    [[ "$status" -eq 7 ]]  # Connection refused
    [[ "$output" =~ "Connection refused" ]]
}

# =============================================================================
# Docker Command Tests
# =============================================================================

@test "docker ps should show running whisper container" {
    run docker ps
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ "whisper" ]]
    [[ "$output" =~ "Up" ]]
}

@test "docker ps with format should return container name" {
    run docker ps --format "{{.Names}}"
    [[ "$status" -eq 0 ]]
    [[ "$output" == "whisper" ]]
}

@test "docker run should start whisper container" {
    # Stop first
    mock::whisper::set_service_status "stopped"
    
    run_whisper_command docker run -d --name whisper -p 8090:9000 onerahmet/openai-whisper-asr-webservice:latest
    [[ "$status" -eq 0 ]]
    [[ -n "$output" ]]  # Should return container ID
    
    # Service should be running now
    local status=$(mock::whisper::get::service_status)
    [[ "$status" == "running" ]]
}

@test "docker stop should stop whisper container" {
    run_whisper_command docker stop whisper
    [[ "$status" -eq 0 ]]
    [[ "$output" == "whisper" ]]
    
    local status=$(mock::whisper::get::service_status)
    [[ "$status" == "stopped" ]]
}

@test "docker start should start existing container" {
    # Stop first
    mock::whisper::set_service_status "stopped"
    
    run_whisper_command docker start whisper  
    [[ "$status" -eq 0 ]]
    [[ "$output" == "whisper" ]]
    
    local status=$(mock::whisper::get::service_status)
    [[ "$status" == "running" ]]
}

@test "docker logs should show whisper startup messages" {
    run docker logs whisper
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ "Whisper ASR" ]]
    [[ "$output" =~ "Loading model" ]]
    [[ "$output" =~ "Server ready" ]]
}

@test "docker logs should show different messages based on status" {
    # Installing status
    mock::whisper::set_service_status "installing"
    mock::whisper::set_config "progress" "60"
    
    run docker logs whisper
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ "Downloading model" ]]
    [[ "$output" =~ "Progress: 60%" ]]
    
    # Unhealthy status
    mock::whisper::set_service_status "unhealthy"
    
    run docker logs whisper
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ "Error loading model" ]]
}

@test "docker inspect should return container information" {
    run docker inspect whisper
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ '"Name": "/whisper"' ]]
    [[ "$output" =~ '"State"' ]]
    [[ "$output" =~ '"NetworkSettings"' ]]
}

@test "docker inspect with format should return specific fields" {
    run docker inspect whisper --format "{{.State.Status}}"
    [[ "$status" -eq 0 ]]
    [[ "$output" == "running" ]]
    
    run docker inspect whisper --format "{{.NetworkSettings.IPAddress}}"
    [[ "$status" -eq 0 ]]
    [[ "$output" == "172.17.0.2" ]]
}

@test "docker stats should show container resource usage" {
    run docker stats --no-stream whisper
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ "whisper" ]]
    [[ "$output" =~ "CPU" ]]
    [[ "$output" =~ "MEM" ]]
}

@test "docker pull should simulate image download" {
    run docker pull onerahmet/openai-whisper-asr-webservice:latest
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ "Pulling" ]]
    [[ "$output" =~ "Pull complete" ]]
    [[ "$output" =~ "Downloaded newer image" ]]
}

@test "docker commands on non-existent container should fail" {
    run docker stop nonexistent
    [[ "$status" -eq 1 ]]
    [[ "$output" =~ "No such container" ]]
    
    run docker logs nonexistent
    [[ "$status" -eq 1 ]]
    [[ "$output" =~ "No such container" ]]
}

# =============================================================================
# HTTP API Endpoint Tests
# =============================================================================

@test "openapi.json endpoint should return API specification" {
    run curl -s "http://localhost:8090/openapi.json"
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ '"title":"Whisper ASR Webservice"' ]]
    [[ "$output" =~ '"paths"' ]]
    [[ "$output" =~ '"/asr"' ]]
    [[ "$output" =~ '"/detect-language"' ]]
}

@test "docs endpoint should return HTML documentation" {
    run curl -s "http://localhost:8090/docs"
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ "<!DOCTYPE html>" ]]
    [[ "$output" =~ "Whisper ASR" ]]
    [[ "$output" =~ "OpenAPI Spec" ]]
}

@test "health endpoint should return service status" {
    run curl -s "http://localhost:8090/health"
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ '"status":"healthy"' ]]
    [[ "$output" =~ '"model":"base"' ]]
    [[ "$output" =~ '"version"' ]]
    [[ "$output" =~ '"timestamp"' ]]
}

@test "root endpoint should return service information" {
    run curl -s "http://localhost:8090/"
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ '"service":"Whisper ASR"' ]]
    [[ "$output" =~ '"endpoints"' ]]
    [[ "$output" =~ '"/asr"' ]]
}

@test "unknown endpoint should return 404" {
    run curl -s "http://localhost:8090/unknown"
    [[ "$status" -eq 1 ]]  # curl returns 1 for HTTP errors
    [[ "$output" =~ '"error":"Not found"' ]]
    [[ "$output" =~ '"available_endpoints"' ]]
}

# =============================================================================
# ASR (Transcription) Endpoint Tests
# =============================================================================

@test "asr endpoint with GET method should return method not allowed" {
    run curl -s -X GET "http://localhost:8090/asr"
    [[ "$status" -eq 1 ]]
    [[ "$output" =~ '"error":"Method not allowed"' ]]
    [[ "$output" =~ '"code":405' ]]
}

@test "asr endpoint without audio file should return bad request" {
    run curl -s -X POST "http://localhost:8090/asr"
    [[ "$status" -eq 1 ]]
    [[ "$output" =~ '"error":"Bad Request"' ]]
    [[ "$output" =~ '"code":400' ]]
    [[ "$output" =~ "Missing required field: audio_file" ]]
}

@test "asr endpoint with audio file should return transcription" {
    run curl -s -X POST "http://localhost:8090/asr" -F "audio_file=@$TEST_AUDIO_DIR/test.mp3"
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ '"text":' ]]
    [[ "$output" =~ '"segments"' ]]
    [[ "$output" =~ '"language":"en"' ]]
    [[ "$output" =~ '"duration"' ]]
    [[ "$output" =~ '"model":"base"' ]]
    [[ "$output" =~ '"timestamp"' ]]
}

@test "asr endpoint with task=translate should return translation" {
    run curl -s -X POST "http://localhost:8090/asr" \
        -F "audio_file=@$TEST_AUDIO_DIR/test.mp3" \
        -F "task=translate"
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ '"task":"translate"' ]]
    [[ "$output" =~ "English translation" ]]
}

@test "asr endpoint with specific model should use that model" {
    run curl -s -X POST "http://localhost:8090/asr" \
        -F "audio_file=@$TEST_AUDIO_DIR/test.mp3" \
        -F "model=large"
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ '"model":"large"' ]]
}

@test "asr endpoint with language parameter should respect language" {
    run curl -s -X POST "http://localhost:8090/asr" \
        -F "audio_file=@$TEST_AUDIO_DIR/test.mp3" \
        -F "language=es"
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ '"language":"es"' ]]
}

@test "asr endpoint should handle different file types based on filename" {
    # Speech file should have longer transcription
    run curl -s -X POST "http://localhost:8090/asr" -F "audio_file=@$TEST_AUDIO_DIR/speech.wav"
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ "speech recognition test" ]]
    [[ "$output" =~ '"duration":8.5' ]]
    
    # Silent file should have minimal/empty transcription
    run curl -s -X POST "http://localhost:8090/asr" -F "audio_file=@$TEST_AUDIO_DIR/silent.mp3"
    [[ "$status" -eq 0 ]]
    local text=$(echo "$output" | jq -r '.text' 2>/dev/null)
    [[ ${#text} -lt 5 ]]  # Should be very short or empty
}

@test "asr endpoint when service unhealthy should return 503" {
    mock::whisper::set_service_status "unhealthy"
    
    run curl -s -X POST "http://localhost:8090/asr" -F "audio_file=@$TEST_AUDIO_DIR/test.mp3"
    [[ "$status" -eq 1 ]]
    [[ "$output" =~ '"error":"Service Unavailable"' ]]
    [[ "$output" =~ '"code":503' ]]
}

# =============================================================================
# Language Detection Endpoint Tests
# =============================================================================

@test "detect-language endpoint with GET method should return method not allowed" {
    run curl -s -X GET "http://localhost:8090/detect-language"
    [[ "$status" -eq 1 ]]
    [[ "$output" =~ '"error":"Method not allowed"' ]]
    [[ "$output" =~ '"code":405' ]]
}

@test "detect-language endpoint without audio file should return bad request" {
    run curl -s -X POST "http://localhost:8090/detect-language"
    [[ "$status" -eq 1 ]]
    [[ "$output" =~ '"error":"Bad Request"' ]]
    [[ "$output" =~ "Missing required field: audio_file" ]]
}

@test "detect-language endpoint with audio file should detect language" {
    run curl -s -X POST "http://localhost:8090/detect-language" \
        -F "audio_file=@$TEST_AUDIO_DIR/test.mp3"
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ '"language_code":"en"' ]]
    [[ "$output" =~ '"confidence":' ]]
    [[ "$output" =~ '"probabilities"' ]]
    [[ "$output" =~ '"timestamp"' ]]
}

@test "detect-language should detect language hints from filename" {
    # Create test files with language hints
    echo "fake spanish audio" > "$TEST_AUDIO_DIR/spanish_speech.mp3"
    echo "fake french audio" > "$TEST_AUDIO_DIR/french_audio.mp3"
    
    run curl -s -X POST "http://localhost:8090/detect-language" \
        -F "audio_file=@$TEST_AUDIO_DIR/spanish_speech.mp3"
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ '"language_code":"es"' ]]
    
    run curl -s -X POST "http://localhost:8090/detect-language" \
        -F "audio_file=@$TEST_AUDIO_DIR/french_audio.mp3"
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ '"language_code":"fr"' ]]
}

@test "detect-language when service unhealthy should return 503" {
    mock::whisper::set_service_status "unhealthy"
    
    run curl -s -X POST "http://localhost:8090/detect-language" \
        -F "audio_file=@$TEST_AUDIO_DIR/test.mp3"
    [[ "$status" -eq 1 ]]
    [[ "$output" =~ '"error":"Service Unavailable"' ]]
    [[ "$output" =~ '"code":503' ]]
}

# =============================================================================
# Custom Transcript Management Tests
# =============================================================================

@test "custom transcripts should be used for matching files" {
    local custom_transcript='{"text":"This is a custom transcription","language":"en","confidence":0.98}'
    
    mock::whisper::add_transcript "custom.mp3" "$custom_transcript"
    
    run curl -s -X POST "http://localhost:8090/asr" -F "audio_file=@$TEST_AUDIO_DIR/custom.mp3"
    [[ "$status" -eq 0 ]]
    [[ "$output" == "$custom_transcript" ]]
}

@test "transcript patterns should match multiple files" {
    local pattern_transcript='{"text":"Pattern matched transcript","test":true}'
    
    mock::whisper::add_transcript ".*pattern.*" "$pattern_transcript"
    
    run curl -s -X POST "http://localhost:8090/asr" -F "audio_file=@$TEST_AUDIO_DIR/test_pattern_1.mp3"
    [[ "$status" -eq 0 ]]
    [[ "$output" == "$pattern_transcript" ]]
}

@test "transcript assertions should work correctly" {
    mock::whisper::add_transcript "test_transcript.mp3" '{"text":"assertion test transcript"}'
    
    run mock::whisper::assert::transcription_contains "test_transcript.mp3" "assertion test"
    [[ "$status" -eq 0 ]]
    
    run mock::whisper::assert::transcription_contains "test_transcript.mp3" "missing text"
    [[ "$status" -eq 1 ]]
    [[ "$output" =~ "does not contain" ]]
}

# =============================================================================
# Call Tracking Tests
# =============================================================================

@test "API calls should be tracked correctly" {
    # Make multiple calls to different endpoints
    run_whisper_command curl -s "http://localhost:8090/health"
    run_whisper_command curl -s "http://localhost:8090/health"
    run_whisper_command curl -s "http://localhost:8090/docs"
    
    local health_count=$(mock::whisper::get::call_count "/health")
    local docs_count=$(mock::whisper::get::call_count "/docs")
    
    [[ "$health_count" == "2" ]]
    [[ "$docs_count" == "1" ]]
}

@test "endpoint call assertions should work" {
    run_whisper_command curl -s "http://localhost:8090/openapi.json"
    run_whisper_command curl -s "http://localhost:8090/openapi.json"
    
    run mock::whisper::assert::endpoint_called "/openapi.json" 2
    [[ "$status" -eq 0 ]]
    
    run mock::whisper::assert::endpoint_called "/openapi.json" 5
    [[ "$status" -eq 1 ]]
    [[ "$output" =~ "called 2 times, expected at least 5" ]]
}

@test "service status assertions should work" {
    run mock::whisper::assert::service_running
    [[ "$status" -eq 0 ]]
    
    mock::whisper::set_service_status "stopped"
    
    run mock::whisper::assert::service_running
    [[ "$status" -eq 1 ]]
    [[ "$output" =~ "service is not running" ]]
    
    run mock::whisper::assert::service_stopped
    [[ "$status" -eq 0 ]]
}

# =============================================================================
# Error Injection Tests
# =============================================================================

@test "injected docker errors should be triggered" {
    mock::whisper::inject_error "docker" "connection_failed"
    
    # This would need to be implemented in the mock to check for injected errors
    # For now, verify the error was recorded
    run mock::whisper::debug::dump_state
    [[ "$output" =~ "connection_failed" ]]
}

@test "curl headers should work correctly" {
    run curl -s -i "http://localhost:8090/health"
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ "HTTP/1.1 200 OK" ]]
    [[ "$output" =~ "Content-Type: application/json" ]]
    [[ "$output" =~ "Server: Whisper-ASR/1.0" ]]
}

@test "curl with write-out should show status code" {
    run curl -s -w "%{http_code}" "http://localhost:8090/health"
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ "200" ]]
}

@test "curl with max-time should not affect mock behavior" {
    run curl -s --max-time 30 "http://localhost:8090/health"
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ '"status":"healthy"' ]]
}

# =============================================================================
# Integration and Complex Workflow Tests
# =============================================================================

@test "complete whisper workflow should work end-to-end" {
    # 1. Check service health
    run curl -s "http://localhost:8090/health"
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ '"status":"healthy"' ]]
    
    # 2. Get API documentation
    run curl -s "http://localhost:8090/openapi.json"
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ '"title":"Whisper ASR Webservice"' ]]
    
    # 3. Detect language
    run curl -s -X POST "http://localhost:8090/detect-language" \
        -F "audio_file=@$TEST_AUDIO_DIR/test.mp3"
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ '"language_code":"en"' ]]
    
    # 4. Transcribe audio
    run curl -s -X POST "http://localhost:8090/asr" \
        -F "audio_file=@$TEST_AUDIO_DIR/test.mp3" \
        -F "language=en"
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ '"text":' ]]
    [[ "$output" =~ '"language":"en"' ]]
    
    # Verify all endpoints were called
    [[ "$(mock::whisper::get::call_count '/health')" == "1" ]]
    [[ "$(mock::whisper::get::call_count '/openapi.json')" == "1" ]]
    [[ "$(mock::whisper::get::call_count '/detect-language')" == "1" ]]
    [[ "$(mock::whisper::get::call_count '/asr')" == "1" ]]
}

@test "docker container lifecycle should work correctly" {
    # Stop container
    run_whisper_command docker stop whisper
    [[ "$status" -eq 0 ]]
    
    # Verify stopped
    run mock::whisper::assert::service_stopped
    [[ "$status" -eq 0 ]]
    
    # API should be unreachable
    run curl -s "http://localhost:8090/health"
    [[ "$status" -eq 7 ]]  # Connection refused
    
    # Start container
    run_whisper_command docker start whisper
    [[ "$status" -eq 0 ]]
    
    # Verify running
    run mock::whisper::assert::service_running
    [[ "$status" -eq 0 ]]
    
    # API should work again
    run curl -s "http://localhost:8090/health"
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ '"status":"healthy"' ]]
}

@test "model changes should affect transcription responses" {
    # Set to large model
    mock::whisper::set_model "large"
    
    run curl -s -X POST "http://localhost:8090/asr" \
        -F "audio_file=@$TEST_AUDIO_DIR/test.mp3"
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ '"model":"large"' ]]
    
    # Health endpoint should also reflect model change
    run curl -s "http://localhost:8090/health"
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ '"model":"large"' ]]
}

@test "service installation process should be realistic" {
    # Set to installing state
    mock::whisper::set_service_status "installing"
    mock::whisper::set_config "progress" "25"
    mock::whisper::set_config "current_step" "Downloading model"
    
    # Health shows installation progress
    run curl -s "http://localhost:8090/health"
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ '"status":"installing"' ]]
    [[ "$output" =~ '"progress":25' ]]
    
    # API endpoints should be unavailable
    run curl -s -X POST "http://localhost:8090/asr" -F "audio_file=@$TEST_AUDIO_DIR/test.mp3"
    [[ "$status" -eq 1 ]]
    [[ "$output" =~ '"error":"Service initializing"' ]]
    
    # Complete installation
    mock::whisper::set_service_status "running"
    mock::whisper::set_config "progress" "100"
    mock::whisper::set_config "current_step" "ready"
    
    # API should work now
    run curl -s -X POST "http://localhost:8090/asr" -F "audio_file=@$TEST_AUDIO_DIR/test.mp3"
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ '"text":' ]]
}

# =============================================================================
# Debug and Utility Tests
# =============================================================================

@test "debug dump should show comprehensive state information" {
    # Set up some state
    mock::whisper::set_model "medium"
    mock::whisper::add_transcript "debug.mp3" '{"text":"debug transcript"}'
    run_whisper_command curl -s "http://localhost:8090/health"
    
    run mock::whisper::debug::dump_state
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ "Whisper Mock State Dump" ]]
    [[ "$output" =~ "Configuration:" ]]
    [[ "$output" =~ "model: medium" ]]
    [[ "$output" =~ "Containers:" ]]
    [[ "$output" =~ "whisper: running" ]]
    [[ "$output" =~ "Endpoints:" ]]
    [[ "$output" =~ "Transcripts:" ]]
    [[ "$output" =~ "debug.mp3" ]]
}

@test "debug list models should show available models" {
    run mock::whisper::debug::list_models
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ "Available Whisper Models" ]]
    [[ "$output" =~ "tiny:" ]]
    [[ "$output" =~ "base:" ]]
    [[ "$output" =~ "small:" ]]
    [[ "$output" =~ "medium:" ]]
    [[ "$output" =~ "large:" ]]
    [[ "$output" =~ "Current model: base" ]]
}

@test "get functions should return current state" {
    mock::whisper::set_model "small"
    mock::whisper::set_service_status "unhealthy"
    
    [[ "$(mock::whisper::get::current_model)" == "small" ]]
    [[ "$(mock::whisper::get::service_status)" == "unhealthy" ]]
}

# =============================================================================
# Logging Integration Tests
# =============================================================================

@test "whisper commands should be logged correctly" {
    # Ensure we have a test log directory
    [[ -n "$TEST_LOG_DIR" ]]
    
    # Run some commands
    curl -s "http://localhost:8090/health" >/dev/null
    docker ps >/dev/null
    
    # Check that commands were logged
    [[ -f "$TEST_LOG_DIR/http_calls.log" ]]
    [[ -f "$TEST_LOG_DIR/docker_calls.log" ]]
    
    local http_log=$(cat "$TEST_LOG_DIR/http_calls.log")
    local docker_log=$(cat "$TEST_LOG_DIR/docker_calls.log")
    
    [[ "$http_log" =~ "curl: -s http://localhost:8090/health" ]]
    [[ "$docker_log" =~ "docker: ps" ]]
}

@test "whisper state changes should be logged" {
    mock::whisper::set_service_status "installing"
    mock::whisper::set_model "large"
    
    [[ -f "$TEST_LOG_DIR/used_mocks.log" ]]
    
    local state_log=$(cat "$TEST_LOG_DIR/used_mocks.log")
    [[ "$state_log" =~ "whisper_service_status:whisper:installing" ]]
    [[ "$state_log" =~ "whisper_model:current:large" ]]
}

# =============================================================================
# Performance and Edge Case Tests
# =============================================================================

@test "large transcript responses should be handled correctly" {
    local large_transcript='{"text":"'$(printf 'This is a very long transcription. %.0s' {1..1000})'","duration":3600}'
    mock::whisper::add_transcript "large.mp3" "$large_transcript"
    
    run curl -s -X POST "http://localhost:8090/asr" -F "audio_file=@$TEST_AUDIO_DIR/large.mp3"
    [[ "$status" -eq 0 ]]
    [[ ${#output} -gt 10000 ]]
}

@test "empty and null responses should be handled gracefully" {
    mock::whisper::add_transcript "empty.mp3" ""
    
    run curl -s -X POST "http://localhost:8090/asr" -F "audio_file=@$TEST_AUDIO_DIR/empty.mp3"
    [[ "$status" -eq 0 ]]
    [[ -z "$output" ]]
}

@test "concurrent API calls should maintain state correctly" {
    # Simulate concurrent calls by making multiple rapid requests
    for i in {1..10}; do
        curl -s "http://localhost:8090/health" >/dev/null &
    done
    wait
    
    # All calls should have been tracked
    local call_count=$(mock::whisper::get::call_count "/health")
    [[ "$call_count" == "10" ]]
}

@test "malformed requests should be handled gracefully" {
    # Missing form boundary
    run curl -s -X POST "http://localhost:8090/asr" -H "Content-Type: multipart/form-data"
    [[ "$status" -eq 1 ]]
    [[ "$output" =~ '"error":"Bad Request"' ]]
    
    # Invalid JSON in mock responses should not crash
    mock::whisper::add_transcript "invalid.mp3" "not json at all"
    
    run curl -s -X POST "http://localhost:8090/asr" -F "audio_file=@$TEST_AUDIO_DIR/invalid.mp3"
    [[ "$status" -eq 0 ]]
    [[ "$output" == "not json at all" ]]
}

# =============================================================================
# State Persistence and Recovery Tests
# =============================================================================

@test "state should persist across subshells" {
    # Set state in current shell
    mock::whisper::set_model "large"
    mock::whisper::add_transcript "persist.mp3" '{"text":"persistent transcript"}'
    
    # Access from subshell (simulated by function that loads state)
    local model_in_subshell
    model_in_subshell=$(bash -c 'source "'"${BATS_TEST_DIRNAME}/whisper.sh"'"; mock::whisper::get::current_model')
    
    [[ "$model_in_subshell" == "large" ]]
}

@test "mock should recover from corrupted state file gracefully" {
    # Corrupt the state file
    if [[ -n "${WHISPER_MOCK_STATE_FILE}" && -f "$WHISPER_MOCK_STATE_FILE" ]]; then
        echo "corrupted state data" > "$WHISPER_MOCK_STATE_FILE"
    fi
    
    # Mock should still work with defaults
    run curl -s "http://localhost:8090/health"
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ '"status":"healthy"' ]]
}

@test "missing state file should not cause errors" {
    # Remove state file
    if [[ -n "${WHISPER_MOCK_STATE_FILE}" && -f "$WHISPER_MOCK_STATE_FILE" ]]; then
        trash::safe_remove "$WHISPER_MOCK_STATE_FILE" --test-cleanup
    fi
    
    # Mock should still work
    run curl -s "http://localhost:8090/health"
    [[ "$status" -eq 0 ]]
    [[ "$output" =~ '"status":"healthy"' ]]
}