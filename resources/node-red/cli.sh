#!/usr/bin/env bash
################################################################################
# Node-RED Resource CLI - v2.0 Universal Contract Compliant
# 
# Flow-based programming tool for wiring together hardware devices, APIs and online services
#
# Usage:
#   resource-node-red <command> [options]
#   resource-node-red <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

# Determine the actual directory of this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_ROOT="${APP_ROOT:-$(cd "$SCRIPT_DIR/../.." && pwd)}"
NODE_RED_CLI_DIR="$SCRIPT_DIR"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"
# shellcheck disable=SC1091
source "${NODE_RED_CLI_DIR}/config/defaults.sh"
node_red::export_config 2>/dev/null || true

# Source Node-RED libraries
for lib in core docker health recovery test performance iot-integration; do
    lib_file="${NODE_RED_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework in v2.0 mode (auto-creates manage/test/content groups)
cli::init "node-red" "Node-RED flow-based automation platform" "v2"

# ==============================================================================
# REQUIRED HANDLERS - Direct mapping to Node-RED functions
# ==============================================================================
CLI_COMMAND_HANDLERS["manage::install"]="node_red::install"
CLI_COMMAND_HANDLERS["manage::uninstall"]="node_red::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="node_red::docker::start"
CLI_COMMAND_HANDLERS["manage::stop"]="node_red::docker::stop"
CLI_COMMAND_HANDLERS["manage::restart"]="node_red::docker::restart"

# Test handlers - v2.0 compliant
CLI_COMMAND_HANDLERS["test::smoke"]="node_red::test::smoke"
CLI_COMMAND_HANDLERS["test::integration"]="node_red::test::integration"
CLI_COMMAND_HANDLERS["test::unit"]="node_red::test::unit"
CLI_COMMAND_HANDLERS["test::all"]="node_red::test::all"

# Content handlers for flow management functionality
CLI_COMMAND_HANDLERS["content::add"]="node_red::import_flows"
CLI_COMMAND_HANDLERS["content::list"]="node_red::list_flows"
CLI_COMMAND_HANDLERS["content::get"]="node_red::export_flows"
CLI_COMMAND_HANDLERS["content::remove"]="node_red::disable_flow"
CLI_COMMAND_HANDLERS["content::execute"]="node_red::inject"

# ==============================================================================
# REQUIRED INFORMATION COMMANDS
# ==============================================================================
cli::register_command "status" "Show detailed resource status" "node_red::status"
cli::register_command "logs" "Show Node-RED logs" "node_red::view_logs"

# ==============================================================================
# OPTIONAL CUSTOM COMMANDS - Node-RED specific functionality
# ==============================================================================
cli::register_command "credentials" "Show Node-RED credentials for integration" "node_red::cli_credentials"

# Custom content subcommands for flow and backup management
cli::register_subcommand "content" "backup" "Create flow backup" "node_red::create_backup" "modifies-system"
cli::register_subcommand "content" "restore" "Restore flow backup" "node_red::restore_backup" "modifies-system"
cli::register_subcommand "content" "list-backups" "List available backups" "node_red::list_backups"
cli::register_subcommand "content" "enable" "Enable specific flow" "node_red::enable_flow" "modifies-system"
cli::register_subcommand "content" "disable" "Disable specific flow" "node_red::disable_flow" "modifies-system"

# Performance optimization commands
cli::register_command "optimize" "Apply performance optimizations" "node_red::apply_all_optimizations" "modifies-system"
cli::register_command "monitor" "Monitor performance metrics" "node_red::monitor_performance"

# IoT integration commands
cli::register_command "iot-setup" "Setup IoT integration" "node_red::setup_iot_integration" "modifies-system"
cli::register_command "iot-nodes" "Install IoT nodes" "node_red::install_iot_nodes" "modifies-system"

# Simple credentials implementation for Node-RED
node_red::cli_credentials() {
    if docker ps --format "table {{.Names}}\t{{.Status}}" | grep -q "${NODE_RED_CONTAINER_NAME:-node-red}.*Up"; then
        echo "Node-RED is running at:"
        echo "  Web UI: ${NODE_RED_BASE_URL:-http://localhost:1880}"
        echo "  API: ${NODE_RED_BASE_URL:-http://localhost:1880}/flows"
        echo "  Port: ${NODE_RED_PORT:-1880}"
    else
        echo "Node-RED is not running. Start it with: resource-node-red manage start"
    fi
}

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi