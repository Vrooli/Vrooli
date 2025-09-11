#!/usr/bin/env bash
################################################################################
# FreeCAD Resource CLI - v2.0 Universal Contract Compliant
# 
# Parametric 3D CAD modeler with Python API
#
# Usage:
#   resource-freecad <command> [options]
#   resource-freecad <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    FREECAD_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    # Recalculate APP_ROOT from resolved symlink location
    APP_ROOT="$(builtin cd "${FREECAD_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
FREECAD_CLI_DIR="${APP_ROOT}/resources/freecad"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"
# shellcheck disable=SC1091
source "${FREECAD_CLI_DIR}/config/defaults.sh"

# Source FreeCAD libraries
for lib in core test; do
    lib_file="${FREECAD_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework in v2.0 mode (auto-creates manage/test/content groups)
cli::init "freecad" "Parametric 3D CAD modeler with Python API" "v2"

# Register FreeCAD-specific content subcommands
cli::register_subcommand "content" "generate" "Generate CAD from Python script" "freecad::content::generate"
cli::register_subcommand "content" "export" "Export model to various formats" "freecad::content::export"
cli::register_subcommand "content" "analyze" "Run FEM/simulation analysis" "freecad::content::analyze"

# Information commands
cli::register_command "status" "Show detailed FreeCAD status" "freecad::status"
cli::register_command "logs" "Show FreeCAD logs" "freecad::logs"

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi