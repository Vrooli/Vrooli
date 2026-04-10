#!/usr/bin/env bash
# Speaker Verification - API Interaction Functions

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

#######################################
# Get service info
# Returns: JSON with backend, model, device info
#######################################
speaker_verification::api::info() {
    local response
    if response=$(timeout "${SPEAKER_VERIFICATION_API_TIMEOUT}" curl -sf "${SPEAKER_VERIFICATION_BASE_URL}/v1/info" 2>/dev/null); then
        echo "$response"
        return 0
    else
        log::error "$MSG_API_UNREACHABLE"
        return 1
    fi
}
export -f speaker_verification::api::info

#######################################
# List profiles via API
# Returns: JSON array of profiles
#######################################
speaker_verification::api::list_profiles() {
    local response
    if response=$(timeout "${SPEAKER_VERIFICATION_API_TIMEOUT}" curl -sf "${SPEAKER_VERIFICATION_BASE_URL}/v1/profiles" 2>/dev/null); then
        echo "$response"
        return 0
    else
        log::error "$MSG_API_UNREACHABLE"
        return 1
    fi
}
export -f speaker_verification::api::list_profiles

#######################################
# Get a specific profile via API
# Arguments: profile_id
# Returns: JSON profile data
#######################################
speaker_verification::api::get_profile() {
    local profile_id="${1:?Profile ID required}"
    local response
    if response=$(timeout "${SPEAKER_VERIFICATION_API_TIMEOUT}" curl -sf "${SPEAKER_VERIFICATION_BASE_URL}/v1/profiles/${profile_id}" 2>/dev/null); then
        echo "$response"
        return 0
    else
        log::error "$MSG_PROFILE_NOT_FOUND"
        return 1
    fi
}
export -f speaker_verification::api::get_profile

#######################################
# Delete a profile via API
# Arguments: profile_id
# Returns: 0 on success, 1 on failure
#######################################
speaker_verification::api::delete_profile() {
    local profile_id="${1:?Profile ID required}"
    if timeout "${SPEAKER_VERIFICATION_API_TIMEOUT}" curl -sf -X DELETE "${SPEAKER_VERIFICATION_BASE_URL}/v1/profiles/${profile_id}" &>/dev/null; then
        return 0
    else
        log::error "$MSG_PROFILE_NOT_FOUND"
        return 1
    fi
}
export -f speaker_verification::api::delete_profile

#######################################
# Enroll a speaker profile via API
# Arguments: profile_id, audio_file [, display_name]
# Returns: JSON enrollment result
#######################################
speaker_verification::api::enroll() {
    local profile_id="${1:?Profile ID required}"
    local audio_file="${2:?Audio file required}"
    local display_name="${3:-$profile_id}"

    if [[ ! -f "$audio_file" ]]; then
        log::error "Audio file not found: $audio_file"
        return 1
    fi

    local response
    if response=$(timeout "${SPEAKER_VERIFICATION_API_TIMEOUT}" curl -sf \
        -X POST \
        -F "audio=@${audio_file}" \
        -F "profile_id=${profile_id}" \
        -F "display_name=${display_name}" \
        "${SPEAKER_VERIFICATION_BASE_URL}/v1/profiles" 2>/dev/null); then
        echo "$response"
        return 0
    else
        log::error "Enrollment failed"
        return 1
    fi
}
export -f speaker_verification::api::enroll

#######################################
# Verify audio against a stored profile
# Arguments: profile_id, audio_file [, threshold]
# Returns: JSON verification result
#######################################
speaker_verification::api::verify() {
    local profile_id="${1:?Profile ID required}"
    local audio_file="${2:?Audio file required}"
    local threshold="${3:-}"

    if [[ ! -f "$audio_file" ]]; then
        log::error "Audio file not found: $audio_file"
        return 1
    fi

    local curl_args=(
        "-sf"
        "-X" "POST"
        "-F" "audio=@${audio_file}"
        "-F" "profile_id=${profile_id}"
    )

    if [[ -n "$threshold" ]]; then
        curl_args+=("-F" "threshold=${threshold}")
    fi

    curl_args+=("${SPEAKER_VERIFICATION_BASE_URL}/v1/verify")

    local response
    if response=$(timeout "${SPEAKER_VERIFICATION_API_TIMEOUT}" curl "${curl_args[@]}" 2>/dev/null); then
        echo "$response"
        return 0
    else
        log::error "Verification failed"
        return 1
    fi
}
export -f speaker_verification::api::verify

#######################################
# Extract embeddings (debug endpoint)
# Arguments: audio_file
# Returns: JSON with embedding data
#######################################
speaker_verification::api::embeddings() {
    local audio_file="${1:?Audio file required}"

    if [[ ! -f "$audio_file" ]]; then
        log::error "Audio file not found: $audio_file"
        return 1
    fi

    local response
    if response=$(timeout "${SPEAKER_VERIFICATION_API_TIMEOUT}" curl -sf \
        -X POST \
        -F "audio=@${audio_file}" \
        "${SPEAKER_VERIFICATION_BASE_URL}/v1/embeddings" 2>/dev/null); then
        echo "$response"
        return 0
    else
        log::error "Embedding extraction failed"
        return 1
    fi
}
export -f speaker_verification::api::embeddings
