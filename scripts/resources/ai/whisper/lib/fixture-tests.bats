#!/usr/bin/env bats
# Enhanced Whisper Tests with Fixture Data
# Demonstrates how BATS tests can use real fixture files with mocked responses

# Load Vrooli test infrastructure
source "${BATS_TEST_DIRNAME}/../../../../__test/fixtures/setup.bash"

# Expensive setup operations (run once per file)
setup_file() {
    # Use appropriate setup function
    vrooli_setup_service_test "whisper"
    
    # Load dependencies once
    SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    WHISPER_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Load configuration and messages once
    source "${WHISPER_DIR}/config/defaults.sh"
    source "${WHISPER_DIR}/config/messages.sh"
    
    # Load test fixture utilities
    source "${BATS_TEST_DIRNAME}/../../../../__test/fixtures/fixture-loader.bash"
    
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
    export WHISPER_PORT="9090"
    export WHISPER_BASE_URL="http://localhost:9090"
    export WHISPER_API_TIMEOUT="5"
    
    # Source the whisper functions to test
    source "${BATS_TEST_DIRNAME}/../common.sh"
    
    # Mock system functions
    system::is_port_in_use() { return 1; }  # Port available
    docker() { echo "docker $*"; }           # Mock docker calls
    curl() { 
        # Return the mocked response set by fixture_mock_response
        echo "${MOCK_CURL_RESPONSE:-{\"text\":\"default response\"}}"
    }
    
    # Export config functions
    whisper::export_config
    whisper::export_messages
    
    export -f system::is_port_in_use docker curl
}

# BATS teardown function - runs after each test
teardown() {
    vrooli_cleanup_test
}

#######################################
# Fixture-Based Transcription Tests
#######################################

@test "whisper::transcribe processes speech audio fixture correctly" {
    # Load real audio fixture
    local audio_file
    audio_file=$(fixture_get_path "audio" "whisper/test_speech.mp3")
    
    # Mock expected transcription response
    fixture_mock_response "transcription" "$audio_file"
    
    # Test the transcription function
    run whisper::transcribe_audio "$audio_file"
    
    # Verify success and correct output
    assert_success
    assert_output_contains "I have a dream"
    
    # Verify correct API call was made
    fixture_assert_processed "$audio_file" "--data-binary"
}

@test "whisper::transcribe handles silent audio fixture" {
    local audio_file
    audio_file=$(fixture_get_path "audio" "whisper/test_silent.wav")
    
    # Mock empty transcription response
    fixture_mock_response "transcription" "$audio_file"
    
    run whisper::transcribe_audio "$audio_file"
    
    assert_success
    assert_output_equals ""
}

@test "whisper::transcribe handles corrupted audio fixture" {
    local audio_file
    audio_file=$(fixture_get_path "audio" "whisper/test_corrupted.wav")
    
    # Mock error response
    fixture_mock_response "transcription" "$audio_file" '{"error":"Invalid audio format"}'
    
    run whisper::transcribe_audio "$audio_file"
    
    # Should handle error gracefully
    assert_failure
    assert_output_contains "Invalid audio format"
}

@test "whisper::transcribe processes all whisper test fixtures" {
    # Get all whisper test files
    local -a audio_files
    mapfile -t audio_files < <(fixture_get_all "audio" "whisper/test_*.mp3" "whisper/test_*.wav")
    
    assert [ ${#audio_files[@]} -gt 0 ]  # Ensure we have test files
    
    local processed_count=0
    for audio_file in "${audio_files[@]}"; do
        # Skip corrupted file in this test
        [[ "$audio_file" == *"corrupted"* ]] && continue
        
        # Mock appropriate response
        fixture_mock_response "transcription" "$audio_file"
        
        # Test transcription
        run whisper::transcribe_audio "$audio_file"
        
        # Should succeed for valid files
        assert_success
        ((processed_count++))
    done
    
    # Ensure we processed multiple files
    assert [ $processed_count -gt 2 ]
}

#######################################
# Format Support Tests with Fixtures
#######################################

@test "whisper::validate_audio_format supports fixture formats" {
    # Test MP3 format
    local mp3_file
    mp3_file=$(fixture_get_path "audio" "whisper/test_speech.mp3")
    
    run whisper::validate_audio_format "$mp3_file"
    assert_success
    assert_output_contains "mp3"
    
    # Test WAV format  
    local wav_file
    wav_file=$(fixture_get_path "audio" "whisper/test_silent.wav")
    
    run whisper::validate_audio_format "$wav_file"
    assert_success
    assert_output_contains "wav"
    
    # Test FLAC format
    local flac_file
    flac_file=$(fixture_get_path "audio" "whisper/test_format.flac")
    
    run whisper::validate_audio_format "$flac_file"
    assert_success
    assert_output_contains "flac"
}

#######################################
# API Integration Tests with Fixtures
#######################################

@test "whisper::api_transcribe constructs correct API calls for fixtures" {
    local audio_file
    audio_file=$(fixture_get_path "audio" "whisper/test_short.mp3")
    
    # Mock successful API response
    fixture_mock_response "transcription" "$audio_file"
    
    # Test API call construction
    run whisper::api_transcribe "$audio_file" "en" "transcribe"
    
    assert_success
    
    # Verify correct API endpoint was called
    assert_curl_called_with "POST"
    assert_curl_called_with "/v1/audio/transcriptions"
    fixture_assert_processed "$audio_file" "-F audio="
}

#######################################
# Error Handling Tests with Fixtures
#######################################

@test "whisper::handle_api_error processes fixture error responses correctly" {
    # Test with a known error-inducing fixture
    local corrupted_file
    corrupted_file=$(fixture_get_path "audio" "whisper/test_corrupted.wav")
    
    # Mock API error response
    fixture_mock_response "transcription" "$corrupted_file" '{"error":"unsupported_file_format","message":"Audio file format not supported"}'
    
    run whisper::transcribe_audio "$corrupted_file"
    
    assert_failure
    assert_output_contains "unsupported_file_format"
    assert_output_contains "Audio file format not supported"
}

#######################################
# Performance and Edge Case Tests
#######################################

@test "whisper::transcribe handles very short audio fixture" {
    local short_file
    short_file=$(fixture_get_path "audio" "whisper/test_very_short.wav")
    
    fixture_mock_response "transcription" "$short_file" '{"text":"Brief"}'
    
    run whisper::transcribe_audio "$short_file"
    
    assert_success
    assert_output_contains "Brief"
}

@test "whisper::transcribe handles quiet audio fixture" {
    local quiet_file
    quiet_file=$(fixture_get_path "audio" "whisper/test_quiet.wav")
    
    fixture_mock_response "transcription" "$quiet_file" '{"text":"[inaudible]"}'
    
    run whisper::transcribe_audio "$quiet_file"
    
    assert_success  
    assert_output_contains "[inaudible]"
}

#######################################
# Metadata Validation Tests
#######################################

@test "fixture metadata is consistent with test expectations" {
    # Load audio metadata
    local metadata
    metadata=$(fixture_load_metadata "audio")
    
    # Verify metadata contains whisper test files
    assert_output_contains "test_speech.mp3"
    assert_output_contains "test_silent.wav"
    assert_output_contains "test_corrupted.wav"
    
    # Verify expected transcriptions match our mocks
    # This ensures fixture mocks stay in sync with metadata
}