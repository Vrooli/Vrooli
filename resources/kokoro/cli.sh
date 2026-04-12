#!/usr/bin/env bash
################################################################################
# Kokoro Resource CLI - v2.0 Universal Contract Compliant
#
# Kokoro text-to-speech synthesis service
#
# Usage:
#   resource-kokoro <command> [options]
#   resource-kokoro <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    KOKORO_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    # Recalculate APP_ROOT from resolved symlink location
    APP_ROOT="$(builtin cd "${KOKORO_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
KOKORO_CLI_DIR="${var_RESOURCES_DIR}/kokoro"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"

# Source agent management (load config and manager directly)
if [[ -f "${APP_ROOT}/resources/kokoro/config/agents.conf" ]]; then
    source "${APP_ROOT}/resources/kokoro/config/agents.conf"
    source "${APP_ROOT}/scripts/resources/agents/agent-manager.sh"
fi
# shellcheck disable=SC1091
source "${KOKORO_CLI_DIR}/config/defaults.sh"
# shellcheck disable=SC1091
source "${KOKORO_CLI_DIR}/config/messages.sh" 2>/dev/null || true

# Source Kokoro libraries
for lib in common docker install status api agents; do
    lib_file="${KOKORO_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework in v2.0 mode (auto-creates manage/test/content groups)
cli::init "kokoro" "Kokoro text-to-speech synthesis service" "v2"

kokoro::native::resource() {
    local operation="$1"
    shift || true
    vrooli resource kokoro "$operation" "$@"
}

kokoro::native::install() { kokoro::native::resource install "$@"; }
kokoro::native::uninstall() { kokoro::native::resource uninstall "$@"; }
kokoro::native::start() { kokoro::native::resource start "$@"; }
kokoro::native::stop() { kokoro::native::resource stop "$@"; }
kokoro::native::restart() { kokoro::native::resource restart "$@"; }
kokoro::native::status() { kokoro::native::resource status "$@"; }
kokoro::native::logs() { kokoro::native::resource logs "$@"; }

# Standard lifecycle/status/logs now route through the native control plane.
CLI_COMMAND_HANDLERS["manage::install"]="kokoro::native::install"
CLI_COMMAND_HANDLERS["manage::uninstall"]="kokoro::native::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="kokoro::native::start"
CLI_COMMAND_HANDLERS["manage::stop"]="kokoro::native::stop"
CLI_COMMAND_HANDLERS["manage::restart"]="kokoro::native::restart"
CLI_COMMAND_HANDLERS["test::smoke"]="kokoro::native::status"

# Override content handlers for Kokoro-specific text-to-speech functionality
CLI_COMMAND_HANDLERS["content::execute"]="kokoro::synthesize_text"

# Add Kokoro-specific content subcommands not in the standard framework
cli::register_subcommand "content" "synthesize" "Synthesize text to speech" "kokoro::synthesize_text"
cli::register_subcommand "content" "voices" "List available voices" "kokoro::list_voices"

# Additional information commands
cli::register_command "status" "Show detailed resource status" "kokoro::native::status"
cli::register_command "logs" "Show Kokoro logs" "kokoro::native::logs"
# Create wrapper for agents command that delegates to manager
kokoro::agents::command() {
    if type -t agent_manager::load_config &>/dev/null; then
        "${APP_ROOT}/scripts/resources/agents/agent-manager.sh" --config="kokoro" "$@"
    else
        log::error "Agent management not available"
        return 1
    fi
}
export -f kokoro::agents::command

cli::register_command "agents" "Manage running kokoro agents" "kokoro::agents::command"

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi
