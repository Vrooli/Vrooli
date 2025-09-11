#!/usr/bin/env bash
################################################################################
# Blender Resource CLI - v2.0 Universal Contract Compliant
# 
# Professional 3D creation suite with Python API for Vrooli automation scenarios
#
# Usage:
#   resource-blender <command> [options]
#   resource-blender <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    BLENDER_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    # Recalculate APP_ROOT from resolved symlink location
    APP_ROOT="$(builtin cd "${BLENDER_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
BLENDER_CLI_DIR="${APP_ROOT}/resources/blender"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"
# shellcheck disable=SC1091
source "${BLENDER_CLI_DIR}/config/defaults.sh"

# Source Blender libraries
for lib in core docker status inject test; do
    lib_file="${BLENDER_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework in v2.0 mode (auto-creates manage/test/content groups)
cli::init "blender" "Blender 3D creation suite management" "v2"

# ==============================================================================
# REQUIRED HANDLERS - These MUST be mapped for v2.0 compliance
# ==============================================================================
CLI_COMMAND_HANDLERS["manage::install"]="blender::install"
CLI_COMMAND_HANDLERS["manage::uninstall"]="blender::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="blender::start"  
CLI_COMMAND_HANDLERS["manage::stop"]="blender::stop"
CLI_COMMAND_HANDLERS["manage::restart"]="blender::docker::restart"
CLI_COMMAND_HANDLERS["test::smoke"]="blender::health_check"
CLI_COMMAND_HANDLERS["test::integration"]="blender::test"
CLI_COMMAND_HANDLERS["test::unit"]="blender::test"
CLI_COMMAND_HANDLERS["test::all"]="blender::run_tests"

# Content handlers - Blender-specific 3D creation and script management functionality
CLI_COMMAND_HANDLERS["content::add"]="blender::inject"
CLI_COMMAND_HANDLERS["content::list"]="blender::list_injected"
CLI_COMMAND_HANDLERS["content::get"]="blender::export"
CLI_COMMAND_HANDLERS["content::remove"]="blender::remove_injected"
CLI_COMMAND_HANDLERS["content::execute"]="blender::run_script"

# ==============================================================================
# REQUIRED INFORMATION COMMANDS
# ==============================================================================
cli::register_command "status" "Show detailed resource status" "blender::status"
cli::register_command "logs" "Show Blender logs" "blender::docker::logs"

# ==============================================================================
# BLENDER-SPECIFIC CONTENT COMMANDS
# ==============================================================================
# Export operations for Blender outputs
cli::register_subcommand "content" "export" "Export specific output file" "blender::export"
cli::register_subcommand "content" "export-all" "Export all output files" "blender::export_all"

# ==============================================================================
# BLENDER-SPECIFIC TEST COMMANDS
# ==============================================================================
# Additional test phases for Blender functionality
cli::register_subcommand "test" "render" "Test rendering capability" "blender::test"
cli::register_subcommand "test" "physics" "Test physics simulation" "blender::test::physics"
cli::register_subcommand "test" "validation" "Validate physics accuracy" "blender::test::validation"

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi