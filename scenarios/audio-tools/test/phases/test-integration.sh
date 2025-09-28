#!/bin/bash
# Integration tests for audio-tools P0 features
# Tests all core audio processing operations

set -euo pipefail

# Configuration
# Detect the actual running port from the service
if [ -z "${API_PORT:-}" ]; then
    # Using ps with grep as pgrep may not capture complex patterns
    # shellcheck disable=SC2009
    API_PORT=$(ps aux | grep -E "audio-tools.*-port" | grep -oE "\-port [0-9]+" | awk '{print $2}' | head -1)
    if [ -z "$API_PORT" ]; then
        API_PORT="19607"
    fi
fi
readonly API_BASE="http://localhost:${API_PORT}"
readonly TEST_DIR="/tmp/audio-tools-test-$$"
# Test configuration

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m' # No Color

# Test tracking
TESTS_PASSED=0
TESTS_FAILED=0

# Setup test environment
setup() {
    echo -e "${BLUE}Setting up test environment...${NC}"
    mkdir -p "$TEST_DIR"
    
    # Create test audio file (3 second sine wave)
    ffmpeg -y -f lavfi -i "sine=frequency=440:duration=3" -ar 44100 -ac 2 -f wav "$TEST_DIR/test.wav" 2>/dev/null
    
    # Wait for API to be ready
    local retries=10
    while [ $retries -gt 0 ]; do
        if curl -sf "$API_BASE/health" > /dev/null 2>&1; then
            echo -e "${GREEN}API is ready${NC}"
            return 0
        fi
        echo "Waiting for API to be ready... ($retries retries left)"
        sleep 1
        retries=$((retries - 1))
    done
    
    echo -e "${RED}API failed to start${NC}"
    return 1
}

# Cleanup test environment
cleanup() {
    echo -e "${BLUE}Cleaning up test environment...${NC}"
    rm -rf "$TEST_DIR"
}

