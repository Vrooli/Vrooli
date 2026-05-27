#!/usr/bin/env bash
################################################################################
# Kyutai STT API Functions
#
# Helpers for interacting with the Kyutai STT HTTP/WebSocket API.
################################################################################

#######################################
# Test API connectivity (health + info)
#######################################
kyutai_stt::test_api() {
    local base_url="${KYUTAI_STT_BASE_URL:-http://localhost:8094}"

    log::info "Testing Kyutai STT API connectivity..."

    # Health endpoint
    local response
    response=$(curl -s -o /dev/null -w "%{http_code}" "$base_url/health" --max-time "$KYUTAI_STT_API_TIMEOUT" 2>/dev/null)

    if [[ "$response" == "200" ]]; then
        log::success "✅ Health endpoint is accessible"
    else
        log::error "❌ Cannot connect to API at $base_url (HTTP: $response)"
        return 1
    fi

    # Info endpoint
    response=$(curl -s -o /dev/null -w "%{http_code}" "$base_url/v1/info" --max-time "$KYUTAI_STT_API_TIMEOUT" 2>/dev/null)
    if [[ "$response" == "200" ]]; then
        log::success "✅ Info endpoint responsive"
    else
        log::warn "⚠️  Info endpoint returned: $response"
    fi

    log::info "API test completed"
}

#######################################
# Fetch and print /v1/info
#######################################
kyutai_stt::get_info() {
    local base_url="${KYUTAI_STT_BASE_URL:-http://localhost:8094}"

    log::info "Fetching Kyutai STT info..."

    local response
    response=$(curl -s "$base_url/v1/info" --max-time "$KYUTAI_STT_API_TIMEOUT" 2>/dev/null)

    if [[ -n "$response" ]] && echo "$response" | jq . >/dev/null 2>&1; then
        echo "$response" | jq .
        return 0
    fi

    log::error "Failed to fetch info from Kyutai STT API"
    log::info "Check if Kyutai STT is running: resource-kyutai-stt status"
    return 1
}

#######################################
# Print static API information / endpoints
#######################################
kyutai_stt::get_api_info() {
    echo "Kyutai STT API Information:"
    echo ""
    echo "Base URL: ${KYUTAI_STT_BASE_URL:-http://localhost:8094}"
    echo "WebSocket URL: ${KYUTAI_STT_WS_URL:-ws://localhost:8094/v1/stream}"
    echo "Port: ${KYUTAI_STT_PORT:-8094}"
    echo ""
    echo "Endpoints:"
    echo "  GET  /health      Liveness + model_loaded + device"
    echo "  GET  /v1/info     Backend, model id, device, sample_rate, version"
    echo "  WS   /v1/stream   Streaming transcription (PCM s16le 16kHz mono in)"
    echo ""
    echo "Stream protocol (client -> server):"
    echo "  TEXT  {\"type\":\"start\",\"sample_rate\":16000,\"language\":\"en\"}"
    echo "  BINARY  raw little-endian 16-bit PCM mono @ 16kHz"
    echo "  TEXT  {\"type\":\"end\"}"
    echo ""
    echo "Stream protocol (server -> client, JSON TEXT frames):"
    echo "  {\"type\":\"partial\",\"text\":...}"
    echo "  {\"type\":\"segment\",\"text\":...,\"start_ms\":...,\"end_ms\":...}"
    echo "  {\"type\":\"done\"}"
    echo "  {\"type\":\"error\",\"message\":...}"
    echo ""
    echo "Model: ${KYUTAI_STT_HF_REPO:-kyutai/stt-1b-en_fr} (loaded once at container startup)"
}

# Export functions for subshell availability
export -f kyutai_stt::test_api
export -f kyutai_stt::get_info
export -f kyutai_stt::get_api_info
