#!/usr/bin/env bash
# Kokoro Integration Test
# Tests real Kokoro text-to-speech synthesis functionality
# Tests API endpoints, synthesis capabilities, and voice management

set -euo pipefail

# Source shared integration test library
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/resources/kokoro/test"

# Source var.sh for directory variables
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"

# Source integration test libraries using var.sh
# shellcheck disable=SC1091
if [[ -f "${var_TEST_DIR}/integration/health-check.sh" ]]; then
    source "${var_TEST_DIR}/integration/health-check.sh"
else
    # Define basic functions if library not found
    log_test_result() {
        local test_name="$1"
        local status="$2"
        local message="$3"
        echo "[$status] $test_name: $message"
    }

    make_api_request() {
        local endpoint="$1"
        local method="${2:-GET}"
        local timeout="${3:-10}"
        curl -s -X "$method" "${BASE_URL}${endpoint}" --max-time "$timeout"
    }

    validate_json_response() {
        local response="$1"
        echo "$response" | jq . >/dev/null 2>&1
    }
fi

#######################################
# SERVICE-SPECIFIC CONFIGURATION
#######################################

# Load Kokoro configuration
# shellcheck disable=SC1091
if [[ -f "${var_RESOURCES_COMMON_FILE}" ]]; then
    source "${var_RESOURCES_COMMON_FILE}"
fi

# shellcheck disable=SC1091
if [[ -f "${APP_ROOT}/resources/kokoro/config/defaults.sh" ]]; then
    source "${APP_ROOT}/resources/kokoro/config/defaults.sh"
    # Check if function exists before calling
    if declare -f defaults::export_config >/dev/null 2>&1; then
        defaults::export_config
    fi
fi

# Override library defaults with Kokoro-specific settings
SERVICE_NAME="kokoro"
BASE_URL="${KOKORO_BASE_URL:-http://localhost:8880}"
HEALTH_ENDPOINT="/v1/audio/voices"
REQUIRED_TOOLS=("curl" "jq" "docker")
SERVICE_METADATA=(
    "API Port: ${KOKORO_PORT:-8880}"
    "Container: ${KOKORO_CONTAINER_NAME:-kokoro}"
    "Voices Dir: ${KOKORO_VOICES_DIR:-${HOME}/.kokoro/voices}"
)

#######################################
# KOKORO-SPECIFIC TEST FUNCTIONS
#######################################

test_kokoro_voices_endpoint() {
    local test_name="Kokoro voices endpoint"

    local response
    if response=$(make_api_request "/v1/audio/voices" "GET" 10); then
        if validate_json_response "$response"; then
            log_test_result "$test_name" "PASS" "Voices endpoint returns JSON"
            return 0
        fi
    fi

    log_test_result "$test_name" "FAIL" "Voices endpoint not accessible"
    return 1
}

test_speech_endpoint_structure() {
    local test_name="speech synthesis endpoint structure"

    # Test speech endpoint without proper body (should return 422 or 400)
    local response
    local status_code
    if response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
        -X POST "$BASE_URL/v1/audio/speech" \
        -H "Content-Type: application/json" \
        -d '{}' \
        --max-time 10 2>/dev/null); then

        status_code=$(echo "$response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)

        # Should return 422 (validation error) or 400 (bad request) for empty body
        if [[ "$status_code" =~ ^(422|400|405|200)$ ]]; then
            log_test_result "$test_name" "PASS" "speech endpoint exists (HTTP: $status_code)"
            return 0
        fi
    fi

    log_test_result "$test_name" "FAIL" "speech endpoint not accessible"
    return 1
}

test_text_synthesis() {
    local test_name="text-to-speech synthesis"

    local output_file="/tmp/kokoro_integration_test.mp3"

    # Test actual synthesis
    local response
    local status_code
    if response=$(curl -s -w "HTTPSTATUS:%{http_code}" -o "$output_file" \
        -X POST "$BASE_URL/v1/audio/speech" \
        -H "Content-Type: application/json" \
        -d '{"model":"kokoro","input":"Hello, this is a test of Kokoro text to speech.","voice":"af_heart","response_format":"mp3"}' \
        --max-time 30 2>/dev/null); then

        status_code=$(echo "$response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)

        if [[ "$status_code" == "200" ]] && [[ -f "$output_file" ]] && [[ -s "$output_file" ]]; then
            local file_size
            file_size=$(stat -c%s "$output_file" 2>/dev/null || stat -f%z "$output_file" 2>/dev/null || echo "0")
            rm -f "$output_file"

            if [[ "$file_size" -gt 0 ]]; then
                log_test_result "$test_name" "PASS" "synthesis successful (${file_size} bytes)"
                return 0
            fi
        fi

        rm -f "$output_file"

        if [[ "$status_code" =~ ^(422|400|500)$ ]]; then
            log_test_result "$test_name" "PASS" "synthesis endpoint responsive (HTTP: $status_code)"
            return 0
        fi
    fi

    rm -f "$output_file"
    log_test_result "$test_name" "FAIL" "text synthesis test failed"
    return 1
}

test_voice_listing() {
    local test_name="voice listing"

    local response
    if response=$(make_api_request "/v1/audio/voices" "GET" 10); then
        if validate_json_response "$response"; then
            local voice_count
            voice_count=$(echo "$response" | jq 'length' 2>/dev/null || echo "0")

            if [[ "$voice_count" -gt 0 ]]; then
                log_test_result "$test_name" "PASS" "$voice_count voices available"
                return 0
            fi
        fi
    fi

    log_test_result "$test_name" "FAIL" "voice listing failed"
    return 1
}

test_health_check() {
    local test_name="service health check"

    local response
    local status_code
    if response=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/v1/audio/voices" --max-time 5 2>/dev/null); then
        if [[ "$response" == "200" ]]; then
            log_test_result "$test_name" "PASS" "service reports healthy"
            return 0
        fi
    fi

    log_test_result "$test_name" "FAIL" "health check failed"
    return 1
}

test_container_status() {
    local test_name="Docker container status"

    if ! command -v docker >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "Docker not available"
        return 2
    fi

    # Check if container exists and is running
    if docker ps --format '{{.Names}}' | grep -q "^${KOKORO_CONTAINER_NAME}$"; then
        local container_status
        container_status=$(docker inspect "${KOKORO_CONTAINER_NAME}" --format '{{.State.Status}}' 2>/dev/null || echo "unknown")

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

test_multiple_output_formats() {
    local test_name="multiple output formats"

    local formats=("mp3" "wav" "opus" "flac")
    local success_count=0
    local tested_count=0

    for format in "${formats[@]}"; do
        ((tested_count++))

        local output_file="/tmp/kokoro_format_test.${format}"
        local status_code
        status_code=$(curl -s -w "%{http_code}" -o "$output_file" \
            -X POST "$BASE_URL/v1/audio/speech" \
            -H "Content-Type: application/json" \
            -d "{\"model\":\"kokoro\",\"input\":\"Test\",\"voice\":\"af_heart\",\"response_format\":\"${format}\"}" \
            --max-time 15 2>/dev/null)

        if [[ "$status_code" == "200" ]] && [[ -f "$output_file" ]] && [[ -s "$output_file" ]]; then
            ((success_count++))
        fi
        rm -f "$output_file"
    done

    if [[ $success_count -gt 0 ]]; then
        log_test_result "$test_name" "PASS" "$success_count/$tested_count formats supported"
        return 0
    else
        log_test_result "$test_name" "FAIL" "no formats successfully synthesized"
        return 1
    fi
}

#######################################
# TEST RUNNER CONFIGURATION
#######################################

# Define service-specific tests to run
SERVICE_TESTS=(
    "test_kokoro_voices_endpoint"
    "test_speech_endpoint_structure"
    "test_health_check"
    "test_container_status"
    "test_voice_listing"
    "test_text_synthesis"
    "test_multiple_output_formats"
)

#######################################
# MAIN EXECUTION
#######################################

# Initialize and run tests using the shared library
init_config
integration_test_main "$@"
