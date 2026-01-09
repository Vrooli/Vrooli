#!/usr/bin/env bash
# Codex Common Functions

# Set script directory for sourcing
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
CODEX_COMMON_DIR="${APP_ROOT}/resources/codex/lib"

# Source configuration
# shellcheck disable=SC1091
source "${APP_ROOT}/resources/codex/config/defaults.sh"

# Source shared utilities
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/log.sh"

#######################################
# Determine writable Codex home directory
# Returns:
#   Path to writable Codex home directory
#######################################
codex::ensure_home() {
    if [[ -n "${CODEX_EFFECTIVE_HOME:-}" && -d "${CODEX_EFFECTIVE_HOME}" ]]; then
        echo "${CODEX_EFFECTIVE_HOME}"
        return 0
    fi

    local preferred_home="${CODEX_HOME:-${HOME}/.codex}"
    local resolved_home=""

    if mkdir -p "${preferred_home}" 2>/dev/null; then
        local test_file="${preferred_home}/.codex-write-test"
        if touch "${test_file}" 2>/dev/null; then
            rm -f "${test_file}"
            resolved_home="${preferred_home}"
            CODEX_HOME_OVERRIDE_REQUIRED="false"
        else
            resolved_home=""
        fi
    fi

    if [[ -z "${resolved_home}" ]]; then
        local workspace_base="${CODEX_WORKSPACE:-/tmp/codex-workspace}"
        local fallback_home="${workspace_base}/.codex-home"
        if mkdir -p "${fallback_home}" 2>/dev/null; then
            resolved_home="${fallback_home}"
            log::warn "Codex home ${preferred_home} is not writable. Using fallback: ${resolved_home}"
        else
            resolved_home="/tmp/codex-home"
            mkdir -p "${resolved_home}" 2>/dev/null || true
            log::warn "Codex home not writable. Falling back to ${resolved_home}"
        fi
        CODEX_HOME_OVERRIDE_REQUIRED="true"
    fi

    mkdir -p "${resolved_home}/sessions" "${resolved_home}/log" "${resolved_home}/logs" \
        "${resolved_home}/scripts" "${resolved_home}/outputs" 2>/dev/null || true

    export CODEX_HOME="${resolved_home}"
    CODEX_EFFECTIVE_HOME="${resolved_home}"

    echo "${resolved_home}"
}

#######################################
# Get OpenAI API key from Vault or environment
# Returns:
#   API key string or empty if not found
#######################################
codex::get_api_key() {
    local api_key=""
    
    # Try environment variable first
    if [[ -n "${OPENAI_API_KEY:-}" ]]; then
        api_key="${OPENAI_API_KEY}"
    # Try Vault using resource-vault command
    elif command -v resource-vault &>/dev/null; then
        api_key=$(resource-vault content get --path "resources/codex/api/openai" --key "api_key" --format raw 2>/dev/null || echo "")
    # Try credentials file
    elif [[ -f "${HOME}/.openai/credentials" ]]; then
        api_key=$(grep "api_key=" "${HOME}/.openai/credentials" | cut -d= -f2)
    fi
    
    echo "${api_key}"
}

#######################################
# Check if Codex is configured
# Returns:
#   0 if configured, 1 otherwise
#######################################
codex::is_configured() {
    local api_key
    api_key=$(codex::get_api_key)
    
    if [[ -n "${api_key}" ]]; then
        return 0
    else
        return 1
    fi
}

#######################################
# Check if Codex service is available
# Uses `codex login status` to verify authentication
# Returns:
#   0 if available (logged in)
#   1 if unavailable (not logged in or CLI not found)
#   2 if unknown (can't determine status)
#######################################
codex::is_available() {
    # First check if codex CLI is installed
    if ! command -v codex &>/dev/null; then
        return 1
    fi

    # Use codex login status to check authentication
    # This doesn't use any API credits
    local login_output
    login_output=$(timeout 5 codex login status 2>&1) || true
    local exit_code=$?

    # Check if logged in (output contains "Logged in")
    if [[ "$login_output" =~ "Logged in" ]]; then
        return 0
    fi

    # Check if explicitly not logged in
    if [[ "$login_output" =~ "Not logged in" ]] || [[ "$login_output" =~ "not logged in" ]]; then
        return 1
    fi

    # Timeout or other issue - return unknown
    if [[ $exit_code -eq 124 ]]; then
        # Timeout
        return 2
    fi

    # If we can't determine status, return unknown (2) instead of failing
    # This prevents showing "unhealthy" when we just don't know
    return 2
}

#######################################
# Get Codex version/model info
# Returns:
#   Model name or "unknown"
#######################################
codex::get_version() {
    echo "${CODEX_DEFAULT_MODEL}"
}

#######################################
# Save status to file
# Arguments:
#   status - Status to save (running/stopped)
#######################################
codex::save_status() {
    local status="${1:-stopped}"
    echo "${status}" > "${CODEX_STATUS_FILE}"
}

#######################################
# Detect rate limit from API response
# Arguments:
#   $1 - API response (JSON or text)
#   $2 - HTTP status code (optional)
# Returns:
#   JSON with rate limit detection info
#######################################
codex::detect_rate_limit() {
    local response="${1:-}"
    local http_code="${2:-0}"

    local detected="false"
    local limit_type="unknown"
    local retry_after="300"
    local reset_time=""
    local error_message=""

    # Check for 429 status code
    if [[ "$http_code" == "429" ]]; then
        detected="true"
        error_message="Rate limit exceeded (HTTP 429)"
    fi

    # Check for rate limit in JSON error response
    if echo "$response" | jq -e '.error' &>/dev/null; then
        local error_type
        error_type=$(echo "$response" | jq -r '.error.type // ""' 2>/dev/null)
        local error_code
        error_code=$(echo "$response" | jq -r '.error.code // ""' 2>/dev/null)

        # OpenAI rate limit error types
        if [[ "$error_type" == "rate_limit_exceeded" ]] || [[ "$error_code" == "rate_limit_exceeded" ]]; then
            detected="true"
            error_message=$(echo "$response" | jq -r '.error.message // "Rate limit exceeded"' 2>/dev/null)

            # Try to extract limit type from message
            if [[ "$error_message" =~ RPM|requests.*minute ]]; then
                limit_type="requests_per_minute"
                retry_after="60"
            elif [[ "$error_message" =~ TPM|tokens.*minute ]]; then
                limit_type="tokens_per_minute"
                retry_after="60"
            elif [[ "$error_message" =~ RPD|requests.*day ]]; then
                limit_type="requests_per_day"
                retry_after="3600"
            elif [[ "$error_message" =~ TPD|tokens.*day ]]; then
                limit_type="tokens_per_day"
                retry_after="3600"
            fi
        fi
    fi

    # Check for rate limit keywords in text response
    if [[ "$response" =~ (rate.*limit|Rate.*limit|quota.*exceeded|too.*many.*requests) ]]; then
        detected="true"
        if [[ -z "$error_message" ]]; then
            error_message="Rate limit detected in response"
        fi

        # Try to parse retry time from message
        if [[ "$response" =~ ([0-9]+).*second ]]; then
            retry_after="${BASH_REMATCH[1]}"
        elif [[ "$response" =~ ([0-9]+).*minute ]]; then
            retry_after=$((${BASH_REMATCH[1]} * 60))
        elif [[ "$response" =~ ([0-9]+).*hour ]]; then
            retry_after=$((${BASH_REMATCH[1]} * 3600))
        fi
    fi

    # Calculate reset time
    if [[ "$detected" == "true" ]]; then
        reset_time=$(date -u -d "+${retry_after} seconds" "+%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || date -u "+%Y-%m-%dT%H:%M:%SZ")
    fi

    # Return JSON
    cat <<EOF
{
    "detected": $detected,
    "limit_type": "$limit_type",
    "retry_after_seconds": $retry_after,
    "reset_time": "$reset_time",
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "error_message": $(echo "$error_message" | jq -R -s '.' 2>/dev/null || echo '""')
}
EOF
}

#######################################
# Record rate limit event to status file
# Arguments:
#   $1 - Rate limit info JSON (from detect_rate_limit)
#######################################
codex::record_rate_limit() {
    local rate_info="${1:-}"

    if [[ -z "$rate_info" ]]; then
        return 1
    fi

    # Check if rate limit was detected
    local detected
    detected=$(echo "$rate_info" | jq -r '.detected // false')

    if [[ "$detected" != "true" ]]; then
        return 0
    fi

    # Get or create state file
    local codex_home
    codex_home=$(codex::ensure_home)
    local state_file="${codex_home}/state.json"

    # Initialize state file if it doesn't exist
    if [[ ! -f "$state_file" ]]; then
        echo '{}' > "$state_file"
    fi

    # Update state with rate limit info
    local updated_state
    updated_state=$(jq --argjson rate_limit "$rate_info" \
        '. + {last_rate_limit: $rate_limit, rate_limited: true}' \
        "$state_file" 2>/dev/null || echo '{"last_rate_limit": null, "rate_limited": false}')

    echo "$updated_state" > "$state_file"

    log::warn "⚠️  Rate limit detected - reset at $(echo "$rate_info" | jq -r '.reset_time')"
}
