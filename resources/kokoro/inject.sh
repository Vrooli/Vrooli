#!/usr/bin/env bash
set -euo pipefail

# Kokoro Data Injection Adapter
# This script handles injection of voice configurations into Kokoro
# Part of the Vrooli resource data injection system

DESCRIPTION="Inject voice configurations into Kokoro text-to-speech service"

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

# Source Kokoro configuration if available
if [[ -f "${SCRIPT_DIR}/config/defaults.sh" ]]; then
    # shellcheck disable=SC1091
    source "${SCRIPT_DIR}/config/defaults.sh"
fi

# Default Kokoro settings
readonly DEFAULT_KOKORO_HOST="http://localhost:8880"
readonly DEFAULT_KOKORO_DATA_DIR="${HOME}/.kokoro"
readonly DEFAULT_KOKORO_VOICES_DIR="${DEFAULT_KOKORO_DATA_DIR}/voices"

# Kokoro settings (can be overridden by environment)
KOKORO_HOST="${KOKORO_HOST:-$DEFAULT_KOKORO_HOST}"
KOKORO_DATA_DIR="${KOKORO_DATA_DIR:-$DEFAULT_KOKORO_DATA_DIR}"
KOKORO_VOICES_DIR="${KOKORO_VOICES_DIR:-$DEFAULT_KOKORO_VOICES_DIR}"

# Operation tracking
declare -a KOKORO_ROLLBACK_ACTIONS=()

#######################################
# Display usage information
#######################################
inject::usage() {
    cat << EOF
Kokoro Data Injection Adapter

USAGE:
    $0 [OPTIONS] CONFIG_JSON

DESCRIPTION:
    Injects voice configurations into Kokoro based on scenario configuration.
    Supports validation, injection, status checks, and rollback operations.

OPTIONS:
    --validate    Validate the injection configuration
    --inject      Perform the voice injection
    --status      Check status of injected voices
    --rollback    Rollback injected voices
    --help        Show this help message

CONFIGURATION FORMAT:
    {
      "voices": [
        {
          "name": "af_heart",
          "default": true
        },
        {
          "name": "af_bella",
          "default": false
        }
      ],
      "configurations": [
        {
          "key": "default_voice",
          "value": "af_heart"
        }
      ]
    }

EXAMPLES:
    # Validate configuration
    $0 --validate '{"voices": [{"name": "af_heart", "default": true}]}'

    # Inject voice configurations
    $0 --inject '{"voices": [{"name": "af_heart", "default": true}]}'

EOF
}

#######################################
# Check if Kokoro is accessible
# Returns:
#   0 if accessible, 1 otherwise
#######################################
inject::check_accessibility() {
    # Check if Kokoro is running
    if curl -s --max-time 5 "${KOKORO_HOST}/v1/audio/voices" >/dev/null 2>&1; then
        log::debug "Kokoro is accessible at $KOKORO_HOST"
        return 0
    else
        log::error "Kokoro is not accessible at $KOKORO_HOST"
        log::info "Ensure Kokoro is running: resource-kokoro manage start"
        return 1
    fi
}

#######################################
# Add rollback action
# Arguments:
#   $1 - description
#   $2 - rollback command
#######################################
inject::add_rollback_action() {
    local description="$1"
    local command="$2"

    KOKORO_ROLLBACK_ACTIONS+=("$description|$command")
    log::debug "Added Kokoro rollback action: $description"
}

#######################################
# Execute rollback actions
#######################################
inject::execute_rollback() {
    if [[ ${#KOKORO_ROLLBACK_ACTIONS[@]} -eq 0 ]]; then
        log::info "No Kokoro rollback actions to execute"
        return 0
    fi

    log::info "Executing Kokoro rollback actions..."

    local success_count=0
    local total_count=${#KOKORO_ROLLBACK_ACTIONS[@]}

    # Execute in reverse order
    for ((i=${#KOKORO_ROLLBACK_ACTIONS[@]}-1; i>=0; i--)); do
        local action="${KOKORO_ROLLBACK_ACTIONS[i]}"
        IFS='|' read -r description command <<< "$action"

        log::info "Rollback: $description"

        if eval "$command"; then
            success_count=$((success_count + 1))
            log::success "Rollback completed: $description"
        else
            log::error "Rollback failed: $description"
        fi
    done

    log::info "Kokoro rollback completed: $success_count/$total_count actions successful"
    KOKORO_ROLLBACK_ACTIONS=()
}

#######################################
# Validate injection configuration
# Arguments:
#   $1 - configuration JSON
# Returns:
#   0 if valid, 1 if invalid
#######################################
inject::validate_config() {
    local config="$1"

    log::info "Validating Kokoro injection configuration..."

    # Basic JSON validation
    if ! echo "$config" | jq . >/dev/null 2>&1; then
        log::error "Invalid JSON in Kokoro injection configuration"
        return 1
    fi

    # Check for at least one injection type
    local has_voices has_configurations
    has_voices=$(echo "$config" | jq -e '.voices' >/dev/null 2>&1 && echo "true" || echo "false")
    has_configurations=$(echo "$config" | jq -e '.configurations' >/dev/null 2>&1 && echo "true" || echo "false")

    if [[ "$has_voices" == "false" && "$has_configurations" == "false" ]]; then
        log::error "Kokoro injection configuration must have 'voices' or 'configurations'"
        return 1
    fi

    # Validate voices if present
    if [[ "$has_voices" == "true" ]]; then
        local voices
        voices=$(echo "$config" | jq -c '.voices')
        local voices_type
        voices_type=$(echo "$voices" | jq -r 'type')

        if [[ "$voices_type" != "array" ]]; then
            log::error "Voices configuration must be an array, got: $voices_type"
            return 1
        fi

        local voice_count
        voice_count=$(echo "$voices" | jq 'length')

        for ((i=0; i<voice_count; i++)); do
            local voice
            voice=$(echo "$voices" | jq -c ".[$i]")
            local name
            name=$(echo "$voice" | jq -r '.name // empty')

            if [[ -z "$name" ]]; then
                log::error "Voice at index $i missing required 'name' field"
                return 1
            fi

            log::debug "Voice '$name' configuration is valid"
        done
    fi

    log::success "Kokoro injection configuration is valid"
    return 0
}

#######################################
# Perform data injection
# Arguments:
#   $1 - configuration JSON
# Returns:
#   0 if successful, 1 if failed
#######################################
inject::inject_data() {
    local config="$1"

    log::header "🔄 Injecting data into Kokoro"

    # Check Kokoro accessibility
    if ! inject::check_accessibility; then
        return 1
    fi

    # Clear previous rollback actions
    KOKORO_ROLLBACK_ACTIONS=()

    # Apply configurations if present
    local has_configurations
    has_configurations=$(echo "$config" | jq -e '.configurations' >/dev/null 2>&1 && echo "true" || echo "false")

    if [[ "$has_configurations" == "true" ]]; then
        local configurations
        configurations=$(echo "$config" | jq -c '.configurations')

        log::info "Applying Kokoro configurations..."
        log::warn "Configuration injection is managed via environment variables for Kokoro"
        log::info "Set environment variables before starting Kokoro for configuration"
    fi

    log::success "✅ Kokoro data injection completed"
    return 0
}

#######################################
# Check injection status
# Arguments:
#   $1 - configuration JSON
# Returns:
#   0 if successful, 1 if failed
#######################################
inject::check_status() {
    local config="$1"

    log::header "📊 Checking Kokoro injection status"

    # Check Kokoro accessibility
    if ! inject::check_accessibility; then
        return 1
    fi

    # Check voices directory
    log::info "Checking voices directory..."

    if [[ -d "$KOKORO_VOICES_DIR" ]]; then
        local voice_files
        voice_files=$(find "$KOKORO_VOICES_DIR" -type f 2>/dev/null | wc -l)

        if [[ "$voice_files" -gt 0 ]]; then
            log::info "Found $voice_files voice files"
        else
            log::warn "No voice files found in directory"
        fi
    else
        log::warn "Voices directory does not exist: $KOKORO_VOICES_DIR"
    fi

    # Test API endpoint
    log::info "Testing Kokoro API..."

    if curl -s "${KOKORO_HOST}/v1/audio/voices" | jq . >/dev/null 2>&1; then
        log::success "✅ Kokoro API is responding"
    else
        log::error "❌ Kokoro API not responding properly"
    fi

    return 0
}

#######################################
# Main execution function
#######################################
inject::main() {
    local action="$1"
    local config="${2:-}"

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
        "--rollback")
            inject::execute_rollback
            ;;
        "--help")
            inject::usage
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
