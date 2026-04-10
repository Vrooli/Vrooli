#!/usr/bin/env bash
################################################################################
# Kokoro API Functions
#
# Functions for interacting with the Kokoro TTS API
################################################################################

#######################################
# Synthesize text to speech using Kokoro API
# Arguments:
#   $1 - Text to synthesize
#   $2 - Output file path [optional, default: /tmp/kokoro_output.mp3]
#   $3 - Voice [optional, default: KOKORO_DEFAULT_VOICE]
#   $4 - Output format [optional, default: mp3]
# Outputs: Audio file at specified path
#######################################
kokoro::synthesize_text() {
    local text="${1:-}"
    local output_file="${2:-/tmp/kokoro_output.mp3}"
    local voice="${3:-${KOKORO_DEFAULT_VOICE:-af_heart}}"
    local response_format="${4:-mp3}"

    if [[ -z "$text" ]]; then
        log::error "No text specified for synthesis"
        log::info "Usage: kokoro::synthesize_text 'text to speak' [output_file] [voice] [format]"
        return 1
    fi

    # Check if service is healthy
    if ! kokoro::is_healthy; then
        log::error "Kokoro service is not available"
        log::info "Start it with: resource-kokoro manage start"
        return 1
    fi

    log::info "Synthesizing text with voice: $voice"
    log::info "Output format: $response_format"
    log::info "Output file: $output_file"

    # Prepare JSON payload
    local payload
    payload=$(jq -n \
        --arg model "kokoro" \
        --arg input "$text" \
        --arg voice "$voice" \
        --arg response_format "$response_format" \
        '{model: $model, input: $input, voice: $voice, response_format: $response_format}')

    # Execute synthesis with progress
    log::info "Processing..."
    local start_time
    start_time=$(date +%s)

    local http_code
    http_code=$(curl -s -w "%{http_code}" -o "$output_file" \
        -X POST "${KOKORO_BASE_URL:-http://localhost:8880}/v1/audio/speech" \
        -H "Content-Type: application/json" \
        -d "$payload" \
        --max-time 60 2>/dev/null)

    local end_time
    end_time=$(date +%s)
    local duration=$((end_time - start_time))

    if [[ "$http_code" == "200" ]] && [[ -f "$output_file" ]] && [[ -s "$output_file" ]]; then
        local file_size
        file_size=$(du -h "$output_file" | cut -f1)
        log::success "✅ Synthesis completed in ${duration}s"
        log::info "Output file: $output_file ($file_size)"
        return 0
    else
        log::error "❌ Synthesis failed (HTTP: $http_code)"
        if [[ -f "$output_file" ]]; then
            local error_content
            error_content=$(cat "$output_file" 2>/dev/null)
            if [[ -n "$error_content" ]]; then
                log::error "Response: $error_content"
            fi
            rm -f "$output_file"
        fi
        return 1
    fi
}

#######################################
# List available voices from Kokoro API
# Outputs: JSON list of available voices
#######################################
kokoro::list_voices() {
    log::info "Fetching available voices..."

    local response
    response=$(curl -s "$KOKORO_BASE_URL/v1/audio/voices" \
        --max-time "$KOKORO_API_TIMEOUT" 2>/dev/null)

    if [[ -n "$response" ]]; then
        if echo "$response" | jq . >/dev/null 2>&1; then
            echo "Available Kokoro Voices:"
            echo ""
            echo "$response" | jq -r '.[] // empty' 2>/dev/null || echo "$response" | jq '.' 2>/dev/null
            echo ""
            echo "Default voice: ${KOKORO_DEFAULT_VOICE:-af_heart}"
            return 0
        else
            echo "$response"
            return 0
        fi
    else
        log::error "Failed to fetch voices from Kokoro API"
        log::info "Check if Kokoro is running: resource-kokoro status"
        return 1
    fi
}

#######################################
# Test API connectivity
#######################################
kokoro::test_api() {
    local base_url="${KOKORO_BASE_URL:-http://localhost:8880}"

    log::info "Testing Kokoro API connectivity..."

    # Test voices endpoint (health check)
    local response
    response=$(curl -s -o /dev/null -w "%{http_code}" "$base_url/v1/audio/voices" --max-time "$KOKORO_API_TIMEOUT" 2>/dev/null)

    if [[ "$response" == "200" ]]; then
        log::success "✅ API is accessible (voices endpoint)"
    else
        log::error "❌ Cannot connect to API at $base_url (HTTP: $response)"
        return 1
    fi

    # Test speech endpoint (should return 422 without body)
    response=$(curl -s -o /dev/null -w "%{http_code}" \
        -X POST "$base_url/v1/audio/speech" \
        -H "Content-Type: application/json" \
        -d '{}' \
        --max-time "$KOKORO_API_TIMEOUT" 2>/dev/null)

    if [[ "$response" =~ ^(200|400|422)$ ]]; then
        log::success "✅ Speech synthesis endpoint responsive"
    else
        log::warn "⚠️  Speech synthesis endpoint returned: $response"
    fi

    log::info "API test completed"
}

#######################################
# Get API information
#######################################
kokoro::get_api_info() {
    echo "Kokoro TTS API Information:"
    echo ""
    echo "Base URL: ${KOKORO_BASE_URL:-http://localhost:8880}"
    echo "Port: ${KOKORO_PORT:-8880}"
    echo "Speech synthesis endpoint: POST /v1/audio/speech"
    echo "List voices endpoint: GET /v1/audio/voices"
    echo ""
    echo "Supported output formats:"
    echo "  mp3, wav, opus, flac"
    echo ""
    echo "Default voice: ${KOKORO_DEFAULT_VOICE:-af_heart}"
    echo "Model: Kokoro 82M (single model, no selection needed)"
    echo ""
    echo "OpenAI-compatible API - can be used as a drop-in replacement"
}

# Export functions for subshell availability
export -f kokoro::synthesize_text
export -f kokoro::list_voices
export -f kokoro::test_api
export -f kokoro::get_api_info
