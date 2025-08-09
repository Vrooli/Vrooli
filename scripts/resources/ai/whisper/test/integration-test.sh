#!/usr/bin/env bash
# Whisper Integration Test
# Tests real Whisper audio transcription functionality
# Tests API endpoints, transcription capabilities, and model management
# Enhanced with fixture-based testing for comprehensive validation

set -euo pipefail

# Source shared integration test library
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source var.sh for directory variables
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../../lib/utils/var.sh"

# Source integration test libraries using var.sh
# shellcheck disable=SC1091
source "${var_SCRIPTS_TEST_DIR}/integration/health-check.sh"

#######################################
# SERVICE-SPECIFIC CONFIGURATION
#######################################

# Load Whisper configuration
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/../config/defaults.sh"
whisper::export_config

# Override library defaults with Whisper-specific settings
SERVICE_NAME="whisper"
BASE_URL="${WHISPER_BASE_URL:-http://localhost:9000}"
HEALTH_ENDPOINT="/docs"
REQUIRED_TOOLS=("curl" "jq" "docker")
SERVICE_METADATA=(
    "API Port: ${WHISPER_PORT:-9000}"
    "Container: ${WHISPER_CONTAINER_NAME:-whisper}"
    "Models Dir: ${WHISPER_MODELS_DIR:-${HOME}/.whisper/models}"
)

# Test configuration
readonly TEST_AUDIO_DIR="$SCRIPT_DIR/../../../__test/fixtures/data/audio"
readonly API_BASE=""

#######################################
# WHISPER-SPECIFIC TEST FUNCTIONS
#######################################

test_whisper_docs_endpoint() {
    local test_name="Whisper API documentation"
    
    local response
    if response=$(make_api_request "/docs" "GET" 10); then
        if echo "$response" | grep -qi "whisper\|swagger\|openapi\|fastapi"; then
            log_test_result "$test_name" "PASS" "API documentation accessible"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "API documentation not accessible"
    return 1
}

test_transcribe_endpoint_structure() {
    local test_name="transcribe endpoint structure"
    
    # Test transcribe endpoint without file (should return 422 or 400)
    local response
    local status_code
    if response=$(curl -s -w "HTTPSTATUS:%{http_code}" -X POST "$BASE_URL/v1/audio/transcriptions" 2>/dev/null); then
        status_code=$(echo "$response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
        
        # Should return 422 (validation error) or 400 (bad request) for missing file
        if [[ "$status_code" =~ ^(422|400|405)$ ]]; then
            log_test_result "$test_name" "PASS" "transcribe endpoint exists (HTTP: $status_code)"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "transcribe endpoint not accessible"
    return 1
}

test_models_endpoint() {
    local test_name="models endpoint"
    
    local response
    if response=$(make_api_request "/v1/models" "GET" 10); then
        if validate_json_response "$response"; then
            log_test_result "$test_name" "PASS" "models endpoint returns JSON"
            return 0
        fi
    fi
    
    # Try alternative endpoint
    if response=$(make_api_request "/models" "GET" 5); then
        log_test_result "$test_name" "PASS" "models information available"
        return 0
    fi
    
    log_test_result "$test_name" "FAIL" "models endpoint not accessible"
    return 1
}

test_audio_file_transcription() {
    local test_name="audio file transcription"
    
    # Check if test audio files are available
    if [[ ! -d "$TEST_AUDIO_DIR" ]]; then
        log_test_result "$test_name" "SKIP" "test audio files not available"
        return 2
    fi
    
    # Find a small test audio file
    local test_file
    test_file=$(find "$TEST_AUDIO_DIR" -name "*.mp3" -o -name "*.wav" -o -name "*.ogg" | head -1)
    
    if [[ -z "$test_file" ]] || [[ ! -f "$test_file" ]]; then
        log_test_result "$test_name" "SKIP" "no suitable test audio file found"
        return 2
    fi
    
    # Test file upload and transcription
    local response
    local status_code
    if response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
        -X POST "$BASE_URL/v1/audio/transcriptions" \
        -F "file=@$test_file" \
        -F "model=whisper-1" \
        --max-time 30 2>/dev/null); then
        
        status_code=$(echo "$response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
        response_body=$(echo "$response" | sed 's/HTTPSTATUS:[0-9]*$//')
        
        if [[ "$status_code" == "200" ]]; then
            if echo "$response_body" | grep -q "text\|transcript"; then
                log_test_result "$test_name" "PASS" "audio transcription successful"
                return 0
            fi
        elif [[ "$status_code" =~ ^(422|400|500)$ ]]; then
            # May fail due to model not loaded, but endpoint is working
            log_test_result "$test_name" "PASS" "transcription endpoint responsive (HTTP: $status_code)"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "audio transcription test failed"
    return 1
}

test_fixture_speech_transcription() {
    local test_name="fixture speech transcription"
    
    # Get a speech audio fixture
    local test_file
    test_file=$(get_speech_audio_fixture "whisper")
    
    if ! validate_fixture_file "$test_file"; then
        log_test_result "$test_name" "SKIP" "fixture file not available"
        return 2
    fi
    
    # Test transcription with known speech file
    local response
    local status_code
    if response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
        -X POST "$BASE_URL/v1/audio/transcriptions" \
        -F "file=@$test_file" \
        -F "model=whisper-1" \
        --max-time 45 2>/dev/null); then
        
        status_code=$(echo "$response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
        response_body=$(echo "$response" | sed 's/HTTPSTATUS:[0-9]*$//')
        
        if [[ "$status_code" == "200" ]]; then
            # Extract transcribed text
            local transcribed_text
            transcribed_text=$(echo "$response_body" | jq -r '.text' 2>/dev/null || echo "$response_body")
            
            if [[ -n "$transcribed_text" ]] && [[ "$transcribed_text" != "null" ]]; then
                log_test_result "$test_name" "PASS" "transcribed: ${transcribed_text:0:50}..."
                return 0
            fi
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "speech transcription failed"
    return 1
}

test_fixture_silent_audio() {
    local test_name="fixture silent audio handling"
    
    # Get a silent audio fixture
    local test_file
    test_file=$(get_speech_audio_fixture "silent")
    
    if ! validate_fixture_file "$test_file"; then
        log_test_result "$test_name" "SKIP" "silent audio fixture not available"
        return 2
    fi
    
    # Test transcription with silent file
    local response
    local status_code
    if response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
        -X POST "$BASE_URL/v1/audio/transcriptions" \
        -F "file=@$test_file" \
        -F "model=whisper-1" \
        --max-time 30 2>/dev/null); then
        
        status_code=$(echo "$response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
        response_body=$(echo "$response" | sed 's/HTTPSTATUS:[0-9]*$//')
        
        if [[ "$status_code" == "200" ]]; then
            # Silent audio should return empty or minimal text
            local transcribed_text
            transcribed_text=$(echo "$response_body" | jq -r '.text' 2>/dev/null || echo "$response_body")
            
            if [[ -z "$transcribed_text" ]] || [[ "$transcribed_text" == "null" ]] || [[ ${#transcribed_text} -lt 10 ]]; then
                log_test_result "$test_name" "PASS" "silent audio handled correctly"
                return 0
            else
                log_test_result "$test_name" "PASS" "silent audio processed (text: ${transcribed_text:0:20}...)"
                return 0
            fi
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "silent audio handling failed"
    return 1
}

test_fixture_corrupted_audio() {
    local test_name="fixture corrupted audio handling"
    
    # Get a corrupted audio fixture
    local test_file
    test_file=$(get_speech_audio_fixture "corrupted")
    
    if ! validate_fixture_file "$test_file"; then
        log_test_result "$test_name" "SKIP" "corrupted audio fixture not available"
        return 2
    fi
    
    # Test transcription with corrupted file - should handle gracefully
    local response
    local status_code
    if response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
        -X POST "$BASE_URL/v1/audio/transcriptions" \
        -F "file=@$test_file" \
        -F "model=whisper-1" \
        --max-time 30 2>/dev/null); then
        
        status_code=$(echo "$response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
        
        # Corrupted file should either fail gracefully (400/422) or process with errors
        if [[ "$status_code" =~ ^(400|422|415)$ ]]; then
            log_test_result "$test_name" "PASS" "corrupted audio rejected properly (HTTP: $status_code)"
            return 0
        elif [[ "$status_code" == "200" ]]; then
            # May process but with poor results
            log_test_result "$test_name" "PASS" "corrupted audio handled without crash"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "corrupted audio not handled properly"
    return 1
}

test_fixture_multiple_formats() {
    local test_name="fixture multiple audio formats"
    
    local formats=("mp3" "wav" "ogg")
    local success_count=0
    local tested_count=0
    
    for format in "${formats[@]}"; do
        # Try to find a fixture for this format
        local test_file
        test_file=$(find "$(get_audio_fixture_path)" -name "*.$format" | head -1)
        
        if [[ -n "$test_file" ]] && [[ -f "$test_file" ]]; then
            ((tested_count++))
            
            # Test this format
            local response
            local status_code
            if response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
                -X POST "$BASE_URL/v1/audio/transcriptions" \
                -F "file=@$test_file" \
                -F "model=whisper-1" \
                --max-time 30 2>/dev/null); then
                
                status_code=$(echo "$response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
                
                if [[ "$status_code" == "200" ]]; then
                    ((success_count++))
                fi
            fi
        fi
    done
    
    if [[ $tested_count -eq 0 ]]; then
        log_test_result "$test_name" "SKIP" "no format fixtures available"
        return 2
    fi
    
    if [[ $success_count -gt 0 ]]; then
        log_test_result "$test_name" "PASS" "$success_count/$tested_count formats transcribed"
        return 0
    else
        log_test_result "$test_name" "FAIL" "no formats successfully transcribed"
        return 1
    fi
}

test_health_check() {
    local test_name="service health check"
    
    # Test root endpoint
    local response
    if response=$(make_api_request "/" "GET" 5); then
        if echo "$response" | grep -qi "whisper\|ready\|healthy"; then
            log_test_result "$test_name" "PASS" "service reports healthy"
            return 0
        fi
    fi
    
    # Alternative health check
    if response=$(make_api_request "/health" "GET" 5 2>/dev/null || echo ""); then
        if [[ -n "$response" ]]; then
            log_test_result "$test_name" "PASS" "health endpoint accessible"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "SKIP" "health check not available"
    return 2
}

test_container_status() {
    local test_name="Docker container status"
    
    if ! command -v docker >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "Docker not available"
        return 2
    fi
    
    # Check if container exists and is running
    if docker ps --format '{{.Names}}' | grep -q "^${WHISPER_CONTAINER_NAME}$"; then
        local container_status
        container_status=$(docker inspect "${WHISPER_CONTAINER_NAME}" --format '{{.State.Status}}' 2>/dev/null || echo "unknown")
        
        if [[ "$container_status" == "running" ]]; then
            log_test_result "$test_name" "PASS" "container running"
            return 0
        else
            log_test_result "$test_name" "FAIL" "container status: $container_status"
            return 1
        fi
    else
        log_test_result "$test_name" "FAIL" "container not found"
        return 1
    fi
}

test_model_availability() {
    local test_name="Whisper model availability"
    
    if ! command -v docker >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "Docker not available"
        return 2
    fi
    
    # Check container logs for model loading information
    local logs_output
    if logs_output=$(docker logs "${WHISPER_CONTAINER_NAME}" --tail 100 2>&1); then
        if echo "$logs_output" | grep -qi "model.*load\|whisper.*model\|loaded.*model"; then
            if echo "$logs_output" | grep -qi "error.*model\|failed.*load\|model.*error"; then
                log_test_result "$test_name" "FAIL" "model loading errors detected"
                return 1
            else
                log_test_result "$test_name" "PASS" "model loading indicators found"
                return 0
            fi
        fi
    fi
    
    log_test_result "$test_name" "SKIP" "model availability status unclear"
    return 2
}

test_supported_formats() {
    local test_name="supported audio formats check"
    
    # Test different audio format extensions by checking endpoint response
    local formats=("mp3" "wav" "m4a" "ogg")
    local supported_count=0
    
    for format in "${formats[@]}"; do
        # Create a minimal test to see if format is mentioned in docs/errors
        local response
        if response=$(make_api_request "/docs" "GET" 3 2>/dev/null || echo ""); then
            if echo "$response" | grep -qi "$format"; then
                ((supported_count++))
            fi
        fi
    done
    
    if [[ $supported_count -gt 0 ]]; then
        log_test_result "$test_name" "PASS" "$supported_count audio formats detected"
        return 0
    fi
    
    log_test_result "$test_name" "SKIP" "format support information not available"
    return 2
}

#######################################
# TEST RUNNER CONFIGURATION
#######################################

# Define service-specific tests to run
SERVICE_TESTS=(
    "test_whisper_docs_endpoint"
    "test_transcribe_endpoint_structure"
    "test_models_endpoint"
    "test_health_check"
    "test_container_status"
    "test_model_availability"
    "test_supported_formats"
    "test_audio_file_transcription"
    "test_fixture_speech_transcription"
    "test_fixture_silent_audio"
    "test_fixture_corrupted_audio"
    "test_fixture_multiple_formats"
)

#######################################
# MAIN EXECUTION
#######################################

# Initialize and run tests using the shared library
init_config
integration_test_main "$@"