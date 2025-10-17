#!/usr/bin/env bash
################################################################################
# Whisper Resource CLI - v2.0 Universal Contract Compliant
# 
# OpenAI Whisper speech-to-text service
#
# Usage:
#   resource-whisper <command> [options]
#   resource-whisper <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    WHISPER_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    # Recalculate APP_ROOT from resolved symlink location
    APP_ROOT="$(builtin cd "${WHISPER_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
WHISPER_CLI_DIR="${var_RESOURCES_DIR}/whisper"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"

# Source agent management (load config and manager directly)
if [[ -f "${APP_ROOT}/resources/whisper/config/agents.conf" ]]; then
    source "${APP_ROOT}/resources/whisper/config/agents.conf"
    source "${APP_ROOT}/scripts/resources/agents/agent-manager.sh"
fi
# shellcheck disable=SC1091
source "${WHISPER_CLI_DIR}/config/defaults.sh"
# shellcheck disable=SC1091
source "${WHISPER_CLI_DIR}/config/messages.sh" 2>/dev/null || true

# Source Whisper libraries
for lib in common docker install status api agents; do
    lib_file="${WHISPER_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework in v2.0 mode (auto-creates manage/test/content groups)
cli::init "whisper" "OpenAI Whisper speech-to-text service" "v2"

# Override default handlers to point directly to whisper implementations
CLI_COMMAND_HANDLERS["manage::install"]="whisper::install"
CLI_COMMAND_HANDLERS["manage::uninstall"]="whisper::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="whisper::start"
CLI_COMMAND_HANDLERS["manage::stop"]="whisper::stop"
CLI_COMMAND_HANDLERS["manage::restart"]="whisper::restart"
CLI_COMMAND_HANDLERS["test::smoke"]="whisper::status"

# Override content handlers for Whisper-specific audio transcription functionality
CLI_COMMAND_HANDLERS["content::execute"]="whisper::transcribe_audio"

# Add Whisper-specific content subcommands not in the standard framework
cli::register_subcommand "content" "transcribe" "Transcribe audio file" "whisper::transcribe_audio"
cli::register_subcommand "content" "models" "List available models" "whisper::show_models"
cli::register_subcommand "content" "languages" "List supported languages" "whisper::show_languages"

# Additional information commands
cli::register_command "status" "Show detailed resource status" "whisper::status"
cli::register_command "logs" "Show Whisper logs" "whisper::show_logs"
# Create wrapper for agents command that delegates to manager
whisper::agents::command() {
    if type -t agent_manager::load_config &>/dev/null; then
        "${APP_ROOT}/scripts/resources/agents/agent-manager.sh" --config="whisper" "$@"
    else
        log::error "Agent management not available"
        return 1
    fi
}
export -f whisper::agents::command

cli::register_command "agents" "Manage running whisper agents" "whisper::agents::command"

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi