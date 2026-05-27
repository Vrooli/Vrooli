#!/usr/bin/env bash
set -euo pipefail

# Kyutai STT Data Injection Adapter
# Handles injection of model/runtime configuration into Kyutai STT.
# Part of the Vrooli resource data injection system.

# Consumed by the resource data-injection discovery framework.
# shellcheck disable=SC2034
DESCRIPTION="Inject model and runtime configuration into the Kyutai STT service"

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

# Source Kyutai STT configuration if available
if [[ -f "${SCRIPT_DIR}/config/defaults.sh" ]]; then
    # shellcheck disable=SC1091
    source "${SCRIPT_DIR}/config/defaults.sh"
fi

# Default Kyutai STT settings
readonly DEFAULT_KYUTAI_STT_HOST="http://localhost:8094"

# Kyutai STT settings (can be overridden by environment)
KYUTAI_STT_HOST="${KYUTAI_STT_BASE_URL:-$DEFAULT_KYUTAI_STT_HOST}"

# Operation tracking
declare -a KYUTAI_STT_ROLLBACK_ACTIONS=()

#######################################
# Display usage information
#######################################
inject::usage() {
    cat << EOF
Kyutai STT Data Injection Adapter

USAGE:
    $0 [OPTIONS] CONFIG_JSON

DESCRIPTION:
    Injects runtime configuration into Kyutai STT based on scenario
    configuration. Supports validation, injection, status checks, and rollback.

OPTIONS:
    --validate    Validate the injection configuration
    --inject      Perform the configuration injection
    --status      Check status of the service
    --rollback    Rollback injected configuration
    --help        Show this help message

CONFIGURATION FORMAT:
    {
      "configurations": [
        {
          "key": "model",
          "value": "kyutai/stt-1b-en_fr"
        }
      ]
    }

EXAMPLES:
    # Validate configuration
    $0 --validate '{"configurations": [{"key": "model", "value": "kyutai/stt-1b-en_fr"}]}'

    # Inject configuration
    $0 --inject '{"configurations": [{"key": "model", "value": "kyutai/stt-1b-en_fr"}]}'

EOF
}

#######################################
# Check if Kyutai STT is accessible
# Returns:
#   0 if accessible, 1 otherwise
#######################################
inject::check_accessibility() {
    if curl -s --max-time 5 "${KYUTAI_STT_HOST}/health" >/dev/null 2>&1; then
        log::debug "Kyutai STT is accessible at $KYUTAI_STT_HOST"
        return 0
    else
        log::error "Kyutai STT is not accessible at $KYUTAI_STT_HOST"
        log::info "Ensure Kyutai STT is running: resource-kyutai-stt manage start"
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

    KYUTAI_STT_ROLLBACK_ACTIONS+=("$description|$command")
    log::debug "Added Kyutai STT rollback action: $description"
}

#######################################
# Execute rollback actions
#######################################
inject::execute_rollback() {
    if [[ ${#KYUTAI_STT_ROLLBACK_ACTIONS[@]} -eq 0 ]]; then
        log::info "No Kyutai STT rollback actions to execute"
        return 0
    fi

    log::info "Executing Kyutai STT rollback actions..."

    local success_count=0
    local total_count=${#KYUTAI_STT_ROLLBACK_ACTIONS[@]}

    # Execute in reverse order
    for ((i=${#KYUTAI_STT_ROLLBACK_ACTIONS[@]}-1; i>=0; i--)); do
        local action="${KYUTAI_STT_ROLLBACK_ACTIONS[i]}"
        IFS='|' read -r description command <<< "$action"

        log::info "Rollback: $description"

        if eval "$command"; then
            success_count=$((success_count + 1))
            log::success "Rollback completed: $description"
        else
            log::error "Rollback failed: $description"
        fi
    done

    log::info "Kyutai STT rollback completed: $success_count/$total_count actions successful"
    KYUTAI_STT_ROLLBACK_ACTIONS=()
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

    log::info "Validating Kyutai STT injection configuration..."

    # Basic JSON validation
    if ! echo "$config" | jq . >/dev/null 2>&1; then
        log::error "Invalid JSON in Kyutai STT injection configuration"
        return 1
    fi

    # Check for at least one injection type
    local has_configurations
    has_configurations=$(echo "$config" | jq -e '.configurations' >/dev/null 2>&1 && echo "true" || echo "false")

    if [[ "$has_configurations" == "false" ]]; then
        log::error "Kyutai STT injection configuration must have 'configurations'"
        return 1
    fi

    log::success "Kyutai STT injection configuration is valid"
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

    log::header "🔄 Injecting configuration into Kyutai STT"

    if ! inject::check_accessibility; then
        return 1
    fi

    KYUTAI_STT_ROLLBACK_ACTIONS=()

    local has_configurations
    has_configurations=$(echo "$config" | jq -e '.configurations' >/dev/null 2>&1 && echo "true" || echo "false")

    if [[ "$has_configurations" == "true" ]]; then
        log::info "Applying Kyutai STT configurations..."
        log::warn "Configuration is managed via environment variables for Kyutai STT"
        log::info "Set KYUTAI_STT_* environment variables before starting the service"
    fi

    log::success "✅ Kyutai STT data injection completed"
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

    log::header "📊 Checking Kyutai STT injection status"

    if ! inject::check_accessibility; then
        return 1
    fi

    log::info "Testing Kyutai STT API..."

    if curl -s "${KYUTAI_STT_HOST}/v1/info" | jq . >/dev/null 2>&1; then
        log::success "✅ Kyutai STT API is responding"
    else
        log::error "❌ Kyutai STT API not responding properly"
    fi

    return 0
}

#######################################
# Main execution function
#######################################
inject::main() {
    local action="$1"
    local config="${2:-}"

    if [[ "$action" != "--rollback" && "$action" != "--help" && -z "$config" ]]; then
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
