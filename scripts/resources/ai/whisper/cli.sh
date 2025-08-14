#!/usr/bin/env bash
################################################################################
# Whisper Resource CLI
# 
# Lightweight CLI interface for Whisper that delegates to existing lib functions.
#
# Usage:
#   resource-whisper <command> [options]
#
################################################################################

set -euo pipefail

# Get script directory
WHISPER_CLI_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VROOLI_ROOT="${VROOLI_ROOT:-$(cd "$WHISPER_CLI_DIR/../../../.." && pwd)}"
export VROOLI_ROOT
export RESOURCE_DIR="$WHISPER_CLI_DIR"

# Source utilities first
# shellcheck disable=SC1091
source "${VROOLI_ROOT}/scripts/lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LOG_FILE:-${VROOLI_ROOT}/scripts/lib/utils/log.sh}" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}" 2>/dev/null || true

# Source the CLI template
# shellcheck disable=SC1091
source "${VROOLI_ROOT}/scripts/lib/resources/cli/resource-cli-template.sh"

# Source Whisper configuration
# shellcheck disable=SC1091
source "${WHISPER_CLI_DIR}/config/defaults.sh" 2>/dev/null || true
defaults::export_config 2>/dev/null || true

# Initialize with resource name
resource_cli::init "whisper"

################################################################################
# Delegate to existing Whisper functions via manage.sh
################################################################################

# Get credentials for n8n integration
resource_cli::credentials() {
    # Source credentials utilities
    # shellcheck disable=SC1091
    source "${VROOLI_ROOT}/scripts/resources/lib/credentials-utils.sh"
    
    # Parse arguments
    if ! credentials::parse_args "$@"; then
        [[ $? -eq 2 ]] && { credentials::show_help "whisper"; return 0; }
        return 1
    fi
    
    # Get resource status
    local status
    status=$(credentials::get_resource_status "$WHISPER_CONTAINER_NAME")
    
    # Build connections array - Whisper runs as open HTTP API (no auth)
    local connections_array="[]"
    if [[ "$status" == "running" ]]; then
        # Whisper HTTP API connection (no authentication required)
        local connection_obj
        connection_obj=$(jq -n \
            --arg host "localhost" \
            --argjson port "$WHISPER_PORT" \
            --arg path "/asr" \
            --argjson ssl false \
            '{
                host: $host,
                port: $port,
                path: $path,
                ssl: $ssl
            }')
        
        local metadata_obj
        metadata_obj=$(jq -n \
            --arg description "OpenAI Whisper speech-to-text service" \
            --arg model "$WHISPER_DEFAULT_MODEL" \
            --argjson supported_formats '["wav", "mp3", "m4a", "ogg", "flac"]' \
            '{
                description: $description,
                default_model: $model,
                supported_formats: $supported_formats
            }')
        
        local connection
        connection=$(credentials::build_connection \
            "main" \
            "Whisper API" \
            "httpRequest" \
            "$connection_obj" \
            "{}" \
            "$metadata_obj")
        
        connections_array="[$connection]"
    fi
    
    # Build and validate response
    local response
    response=$(credentials::build_response "whisper" "$status" "$connections_array")
    
    if credentials::validate_json "$response"; then
        credentials::format_output "$response"
    else
        log::error "Invalid credentials JSON generated"
        return 1
    fi
}

# Delegate other commands to manage.sh
resource_cli::inject() {
    "${WHISPER_CLI_DIR}/manage.sh" --action transcribe --file "${1:-}" "${@:2}"
}

resource_cli::validate() {
    "${WHISPER_CLI_DIR}/manage.sh" --action status
}

resource_cli::status() {
    "${WHISPER_CLI_DIR}/manage.sh" --action status
}

resource_cli::start() {
    "${WHISPER_CLI_DIR}/manage.sh" --action start
}

resource_cli::stop() {
    "${WHISPER_CLI_DIR}/manage.sh" --action stop
}

resource_cli::install() {
    "${WHISPER_CLI_DIR}/manage.sh" --action install --yes yes
}

resource_cli::uninstall() {
    FORCE="${FORCE:-false}"
    if [[ "$FORCE" != "true" ]]; then
        echo "⚠️  This will remove Whisper and all its data. Use --force to confirm."
        return 1
    fi
    "${WHISPER_CLI_DIR}/manage.sh" --action uninstall --yes yes
}

# Show help
resource_cli::show_help() {
    cat << EOF
🚀 Whisper Resource CLI

USAGE:
    resource-whisper <command> [options]

CORE COMMANDS:
    inject <file>       Transcribe audio file using Whisper
    validate            Validate Whisper configuration
    status              Show Whisper status
    start               Start Whisper container
    stop                Stop Whisper container
    install             Install Whisper
    uninstall           Uninstall Whisper (requires --force)
    credentials         Get connection credentials for n8n integration

OPTIONS:
    --verbose, -v       Show detailed output
    --dry-run           Preview actions without executing
    --force             Force operation (skip confirmations)

EXAMPLES:
    resource-whisper status
    resource-whisper inject audio.wav
    resource-whisper credentials --format pretty

For more information: https://docs.vrooli.com/resources/whisper
EOF
}

# Main command router
resource_cli::main() {
    # Parse common options first
    local remaining_args
    remaining_args=$(resource_cli::parse_options "$@")
    set -- $remaining_args
    
    local command="${1:-help}"
    shift || true
    
    case "$command" in
        # Standard resource commands
        inject|validate|status|start|stop|install|uninstall|credentials)
            resource_cli::$command "$@"
            ;;
            
        help|--help|-h)
            resource_cli::show_help
            ;;
        *)
            log::error "Unknown command: $command"
            echo ""
            resource_cli::show_help
            exit 1
            ;;
    esac
}

# Run main if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    resource_cli::main "$@"
fi