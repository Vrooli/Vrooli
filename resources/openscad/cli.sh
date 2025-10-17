#!/usr/bin/env bash
################################################################################
# OpenSCAD Resource CLI - v2.0 Universal Contract Compliant
# 
# Programmatic 3D CAD modeler using script-based solid modeling
#
# Usage:
#   resource-openscad <command> [options]
#   resource-openscad <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    OPENSCAD_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    # Recalculate APP_ROOT from resolved symlink location
    APP_ROOT="$(builtin cd "${OPENSCAD_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
OPENSCAD_CLI_DIR="${APP_ROOT}/resources/openscad"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"
# shellcheck disable=SC1091
source "${OPENSCAD_CLI_DIR}/config/defaults.sh"

# Source OpenSCAD libraries
for lib in common docker install status lifecycle content test; do
    lib_file="${OPENSCAD_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework in v2.0 mode (auto-creates manage/test/content groups)
cli::init "openscad" "OpenSCAD 3D CAD modeler with script-based solid modeling" "v2"

# ==============================================================================
# REQUIRED HANDLERS - These MUST be mapped for v2.0 compliance
# ==============================================================================
CLI_COMMAND_HANDLERS["manage::install"]="openscad::install"
CLI_COMMAND_HANDLERS["manage::uninstall"]="openscad::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="openscad::docker::start"
CLI_COMMAND_HANDLERS["manage::stop"]="openscad::docker::stop"
CLI_COMMAND_HANDLERS["manage::restart"]="openscad::docker::restart"
CLI_COMMAND_HANDLERS["test::smoke"]="openscad::test::smoke"
CLI_COMMAND_HANDLERS["test::integration"]="openscad::test::integration"
CLI_COMMAND_HANDLERS["test::unit"]="openscad::test::unit"
CLI_COMMAND_HANDLERS["test::all"]="openscad::test::all"

# Content handlers for OpenSCAD 3D modeling functionality
CLI_COMMAND_HANDLERS["content::add"]="openscad::content::add"
CLI_COMMAND_HANDLERS["content::list"]="openscad::content::list"
CLI_COMMAND_HANDLERS["content::get"]="openscad::content::get"
CLI_COMMAND_HANDLERS["content::remove"]="openscad::content::remove"
CLI_COMMAND_HANDLERS["content::execute"]="openscad::content::execute"

# ==============================================================================
# REQUIRED INFORMATION COMMANDS
# ==============================================================================
cli::register_command "status" "Show detailed OpenSCAD resource status" "openscad::status"
cli::register_command "logs" "Show OpenSCAD container logs" "openscad::docker::logs"

# ==============================================================================
# OPENSCAD-SPECIFIC CONTENT SUBCOMMANDS
# ==============================================================================
# Add OpenSCAD-specific content operations for 3D modeling
cli::register_subcommand "content" "render" "Render script to output format (alias for execute)" "openscad::content::execute"
cli::register_subcommand "content" "clear" "Clear all OpenSCAD data (scripts and outputs)" "openscad::clear_data"

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi