#!/usr/bin/env bash
################################################################################
# Whisper Resource CLI
# 
# Ultra-thin CLI wrapper that delegates directly to library functions
#
# Usage:
#   resource-whisper <command> [options]
#
################################################################################

set -euo pipefail

# Get script directory (resolving symlinks for installed CLI)
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    # If this is a symlink, resolve it
    WHISPER_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
else
    WHISPER_CLI_SCRIPT="${BASH_SOURCE[0]}"
fi
WHISPER_CLI_DIR="$(cd "$(dirname "$WHISPER_CLI_SCRIPT")" && pwd)"

# Source standard variables
# shellcheck disable=SC1091
source "${WHISPER_CLI_DIR}/../../../lib/utils/var.sh"

# Source utilities using var_ variables
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"

# Source the CLI Command Framework
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/cli-command-framework.sh"

# Source Whisper configuration
# shellcheck disable=SC1091
source "${WHISPER_CLI_DIR}/config/defaults.sh" 2>/dev/null || true
defaults::export_config 2>/dev/null || true

# Source Whisper libraries - these contain the actual functionality
# shellcheck disable=SC1091
source "${WHISPER_CLI_DIR}/lib/common.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${WHISPER_CLI_DIR}/lib/docker.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${WHISPER_CLI_DIR}/lib/install.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${WHISPER_CLI_DIR}/lib/status.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${WHISPER_CLI_DIR}/lib/api.sh" 2>/dev/null || true

# Initialize CLI framework
cli::init "whisper" "OpenAI Whisper speech-to-text service"

# Override help to provide Whisper-specific examples
cli::register_command "help" "Show this help message with examples" "whisper::show_help"

# Register core commands - direct library function calls
cli::register_command "install" "Install Whisper" "whisper::install" "modifies-system"
cli::register_command "uninstall" "Uninstall Whisper" "whisper::uninstall" "modifies-system"
cli::register_command "start" "Start Whisper" "whisper::start" "modifies-system"
cli::register_command "stop" "Stop Whisper" "whisper::stop" "modifies-system"
cli::register_command "restart" "Restart Whisper" "whisper::restart" "modifies-system"

# Register status and monitoring commands
cli::register_command "status" "Show service status" "status::show_status"
cli::register_command "quick-status" "Show quick status" "status::quick_status"
cli::register_command "logs" "Show container logs" "whisper::show_logs"
cli::register_command "stats" "Show resource usage" "whisper::get_stats"

# Register transcription commands
cli::register_command "transcribe" "Transcribe audio file" "whisper::cli_transcribe"
cli::register_command "inject" "Transcribe audio file (alias)" "whisper::cli_transcribe"

# Register API and testing commands
cli::register_command "test-api" "Test API connectivity" "whisper::test_api"
cli::register_command "api-info" "Show API information" "whisper::get_api_info"

# Register model and language commands
cli::register_command "models" "List available models" "whisper::show_models"
cli::register_command "languages" "List supported languages" "whisper::show_languages"

# Register utility commands
cli::register_command "credentials" "Get n8n credentials" "whisper::show_credentials"

################################################################################
# CLI wrapper functions - minimal wrappers for commands that need argument handling
################################################################################

# Transcribe audio file with argument handling
whisper::cli_transcribe() {
    local file="${1:-}"
    local task="${2:-transcribe}"
    local language="${3:-auto}"
    
    if [[ -z "$file" ]]; then
        log::error "Audio file path required for transcription"
        echo "Usage: resource-whisper transcribe <audio-file> [task] [language]"
        echo ""
        echo "Examples:"
        echo "  resource-whisper transcribe recording.wav"
        echo "  resource-whisper transcribe meeting.mp3 transcribe es"
        echo "  resource-whisper transcribe shared:audio/interview.m4a translate"
        return 1
    fi
    
    # Handle shared: prefix
    if [[ "$file" == shared:* ]]; then
        file="${var_ROOT_DIR}/${file#shared:}"
    fi
    
    if [[ ! -f "$file" ]]; then
        log::error "Audio file not found: $file"
        return 1
    fi
    
    # Call library function
    whisper::transcribe_audio "$file" "$task" "$language"
}

# Show credentials for n8n integration
whisper::show_credentials() {
    # Source credentials utilities
    # shellcheck disable=SC1091
    source "${var_SCRIPTS_RESOURCES_LIB_DIR}/credentials-utils.sh"
    
    credentials::parse_args "$@" || return $?
    
    local status
    status=$(credentials::get_resource_status "${WHISPER_CONTAINER_NAME:-whisper}")
    
    local connections_array="[]"
    if [[ "$status" == "running" ]]; then
        local connection_obj
        connection_obj=$(jq -n \
            --arg host "${WHISPER_HOST:-localhost}" \
            --argjson port "${WHISPER_PORT:-9000}" \
            --arg path "/asr" \
            --argjson ssl false \
            '{host: $host, port: $port, path: $path, ssl: $ssl}')
        
        local metadata_obj
        metadata_obj=$(jq -n \
            --arg description "OpenAI Whisper speech-to-text service" \
            --arg model "${WHISPER_DEFAULT_MODEL:-base}" \
            --argjson supported_formats '["wav", "mp3", "m4a", "ogg", "flac"]' \
            '{description: $description, default_model: $model, supported_formats: $supported_formats}')
        
        local connection
        connection=$(credentials::build_connection \
            "main" "Whisper API" "httpRequest" \
            "$connection_obj" "{}" "$metadata_obj")
        
        connections_array="[$connection]"
    fi
    
    credentials::format_output "$(credentials::build_response "whisper" "$status" "$connections_array")"
}

# Custom help function with examples
whisper::show_help() {
    cli::_handle_help
    
    echo ""
    echo "🎤 Examples:"
    echo ""
    echo "  # Audio transcription"
    echo "  resource-whisper transcribe recording.wav"
    echo "  resource-whisper transcribe meeting.mp3 transcribe es"
    echo "  resource-whisper transcribe audio.wav translate"
    echo ""
    echo "  # Service management"
    echo "  resource-whisper status"
    echo "  resource-whisper start"
    echo "  resource-whisper logs"
    echo ""
    echo "  # Information"
    echo "  resource-whisper models"
    echo "  resource-whisper languages"
    echo "  resource-whisper api-info"
    echo ""
    echo "Supported formats: wav, mp3, m4a, ogg, flac, aac, wma"
    echo "Maximum file size: 25MB"
    echo "Default Port: ${WHISPER_PORT:-9000}"
    echo "Default Model: ${WHISPER_DEFAULT_MODEL:-base}"
    echo "API Endpoint: ${WHISPER_BASE_URL:-http://localhost:9000}/asr"
}

################################################################################
# Main execution
################################################################################

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi