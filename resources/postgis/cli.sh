#!/usr/bin/env bash
################################################################################
# PostGIS Resource CLI - v2.0 Universal Contract Compliant
# 
# Spatial database extension for PostgreSQL with GIS capabilities
#
# Usage:
#   resource-postgis <command> [options]
#   resource-postgis <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    POSTGIS_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    APP_ROOT="$(builtin cd "${POSTGIS_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
POSTGIS_CLI_DIR="${APP_ROOT}/resources/postgis"

# Source standard variables
source "${APP_ROOT}/scripts/lib/utils/var.sh"

# Source utilities
source "${var_LOG_FILE}"
source "${var_RESOURCES_COMMON_FILE}"

# Source v2.0 CLI Command Framework
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"

# Source resource configuration
source "${POSTGIS_CLI_DIR}/config/defaults.sh"

# Source resource libraries (only what exists)
for lib in core common install status test inject; do
    lib_file="${POSTGIS_CLI_DIR}/lib/${lib}.sh"
    [[ -f "$lib_file" ]] && source "$lib_file" 2>/dev/null || true
done

# Initialize CLI framework in v2.0 mode (auto-creates manage/test/content groups)
cli::init "postgis" "Spatial database extension for PostgreSQL" "v2"

# ==============================================================================
# REQUIRED HANDLERS - These MUST be mapped for v2.0 compliance
# ==============================================================================
CLI_COMMAND_HANDLERS["manage::install"]="postgis::install::execute"
CLI_COMMAND_HANDLERS["manage::uninstall"]="postgis::install::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="postgis::docker::start"
CLI_COMMAND_HANDLERS["manage::stop"]="postgis::docker::stop"
CLI_COMMAND_HANDLERS["manage::restart"]="postgis::docker::restart"
CLI_COMMAND_HANDLERS["test::smoke"]="postgis::status::check"
CLI_COMMAND_HANDLERS["test::integration"]="postgis::test::integration"

# Content handlers for spatial data management
CLI_COMMAND_HANDLERS["content::add"]="postgis::content::add"
CLI_COMMAND_HANDLERS["content::list"]="postgis::content::list"
CLI_COMMAND_HANDLERS["content::get"]="postgis::content::get"
CLI_COMMAND_HANDLERS["content::remove"]="postgis::content::remove"
CLI_COMMAND_HANDLERS["content::execute"]="postgis::content::execute"

# ==============================================================================
# REQUIRED INFORMATION COMMANDS
# ==============================================================================
cli::register_command "status" "Show detailed PostGIS status" "postgis::status"
cli::register_command "logs" "Show PostGIS container logs" "postgis::docker::logs"

# ==============================================================================
# OPTIONAL RESOURCE-SPECIFIC COMMANDS FOR SPATIAL DATA
# ==============================================================================
# Add spatial-specific content subcommands
cli::register_subcommand "content" "import-shapefile" "Import shapefile to PostGIS" "postgis_import_shapefile" "modifies-system"
cli::register_subcommand "content" "export-shapefile" "Export PostGIS table to shapefile" "postgis_export_shapefile" "modifies-system"
cli::register_subcommand "content" "enable-database" "Enable PostGIS in specific database" "postgis_enable_database" "modifies-system"
cli::register_subcommand "content" "disable-database" "Disable PostGIS in specific database" "postgis_disable_database" "modifies-system"

# Add spatial query examples command
cli::register_command "examples" "Show example spatial queries" "postgis_show_examples"

# Only execute if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi