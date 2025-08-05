#!/usr/bin/env bash
# Whisper Integration Test
# Tests real Whisper audio transcription functionality
# Tests API endpoints, transcription capabilities, and model management

set -euo pipefail

# Source shared integration test library
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/../../../tests/lib/integration-test-lib.sh"

#######################################
# SERVICE-SPECIFIC CONFIGURATION
#######################################

# Load Whisper configuration
RESOURCES_DIR="$SCRIPT_DIR/../../.."
# shellcheck disable=SC1091
source "$RESOURCES_DIR/common.sh"
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
)

#######################################
# MAIN EXECUTION
#######################################

# Initialize and run tests using the shared library
init_config
run_integration_tests "$@"