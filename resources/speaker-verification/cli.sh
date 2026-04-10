#!/usr/bin/env bash
################################################################################
# Speaker Verification Resource CLI - v2.0 Universal Contract Compliant
#
# Local speaker verification using NVIDIA NeMo TitaNet embeddings
#
# Usage:
#   resource-speaker-verification <command> [options]
#   resource-speaker-verification <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    SV_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    APP_ROOT="$(builtin cd "${SV_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
SV_CLI_DIR="${APP_ROOT}/resources/speaker-verification"

# Source standard variables
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"

# Source v2.0 CLI Command Framework
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"

# Agent management (conditional)
if [[ -f "${SV_CLI_DIR}/config/agents.conf" ]]; then
    # shellcheck disable=SC1091
    source "${SV_CLI_DIR}/config/agents.conf"
    if [[ -f "${APP_ROOT}/scripts/resources/agents/agent-manager.sh" ]]; then
        # shellcheck disable=SC1091
        source "${APP_ROOT}/scripts/resources/agents/agent-manager.sh"
    fi
fi

# Source configuration
# shellcheck disable=SC1091
source "${SV_CLI_DIR}/config/defaults.sh"
speaker_verification::export_config 2>/dev/null || true
# shellcheck disable=SC1091
source "${SV_CLI_DIR}/config/messages.sh" 2>/dev/null || true
messages::export_messages 2>/dev/null || true

# Source all lib files
for lib in common core docker install status api profiles health test; do
    lib_file="${SV_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" || {
            log::warn "Failed to load library: $lib"
        }
    fi
done

# Initialize CLI framework in v2.0 mode
cli::init "speaker-verification" "Speaker verification using NeMo TitaNet" "v2"
# Subcommands under framework-managed groups: content, manage, test

# ==============================================================================
# REQUIRED HANDLERS - Universal Contract v2.0 compliance
# ==============================================================================
CLI_COMMAND_HANDLERS["manage::install"]="speaker_verification::install"
CLI_COMMAND_HANDLERS["manage::uninstall"]="speaker_verification::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="speaker_verification::start"
CLI_COMMAND_HANDLERS["manage::stop"]="speaker_verification::stop"
CLI_COMMAND_HANDLERS["manage::restart"]="speaker_verification::restart"
CLI_COMMAND_HANDLERS["test::smoke"]="speaker_verification::test::smoke"
CLI_COMMAND_HANDLERS["test::integration"]="speaker_verification::test::integration"
CLI_COMMAND_HANDLERS["test::unit"]="speaker_verification::test::unit"
CLI_COMMAND_HANDLERS["test::all"]="speaker_verification::test::all"

# ==============================================================================
# REQUIRED INFORMATION COMMANDS
# ==============================================================================
cli::register_command "status" "Show detailed resource status" "speaker_verification::status"
cli::register_command "logs" "Show speaker verification logs" "speaker_verification::show_logs"

# ==============================================================================
# DOMAIN-SPECIFIC CONTENT SUBCOMMANDS
# ==============================================================================
cli::register_subcommand "content" "enroll" "Enroll a speaker profile from audio" "speaker_verification::content::enroll"
cli::register_subcommand "content" "verify" "Verify speaker against a stored profile" "speaker_verification::content::verify"
cli::register_subcommand "content" "profiles" "Manage speaker profiles (list/get/remove)" "speaker_verification::content::profiles"
cli::register_subcommand "content" "info" "Show backend and model information" "speaker_verification::content::info"

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi
