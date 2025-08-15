#!/usr/bin/env bash
################################################################################
# Whisper Resource CLI
# 
# Lightweight CLI interface for Whisper using the CLI Command Framework
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
source "${var_RESOURCES_COMMON_FILE:-${VROOLI_ROOT}/scripts/resources/common.sh}" 2>/dev/null || true

# Source the CLI Command Framework
# shellcheck disable=SC1091
source "${VROOLI_ROOT}/scripts/resources/lib/cli-command-framework.sh"

# Source Whisper configuration
# shellcheck disable=SC1091
source "${WHISPER_CLI_DIR}/config/defaults.sh" 2>/dev/null || true
defaults::export_config 2>/dev/null || true

# Set defaults if not already set
WHISPER_CONTAINER_NAME="${WHISPER_CONTAINER_NAME:-whisper}"
WHISPER_PORT="${WHISPER_PORT:-9000}"
WHISPER_HOST="${WHISPER_HOST:-localhost}"
WHISPER_DEFAULT_MODEL="${WHISPER_DEFAULT_MODEL:-base}"

# Initialize CLI framework
cli::init "whisper" "OpenAI Whisper speech-to-text service"

# Override help to provide Whisper-specific examples
cli::register_command "help" "Show this help message with Whisper examples" "resource_cli::show_help"

# Register additional Whisper-specific commands
cli::register_command "inject" "Transcribe audio file using Whisper" "resource_cli::inject" "modifies-system"
cli::register_command "transcribe" "Transcribe audio file" "resource_cli::transcribe" "modifies-system"
cli::register_command "models" "List available models" "resource_cli::models"
cli::register_command "languages" "List supported languages" "resource_cli::languages"
cli::register_command "credentials" "Show n8n credentials for Whisper" "resource_cli::credentials"
cli::register_command "uninstall" "Uninstall Whisper (requires --force)" "resource_cli::uninstall" "modifies-system"

################################################################################
# Resource-specific command implementations
################################################################################

# Transcribe audio file using Whisper (inject alias)
resource_cli::inject() {
    local file="${1:-}"
    
    if [[ -z "$file" ]]; then
        log::error "Audio file path required for transcription"
        echo "Usage: resource-whisper inject <audio-file>"
        echo ""
        echo "Examples:"
        echo "  resource-whisper inject recording.wav"
        echo "  resource-whisper inject shared:audio/meeting.mp3"
        return 1
    fi
    
    # Handle shared: prefix
    if [[ "$file" == shared:* ]]; then
        file="${VROOLI_ROOT}/${file#shared:}"
    fi
    
    if [[ ! -f "$file" ]]; then
        log::error "Audio file not found: $file"
        return 1
    fi
    
    # Delegate to manage.sh
    "${WHISPER_CLI_DIR}/manage.sh" --action transcribe --file "$file" "${@:2}"
}

# Transcribe audio file
resource_cli::transcribe() {
    resource_cli::inject "$@"
}

# Validate Whisper configuration
resource_cli::validate() {
    "${WHISPER_CLI_DIR}/manage.sh" --action status
}

# Show Whisper status
resource_cli::status() {
    "${WHISPER_CLI_DIR}/manage.sh" --action status
}

# Start Whisper
resource_cli::start() {
    "${WHISPER_CLI_DIR}/manage.sh" --action start
}

# Stop Whisper
resource_cli::stop() {
    "${WHISPER_CLI_DIR}/manage.sh" --action stop
}

# Install Whisper
resource_cli::install() {
    "${WHISPER_CLI_DIR}/manage.sh" --action install --yes yes
}

# Uninstall Whisper
resource_cli::uninstall() {
    FORCE="${FORCE:-false}"
    
    if [[ "$FORCE" != "true" ]]; then
        echo "⚠️  This will remove Whisper and all its data. Use --force to confirm."
        return 1
    fi
    
    "${WHISPER_CLI_DIR}/manage.sh" --action uninstall --yes yes
}

# Get credentials for n8n integration
resource_cli::credentials() {
    source "${VROOLI_ROOT}/scripts/resources/lib/credentials-utils.sh"
    
    if ! credentials::parse_args "$@"; then
        [[ $? -eq 2 ]] && { credentials::show_help "whisper"; return 0; }
        return 1
    fi
    
    local status
    status=$(credentials::get_resource_status "$WHISPER_CONTAINER_NAME")
    
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
    
    local response
    response=$(credentials::build_response "whisper" "$status" "$connections_array")
    credentials::format_output "$response"
}

