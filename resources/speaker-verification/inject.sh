#!/usr/bin/env bash
set -euo pipefail

# Speaker Verification Data Injection Adapter
# This script handles injection of speaker profile configurations into the
# Speaker Verification service. Part of the Vrooli resource data injection system.

DESCRIPTION="Inject speaker profile configurations into the Speaker Verification service"

SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
RESOURCE_DIR="${SCRIPT_DIR}"
REPO_ROOT="$(builtin cd "${RESOURCE_DIR}/../.." && builtin pwd)"

# Source var.sh first to get directory variables
# shellcheck disable=SC1091
source "${REPO_ROOT}/scripts/lib/utils/var.sh"

# Source common utilities using var.sh variables
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/system_commands.sh"
# shellcheck disable=SC1091
source "${var_TRASH_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"

# Source Speaker Verification configuration if available
if [[ -f "${SCRIPT_DIR}/config/defaults.sh" ]]; then
    # shellcheck disable=SC1091
    source "${SCRIPT_DIR}/config/defaults.sh"
fi

# Default Speaker Verification settings
readonly DEFAULT_SPEAKER_VERIFICATION_HOST="http://localhost:11452"

# Settings (can be overridden by environment)
SPEAKER_VERIFICATION_HOST="${SPEAKER_VERIFICATION_BASE_URL:-$DEFAULT_SPEAKER_VERIFICATION_HOST}"

#######################################
# Display usage information
#######################################
inject::usage() {
    cat << EOF
Speaker Verification Data Injection Adapter

USAGE:
    $0 [OPTIONS] CONFIG_JSON

DESCRIPTION:
    Injects speaker profile configurations into the Speaker Verification service
    based on scenario configuration. Supports validation, injection, and status
    checks.

OPTIONS:
    --validate    Validate the injection configuration
    --inject      Perform the profile injection
    --status      Check status of the service
    --help        Show this help message

CONFIGURATION FORMAT:
    {
      "profiles": [
        {
          "profile_id": "alice",
          "display_name": "Alice",
          "audio_file": "/path/to/alice.wav",
          "notes": "primary operator"
        }
      ]
    }

EXAMPLES:
    # Validate configuration
    $0 --validate '{"profiles": [{"profile_id": "alice", "audio_file": "/tmp/alice.wav"}]}'

    # Inject profile configurations
    $0 --inject '{"profiles": [{"profile_id": "alice", "audio_file": "/tmp/alice.wav"}]}'
EOF
}

#######################################
# Check if the service is accessible
# Returns: 0 if accessible, 1 otherwise
#######################################
inject::check_accessibility() {
    if curl -s --max-time 5 "${SPEAKER_VERIFICATION_HOST}/ready" >/dev/null 2>&1; then
        log::debug "Speaker Verification is accessible at $SPEAKER_VERIFICATION_HOST"
        return 0
    else
        log::error "Speaker Verification is not accessible at $SPEAKER_VERIFICATION_HOST"
        log::info "Ensure the service is running: resource-speaker-verification manage start"
        return 1
    fi
}

#######################################
# Validate injection configuration
# Arguments:
#   $1 - configuration JSON
# Returns: 0 if valid, 1 if invalid
#######################################
inject::validate_config() {
    local config="$1"

    log::info "Validating Speaker Verification injection configuration..."

    if ! echo "$config" | jq . >/dev/null 2>&1; then
        log::error "Invalid JSON in Speaker Verification injection configuration"
        return 1
    fi

    local has_profiles
    has_profiles=$(echo "$config" | jq -e '.profiles' >/dev/null 2>&1 && echo "true" || echo "false")

    if [[ "$has_profiles" == "false" ]]; then
        log::error "Speaker Verification injection configuration must include 'profiles'"
        return 1
    fi

    local profiles profile_count
    profiles=$(echo "$config" | jq -c '.profiles')
    profile_count=$(echo "$profiles" | jq 'length')

    local i
    for ((i = 0; i < profile_count; i++)); do
        local profile audio_file
        profile=$(echo "$profiles" | jq -c ".[$i]")
        audio_file=$(echo "$profile" | jq -r '.audio_file // empty')

        if [[ -z "$audio_file" ]]; then
            log::error "Profile at index $i missing required 'audio_file' field"
            return 1
        fi
    done

    log::success "Speaker Verification injection configuration is valid"
    return 0
}

#######################################
# Perform data injection
# Arguments:
#   $1 - configuration JSON
# Returns: 0 if successful, 1 if failed
#######################################
inject::inject_data() {
    local config="$1"

    log::header "🔄 Injecting profiles into Speaker Verification"

    if ! inject::check_accessibility; then
        return 1
    fi

    local profiles profile_count
    profiles=$(echo "$config" | jq -c '.profiles')
    profile_count=$(echo "$profiles" | jq 'length')

    local i success=0
    for ((i = 0; i < profile_count; i++)); do
        local profile profile_id display_name notes audio_file
        profile=$(echo "$profiles" | jq -c ".[$i]")
        profile_id=$(echo "$profile" | jq -r '.profile_id // ""')
        display_name=$(echo "$profile" | jq -r '.display_name // ""')
        notes=$(echo "$profile" | jq -r '.notes // ""')
        audio_file=$(echo "$profile" | jq -r '.audio_file // empty')

        if [[ ! -f "$audio_file" ]]; then
            log::error "Audio file not found: $audio_file"
            continue
        fi

        if curl -s -X POST "${SPEAKER_VERIFICATION_HOST}/v1/profiles" \
            -F "profile_id=${profile_id}" \
            -F "display_name=${display_name}" \
            -F "notes=${notes}" \
            -F "audio=@${audio_file}" \
            --max-time 60 >/dev/null 2>&1; then
            log::success "Enrolled profile: ${profile_id:-<generated>}"
            success=$((success + 1))
        else
            log::error "Failed to enroll profile: ${profile_id:-<generated>}"
        fi
    done

    log::success "✅ Speaker Verification injection completed: $success/$profile_count profiles"
    return 0
}

#######################################
# Check injection status
# Arguments:
#   $1 - configuration JSON
# Returns: 0 if successful, 1 if failed
#######################################
inject::check_status() {
    local config="$1"

    log::header "📊 Checking Speaker Verification status"

    if ! inject::check_accessibility; then
        return 1
    fi

    if curl -s "${SPEAKER_VERIFICATION_HOST}/v1/profiles" | jq . >/dev/null 2>&1; then
        log::success "✅ Speaker Verification API is responding"
    else
        log::error "❌ Speaker Verification API not responding properly"
    fi

    return 0
}

#######################################
# Main execution function
#######################################
inject::main() {
    local action="$1"
    local config="${2:-}"

    case "$action" in
        "--help")
            inject::usage
            return 0
            ;;
    esac

    if [[ -z "$config" ]]; then
        log::error "Configuration JSON required"
        inject::usage
        exit 1
    fi

    case "$action" in
        "--validate")
            inject::validate_config "$config"
            ;;
        "--inject")
            inject::inject_data "$config"
            ;;
        "--status")
            inject::check_status "$config"
            ;;
        *)
            log::error "Unknown action: $action"
            inject::usage
            exit 1
            ;;
    esac
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    if [[ $# -eq 0 ]]; then
        inject::usage
        exit 1
    fi

    inject::main "$@"
fi