# Test function wrapper
test_case() {
    local test_name="$1"
    local test_command="$2"
    
    echo -e "${BLUE}Testing: $test_name${NC}"
    
    if eval "$test_command"; then
        echo -e "${GREEN}✓ $test_name passed${NC}"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        echo -e "${RED}✗ $test_name failed${NC}"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

# P0 Feature Tests

test_health_check() {
    local response
    response=$(curl -sf "$API_BASE/health" | jq -r '.status')
    [ "$response" = "healthy" ]
}

test_metadata_extraction() {
    local response
    response=$(curl -sf -X POST "$API_BASE/api/v1/audio/metadata" \
        -F "audio=@$TEST_DIR/test.wav" | jq -r '.format')
    [ "$response" = "wav" ]
}

test_format_conversion() {
    local response
    response=$(curl -sf -X POST "$API_BASE/api/v1/audio/convert" \
        -F "audio=@$TEST_DIR/test.wav" \
        -F "format=mp3" | jq -r '.format')
    [ "$response" = "mp3" ]
}

test_audio_trim() {
    local response
    response=$(curl -sf -X POST "$API_BASE/api/v1/audio/edit" \
        -F "audio=@$TEST_DIR/test.wav" \
        -F "operation=trim" \
        -F 'parameters={"start":"0.5","end":"2.5"}' | jq -r '.output_files[0].duration_seconds')
    
    # Check if duration is approximately 2 seconds (2.5 - 0.5)
    local duration_int
    duration_int=$(echo "$response" | cut -d. -f1)
    [ "$duration_int" = "1" ] || [ "$duration_int" = "2" ]
}

test_volume_adjustment() {
    local response
    response=$(curl -sf -X POST "$API_BASE/api/v1/audio/edit" \
        -F "audio=@$TEST_DIR/test.wav" \
        -F "operation=volume" \
        -F 'parameters={"volume_factor":"1.5"}' | jq -r '.job_id')
    [ -n "$response" ]
}

test_audio_normalize() {
    local response
    response=$(curl -sf -X POST "$API_BASE/api/v1/audio/edit" \
        -F "audio=@$TEST_DIR/test.wav" \
        -F "operation=normalize" \
        -F 'parameters={"target_level":"-16"}' | jq -r '.job_id')
    [ -n "$response" ]
}

test_speed_modification() {
    local response
    response=$(curl -sf -X POST "$API_BASE/api/v1/audio/edit" \
        -F "audio=@$TEST_DIR/test.wav" \
        -F "operation=speed" \
        -F 'parameters={"speed_factor":"1.5"}' | jq -r '.job_id')
    [ -n "$response" ]
}

test_pitch_modification() {
    local response
    response=$(curl -sf -X POST "$API_BASE/api/v1/audio/edit" \
        -F "audio=@$TEST_DIR/test.wav" \
        -F "operation=pitch" \
        -F 'parameters={"semitones":"5"}' | jq -r '.job_id')
    [ -n "$response" ]
}

test_audio_enhance() {
    local response
    response=$(curl -sf -X POST "$API_BASE/api/v1/audio/enhance" \
        -F "audio=@$TEST_DIR/test.wav" \
        -F "environment=general" | jq -r '.enhanced_file_path')
    [ -n "$response" ]
}

test_audio_analyze() {
    local response
    response=$(curl -sf -X POST "$API_BASE/api/v1/audio/analyze" \
        -F "audio=@$TEST_DIR/test.wav" | jq -r '.analysis.format')
    [ "$response" = "wav" ]
}

test_vad() {
    local response
    response=$(curl -sf -X POST "$API_BASE/api/v1/audio/vad" \
        -F "audio=@$TEST_DIR/test.wav" \
        -F "threshold=-40" | jq -r '.speech_segments')
    [ -n "$response" ]
}

test_remove_silence() {
    local response
    response=$(curl -sf -X POST "$API_BASE/api/v1/audio/remove-silence" \
        -F "audio=@$TEST_DIR/test.wav" \
        -F "threshold=-40" | jq -r '.output_file')
    [ -n "$response" ]
}

test_api_status() {
    local response
    response=$(curl -sf "$API_BASE/health" | jq -r '.service')
    [ "$response" = "audio-tools" ]
}

test_cli_available() {
    which audio-tools > /dev/null 2>&1
}

# Main test execution
main() {
    trap cleanup EXIT
    
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}Audio Tools P0 Integration Tests${NC}"
    echo -e "${BLUE}========================================${NC}"
    
    setup || exit 1
    
    # Run P0 feature tests
    test_case "Health Check" test_health_check
    test_case "API Status" test_api_status
    test_case "CLI Available" test_cli_available
    test_case "Metadata Extraction" test_metadata_extraction
    test_case "Format Conversion (MP3)" test_format_conversion
    test_case "Audio Trim" test_audio_trim
    test_case "Volume Adjustment" test_volume_adjustment
    test_case "Audio Normalize" test_audio_normalize
    test_case "Speed Modification" test_speed_modification
    test_case "Pitch Modification" test_pitch_modification
    test_case "Audio Enhancement" test_audio_enhance
    test_case "Audio Analysis" test_audio_analyze
    test_case "Voice Activity Detection" test_vad
    test_case "Silence Removal" test_remove_silence
    
    # Summary
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}Test Summary:${NC}"
    echo -e "${GREEN}  Passed: $TESTS_PASSED${NC}"
    if [ $TESTS_FAILED -gt 0 ]; then
        echo -e "${RED}  Failed: $TESTS_FAILED${NC}"
    else
        echo -e "${GREEN}  Failed: $TESTS_FAILED${NC}"
    fi
    echo -e "${BLUE}========================================${NC}"
    
    # Exit with failure if any tests failed
    [ $TESTS_FAILED -eq 0 ]
}

# Run tests
main "$@"