# List available models
resource_cli::models() {
    echo "Available Whisper models:"
    echo ""
    echo "  tiny.en     English-only,  ~39 MB,  ~32x realtime"
    echo "  tiny        Multilingual,  ~39 MB,  ~32x realtime"
    echo "  base.en     English-only,  ~74 MB,  ~16x realtime"
    echo "  base        Multilingual,  ~74 MB,  ~16x realtime"
    echo "  small.en    English-only, ~244 MB,   ~6x realtime"
    echo "  small       Multilingual, ~244 MB,   ~6x realtime"
    echo "  medium.en   English-only, ~769 MB,   ~2x realtime"
    echo "  medium      Multilingual, ~769 MB,   ~2x realtime"
    echo "  large-v1    Multilingual, ~1550 MB,  ~1x realtime"
    echo "  large-v2    Multilingual, ~1550 MB,  ~1x realtime"
    echo "  large-v3    Multilingual, ~1550 MB,  ~1x realtime"
    echo ""
    echo "Current default: $WHISPER_DEFAULT_MODEL"
}

# List supported languages
resource_cli::languages() {
    echo "Supported languages (99 languages):"
    echo ""
    echo "Afrikaans, Albanian, Amharic, Arabic, Armenian, Assamese, Azerbaijani,"
    echo "Bashkir, Basque, Belarusian, Bengali, Bosnian, Breton, Bulgarian,"
    echo "Burmese, Castilian, Catalan, Chinese, Croatian, Czech, Danish, Dutch,"
    echo "English, Estonian, Faroese, Finnish, Flemish, French, Galician,"
    echo "Georgian, German, Greek, Gujarati, Haitian, Haitian Creole, Hausa,"
    echo "Hawaiian, Hebrew, Hindi, Hungarian, Icelandic, Indonesian, Italian,"
    echo "Japanese, Javanese, Kannada, Kazakh, Khmer, Korean, Lao, Latin,"
    echo "Latvian, Lingala, Lithuanian, Luxembourgish, Macedonian, Malagasy,"
    echo "Malay, Malayalam, Maltese, Maori, Marathi, Mongolian, Myanmar,"
    echo "Nepali, Norwegian, Nynorsk, Occitan, Pashto, Persian, Polish,"
    echo "Portuguese, Punjabi, Romanian, Russian, Sanskrit, Serbian, Shona,"
    echo "Sindhi, Sinhala, Slovak, Slovenian, Somali, Spanish, Sundanese,"
    echo "Swahili, Swedish, Tagalog, Tajik, Tamil, Tatar, Telugu, Thai,"
    echo "Tibetan, Turkish, Turkmen, Ukrainian, Urdu, Uzbek, Vietnamese,"
    echo "Welsh, Yiddish, Yoruba"
    echo ""
    echo "Note: Language detection is automatic, or specify with --language parameter"
}

# Custom help function with Whisper-specific examples
resource_cli::show_help() {
    # Show standard framework help first
    cli::_handle_help
    
    # Add Whisper-specific examples
    echo ""
    echo "🎤 OpenAI Whisper Speech-to-Text Examples:"
    echo ""
    echo "Audio Transcription:"
    echo "  resource-whisper transcribe recording.wav               # Transcribe audio file"
    echo "  resource-whisper inject meeting.mp3                    # Same as transcribe"
    echo "  resource-whisper inject shared:audio/interview.m4a     # Transcribe shared file"
    echo ""
    echo "Model & Language Info:"
    echo "  resource-whisper models                                # List available models"
    echo "  resource-whisper languages                             # List supported languages"
    echo ""
    echo "Management:"
    echo "  resource-whisper status                                # Check service status"
    echo "  resource-whisper credentials                           # Get API details"
    echo ""
    echo "Speech Recognition Features:"
    echo "  • Automatic language detection"
    echo "  • 99 language support including multilingual"
    echo "  • Multiple model sizes for speed/accuracy tradeoffs"
    echo "  • Support for common audio formats (wav, mp3, m4a, ogg, flac)"
    echo ""
    echo "Default Port: $WHISPER_PORT"
    echo "Default Model: $WHISPER_DEFAULT_MODEL"
    echo "API Endpoint: http://localhost:$WHISPER_PORT/asr"
}

################################################################################
# Main execution - dispatch to framework
################################################################################

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi