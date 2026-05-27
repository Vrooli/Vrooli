#!/usr/bin/env bash
################################################################################
# Speaker Verification API Functions
#
# Functions for interacting with the Speaker Verification HTTP API
################################################################################

#######################################
# Get backend / model info from the service
# Outputs: JSON from /v1/info
#######################################
speaker_verification::get_info() {
    if ! speaker_verification::is_healthy; then
        log::error "Speaker Verification service is not available"
        log::info "Start it with: resource-speaker-verification manage start"
        return 1
    fi

    local response
    response=$(curl -s "$SPEAKER_VERIFICATION_BASE_URL/v1/info" \
        --max-time "$SPEAKER_VERIFICATION_API_TIMEOUT" 2>/dev/null)

    if [[ -n "$response" ]] && echo "$response" | jq . >/dev/null 2>&1; then
        echo "$response" | jq .
        return 0
    fi

    log::error "Failed to fetch info from Speaker Verification API"
    return 1
}

#######################################
# List enrolled speaker profiles
# Outputs: JSON from /v1/profiles
#######################################
speaker_verification::list_profiles() {
    if ! speaker_verification::is_healthy; then
        log::error "Speaker Verification service is not available"
        return 1
    fi

    local response
    response=$(curl -s "$SPEAKER_VERIFICATION_BASE_URL/v1/profiles" \
        --max-time "$SPEAKER_VERIFICATION_API_TIMEOUT" 2>/dev/null)

    if [[ -n "$response" ]] && echo "$response" | jq . >/dev/null 2>&1; then
        echo "$response" | jq .
        return 0
    fi

    log::error "Failed to list profiles from Speaker Verification API"
    return 1
}

#######################################
# Enroll a speaker profile from an audio file
# Arguments:
#   $1 - audio file path
#   $2 - profile_id (optional; server generates one when empty)
#   $3 - display_name (optional)
#   $4 - notes (optional)
# Outputs: JSON enrollment response
#######################################
speaker_verification::enroll() {
    local file="${1:-}"
    local profile_id="${2:-}"
    local display_name="${3:-}"
    local notes="${4:-}"

    if [[ -z "$file" || ! -f "$file" ]]; then
        log::error "Audio file not found: ${file:-<none>}"
        return 1
    fi

    if ! speaker_verification::is_healthy; then
        log::error "Speaker Verification service is not available"
        return 1
    fi

    curl -s -X POST "$SPEAKER_VERIFICATION_BASE_URL/v1/profiles" \
        -F "profile_id=${profile_id}" \
        -F "display_name=${display_name}" \
        -F "notes=${notes}" \
        -F "audio=@${file}" \
        --max-time 60 2>/dev/null | jq .
}

#######################################
# Verify an audio file against an enrolled profile
# Arguments:
#   $1 - audio file path
#   $2 - profile_id
#   $3 - threshold (optional, default 0.25)
# Outputs: JSON verify response
#######################################
speaker_verification::verify() {
    local file="${1:-}"
    local profile_id="${2:-}"
    local threshold="${3:-0.25}"

    if [[ -z "$file" || ! -f "$file" ]]; then
        log::error "Audio file not found: ${file:-<none>}"
        return 1
    fi
    if [[ -z "$profile_id" ]]; then
        log::error "profile_id is required for verification"
        return 1
    fi

    if ! speaker_verification::is_healthy; then
        log::error "Speaker Verification service is not available"
        return 1
    fi

    curl -s -X POST "$SPEAKER_VERIFICATION_BASE_URL/v1/verify" \
        -F "profile_id=${profile_id}" \
        -F "threshold=${threshold}" \
        -F "audio=@${file}" \
        --max-time 60 2>/dev/null | jq .
}

#######################################
# Delete an enrolled speaker profile
# Arguments:
#   $1 - profile_id
#######################################
speaker_verification::delete_profile() {
    local profile_id="${1:-}"

    if [[ -z "$profile_id" ]]; then
        log::error "profile_id is required for deletion"
        return 1
    fi

    local status_code
    status_code=$(curl -s -o /dev/null -w "%{http_code}" \
        -X DELETE "$SPEAKER_VERIFICATION_BASE_URL/v1/profiles/${profile_id}" \
        --max-time "$SPEAKER_VERIFICATION_API_TIMEOUT" 2>/dev/null)

    if [[ "$status_code" == "200" ]]; then
        log::success "✅ Deleted profile: $profile_id"
        return 0
    fi

    log::error "Failed to delete profile $profile_id (HTTP: $status_code)"
    return 1
}

#######################################
# Get API information
#######################################
speaker_verification::get_api_info() {
    echo "Speaker Verification API Information:"
    echo ""
    echo "Base URL: ${SPEAKER_VERIFICATION_BASE_URL:-http://localhost:11452}"
    echo "Port: ${SPEAKER_VERIFICATION_PORT:-11452}"
    echo "Readiness:       GET    /ready"
    echo "Info:            GET    /v1/info"
    echo "List profiles:   GET    /v1/profiles"
    echo "Enroll:          POST   /v1/profiles        (multipart: profile_id, display_name, notes, audio)"
    echo "Verify:          POST   /v1/verify          (multipart: profile_id, threshold, audio)"
    echo "Extract:         POST   /v1/extract         (reserved; returns HTTP 501)"
    echo "Delete profile:  DELETE /v1/profiles/{id}"
    echo ""
    echo "Model: ${SPEAKER_VERIFICATION_MODEL:-speechbrain/spkrec-ecapa-voxceleb}"
    echo "Embedding dim: ${SPEAKER_VERIFICATION_EMBEDDING_DIM:-192}"
    echo "Sample rate: ${SPEAKER_VERIFICATION_SAMPLE_RATE:-16000} Hz"
}

# Export functions for subshell availability
export -f speaker_verification::get_info
export -f speaker_verification::list_profiles
export -f speaker_verification::enroll
export -f speaker_verification::verify
export -f speaker_verification::delete_profile
export -f speaker_verification::get_api_info
