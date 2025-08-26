#!/usr/bin/env bash
################################################################################
# CrewAI Resource CLI - v2.0 Universal Contract Compliant
# 
# Multi-agent AI framework for building collaborative AI systems
#
# Usage:
#   resource-crewai <command> [options]
#   resource-crewai <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    CREWAI_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    APP_ROOT="$(builtin cd "${CREWAI_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
CREWAI_CLI_DIR="${APP_ROOT}/resources/crewai"

# Source standard variables
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"
# shellcheck disable=SC1091
source "${CREWAI_CLI_DIR}/config/defaults.sh"

# Source CrewAI libraries
for lib in core install status inject content manage; do
    lib_file="${CREWAI_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework in v2.0 mode (auto-creates manage/test/content groups)
cli::init "crewai" "CrewAI multi-agent AI framework" "v2"

# ==============================================================================
# REQUIRED HANDLERS - These MUST be mapped for v2.0 compliance
# ==============================================================================
CLI_COMMAND_HANDLERS["manage::install"]="install_crewai"
CLI_COMMAND_HANDLERS["manage::uninstall"]="crewai::install::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="start_crewai"
CLI_COMMAND_HANDLERS["manage::stop"]="stop_crewai"
CLI_COMMAND_HANDLERS["manage::restart"]="crewai::manage::restart"
CLI_COMMAND_HANDLERS["test::smoke"]="crewai::status::check"

# Content handlers - CrewAI specific content management
CLI_COMMAND_HANDLERS["content::add"]="crewai::content::add"
CLI_COMMAND_HANDLERS["content::list"]="crewai::content::list"
CLI_COMMAND_HANDLERS["content::get"]="crewai::content::get"
CLI_COMMAND_HANDLERS["content::remove"]="crewai::content::remove"
CLI_COMMAND_HANDLERS["content::execute"]="crewai::content::execute"

# ==============================================================================
# REQUIRED INFORMATION COMMANDS
# ==============================================================================
cli::register_command "status" "Show detailed CrewAI status" "crewai::status"
cli::register_command "logs" "Show CrewAI logs" "crewai::logs"

# ==============================================================================
# CUSTOM CrewAI COMMANDS
# ==============================================================================
# Custom content subcommands for CrewAI-specific operations
cli::register_subcommand "content" "inject" "Inject crews/agents from directory" "crewai::content::inject"
cli::register_subcommand "content" "crews" "List available crews" "list_crews"
cli::register_subcommand "content" "agents" "List available agents" "list_agents"

# Only execute if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi