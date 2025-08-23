#!/usr/bin/env bats
# Tests for Whisper API functions

# Source trash module for safe test cleanup
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Load Vrooli test infrastructure (REQUIRED)
source "${BATS_TEST_DIRNAME}/../../../../__test/fixtures/setup.bash"

# Expensive setup operations (run once per file)
setup_file() {
    # Use appropriate setup function
    vrooli_setup_service_test "whisper"
    
    # Load dependencies once
    SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    WHISPER_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Source necessary files
    source "${WHISPER_DIR}/config/defaults.sh"
    source "${WHISPER_DIR}/config/messages.sh"
    
    # Export paths for use in setup()
    export SETUP_FILE_SCRIPT_DIR="$SCRIPT_DIR"
    export SETUP_FILE_WHISPER_DIR="$WHISPER_DIR"
}

# Lightweight per-test setup
setup() {
    # Setup standard mocks
    vrooli_auto_setup
    
    # Use paths from setup_file
    SCRIPT_DIR="${SETUP_FILE_SCRIPT_DIR}"
    WHISPER_DIR="${SETUP_FILE_WHISPER_DIR}"
    
    # Set test environment
    export WHISPER_CONTAINER_NAME="whisper-test"
    export WHISPER_BASE_URL="http://localhost:9090"
    export WHISPER_API_TIMEOUT="10"
    export WHISPER_MODEL_SIZE="base"
    
    # Export config functions
    whisper::export_config
    whisper::export_messages
    
    # Mock health check
    whisper::is_healthy() { return 0; }
}

# BATS teardown function - runs after each test
teardown() {
    vrooli_cleanup_test
    
    # Clean up any test files
    trash::safe_remove "/tmp/whisper_test_*" --test-cleanup
}

# === API Health Check Tests ===
@test "whisper::test_api checks service health" {
    # Mock successful health check
    curl() {
        if [[ "$*" == *"/health"* ]]; then
            echo '{"status":"healthy","model":"base","version":"1.0.0"}'
            return 0
        fi
        return 1
    }
    
    run whisper::test_api
    [ "$status" -eq 0 ]
    [[ "$output" == *"healthy"* ]]
}

@test "whisper::test_api fails when service unhealthy" {
    # Mock failed health check
    whisper::is_healthy() { return 1; }
    
    run whisper::test_api
    [ "$status" -eq 1 ]
}

# === Transcription API Tests ===
@test "whisper::transcribe_audio handles successful transcription" {
    # Create a mock audio file
    local test_file="/tmp/whisper_test_audio.mp3"
    touch "$test_file"
    
    # Mock successful transcription
    curl() {
        if [[ "$*" == *"/transcribe"* ]]; then
            echo '{"text":"Hello world","language":"en","duration":2.5}'
            return 0
        fi
        return 1
    }
    
    run whisper::transcribe_audio "$test_file"
    [ "$status" -eq 0 ]
    [[ "$output" == *"Hello world"* ]]
}

@test "whisper::transcribe_audio validates file exists" {
    run whisper::transcribe_audio "/nonexistent/file.mp3"
    [ "$status" -eq 1 ]
    [[ "$output" == *"not found"* ]] || [[ "$output" == *"does not exist"* ]]
}

@test "whisper::transcribe_audio handles API errors" {
    local test_file="/tmp/whisper_test_audio.mp3"
    touch "$test_file"
    
    # Mock API error
    curl() {
        if [[ "$*" == *"/transcribe"* ]]; then
            echo '{"error":"Model not loaded"}'
            return 1
        fi
        return 1
    }
    
    run whisper::transcribe_audio "$test_file"
    [ "$status" -eq 1 ]
    [[ "$output" == *"error"* ]] || [[ "$output" == *"failed"* ]]
}

# === Model Management Tests ===
@test "whisper::list_models returns available models" {
    # Mock model list response
    curl() {
        if [[ "$*" == *"/models"* ]]; then
            echo '{"models":["tiny","base","small","medium","large"]}'
            return 0
        fi
        return 1
    }
    
    run whisper::list_models
    [ "$status" -eq 0 ]
    [[ "$output" == *"base"* ]]
    [[ "$output" == *"small"* ]]
}

@test "whisper::load_model loads specified model" {
    # Mock model load response
    curl() {
        if [[ "$*" == *"/load"* ]] && [[ "$*" == *"small"* ]]; then
            echo '{"status":"loaded","model":"small"}'
            return 0
        fi
        return 1
    }
    
    run whisper::load_model "small"
    [ "$status" -eq 0 ]
    [[ "$output" == *"loaded"* ]]
}

# === Language Detection Tests ===
@test "whisper::detect_language identifies language from audio" {
    local test_file="/tmp/whisper_test_audio.mp3"
    touch "$test_file"
    
    # Mock language detection
    curl() {
        if [[ "$*" == *"/detect"* ]]; then
            echo '{"language":"en","confidence":0.95}'
            return 0
        fi
        return 1
    }
    
    run whisper::detect_language "$test_file"
    [ "$status" -eq 0 ]
    [[ "$output" == *"en"* ]]
}

# === Translation API Tests ===
@test "whisper::translate_audio translates to English" {
    local test_file="/tmp/whisper_test_audio.mp3"
    touch "$test_file"
    
    # Mock translation response
    curl() {
        if [[ "$*" == *"/translate"* ]]; then
            echo '{"text":"Good morning","source_language":"es","translated":true}'
            return 0
        fi
        return 1
    }
    
    run whisper::translate_audio "$test_file"
    [ "$status" -eq 0 ]
    [[ "$output" == *"Good morning"* ]]
}

# === Batch Processing Tests ===
@test "whisper::process_directory handles multiple files" {
    # Create test directory with files
    local test_dir="/tmp/whisper_test_batch"
    mkdir -p "$test_dir"
    touch "$test_dir/file1.mp3" "$test_dir/file2.wav"
    
    # Mock batch processing
    whisper::transcribe_audio() {
        echo "Transcribed: $1"
        return 0
    }
    
    run whisper::process_directory "$test_dir"
    [ "$status" -eq 0 ]
    [[ "$output" == *"file1.mp3"* ]]
    [[ "$output" == *"file2.wav"* ]]
    
    # Cleanup
    trash::safe_remove "$test_dir" --test-cleanup
}

# === Error Handling Tests ===
@test "whisper::test_api handles timeout gracefully" {
    # Mock timeout
    curl() {
        sleep 2  # Simulate slow response
        return 124  # Timeout exit code
    }
    export WHISPER_API_TIMEOUT="1"
    
    run whisper::test_api
    [ "$status" -eq 1 ]
    [[ "$output" == *"timeout"* ]] || [[ "$output" == *"Timeout"* ]]
}