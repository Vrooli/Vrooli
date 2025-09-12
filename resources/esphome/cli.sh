#!/usr/bin/env bash
################################################################################
# ESPHome Resource CLI - v2.0 Universal Contract Compliant
# 
# Firmware framework for ESP32/ESP8266 microcontrollers using YAML configuration
#
# Usage:
#   resource-esphome <command> [options]
#   resource-esphome <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    ESPHOME_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    APP_ROOT="$(builtin cd "${ESPHOME_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
ESPHOME_CLI_DIR="${APP_ROOT}/resources/esphome"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"
# shellcheck disable=SC1091
source "${ESPHOME_CLI_DIR}/config/defaults.sh"
esphome::export_config 2>/dev/null || true

# Source ESPHome libraries
for lib in core test; do
    lib_file="${ESPHOME_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework in v2.0 mode (auto-creates manage/test/content groups)
cli::init "esphome" "ESPHome IoT firmware framework for ESP32/ESP8266" "v2"

# ==============================================================================
# REQUIRED HANDLERS - Direct mapping to ESPHome functions
# ==============================================================================
CLI_COMMAND_HANDLERS["manage::install"]="esphome::install"
CLI_COMMAND_HANDLERS["manage::uninstall"]="esphome::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="esphome::start"
CLI_COMMAND_HANDLERS["manage::stop"]="esphome::stop"
CLI_COMMAND_HANDLERS["manage::restart"]="esphome::restart"

# Test handlers - v2.0 compliant
CLI_COMMAND_HANDLERS["test::smoke"]="esphome::test::smoke"
CLI_COMMAND_HANDLERS["test::integration"]="esphome::test::integration"
CLI_COMMAND_HANDLERS["test::unit"]="esphome::test::unit"
CLI_COMMAND_HANDLERS["test::all"]="esphome::test::all"

# Content handlers for device and configuration management
CLI_COMMAND_HANDLERS["content::add"]="esphome::add_config"
CLI_COMMAND_HANDLERS["content::list"]="esphome::list_configs"
CLI_COMMAND_HANDLERS["content::get"]="esphome::get_config"
CLI_COMMAND_HANDLERS["content::remove"]="esphome::remove_config"
CLI_COMMAND_HANDLERS["content::execute"]="esphome::compile"

# ==============================================================================
# REQUIRED INFORMATION COMMANDS
# ==============================================================================
cli::register_command "info" "Show structured resource information" "esphome::info"
cli::register_command "status" "Show detailed resource status" "esphome::status"
cli::register_command "logs" "Show ESPHome logs" "esphome::view_logs"

# ==============================================================================
# OPTIONAL CUSTOM COMMANDS - ESPHome specific functionality
# ==============================================================================
cli::register_command "credentials" "Show ESPHome dashboard credentials" "esphome::cli_credentials"

# Device management commands
cli::register_command "discover" "Discover ESP devices on network" "esphome::discover_devices"
cli::register_command "upload" "Upload firmware to device via OTA" "esphome::upload_ota" "modifies-system"
cli::register_command "monitor" "Monitor device serial output" "esphome::monitor_serial"

# Configuration commands
cli::register_command "validate" "Validate YAML configuration" "esphome::validate_config"
cli::register_command "clean" "Clean build artifacts" "esphome::clean_build" "modifies-system"

# Simple credentials implementation for ESPHome
esphome::cli_credentials() {
    if docker ps --format "table {{.Names}}\t{{.Status}}" | grep -q "${ESPHOME_CONTAINER_NAME:-esphome}.*Up"; then
        echo "ESPHome Dashboard is running at:"
        echo "  Web UI: ${ESPHOME_BASE_URL:-http://localhost:6587}"
        echo "  API: ${ESPHOME_BASE_URL:-http://localhost:6587}/api"
        echo ""
        echo "Default credentials (if configured):"
        echo "  Username: admin"
        echo "  Password: (check ESPHOME_DASHBOARD_PASSWORD environment variable)"
    else
        echo "ESPHome is not currently running."
        echo "Start it with: resource-esphome manage start"
    fi
}

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi