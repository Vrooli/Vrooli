#!/usr/bin/env bash
################################################################################
# Neo4j Resource CLI
# 
# Ultra-thin CLI wrapper that delegates directly to library functions
#
# Usage:
#   resource-neo4j <command> [options]
#
################################################################################

set -euo pipefail

# Get script directory (resolving symlinks for installed CLI)
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    # If this is a symlink, resolve it
    NEO4J_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
else
    NEO4J_CLI_SCRIPT="${BASH_SOURCE[0]}"
fi
NEO4J_CLI_DIR="$(cd "$(dirname "$NEO4J_CLI_SCRIPT")" && pwd)"

# Source standard variables
# shellcheck disable=SC1091
source "${NEO4J_CLI_DIR}/../../../lib/utils/var.sh"

# Source utilities using var_ variables
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"

# Source the CLI Command Framework
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/cli-command-framework.sh"

# Source Neo4j libraries - these contain the actual functionality
# shellcheck disable=SC1091
source "${NEO4J_CLI_DIR}/lib/common.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${NEO4J_CLI_DIR}/lib/install.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${NEO4J_CLI_DIR}/lib/start.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${NEO4J_CLI_DIR}/lib/status.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${NEO4J_CLI_DIR}/lib/inject.sh" 2>/dev/null || true

################################################################################
# Define CLI commands
################################################################################

# Help command
neo4j::show_help() {
    cat <<EOF
Neo4j Graph Database Resource

Usage: resource-neo4j <command> [options]

Core Commands:
  install             Install Neo4j
  uninstall          Uninstall Neo4j  
  start              Start Neo4j container
  stop               Stop Neo4j container
  restart            Restart Neo4j container
  status [format]    Show status (format: text|json)
  logs               Show Neo4j logs
  help               Show this help message

Data Management:
  inject <file>      Inject Cypher file
  list-injected      List injected files
  query <cypher>     Execute Cypher query

Examples:
  resource-neo4j install
  resource-neo4j start
  resource-neo4j status json
  resource-neo4j inject schema.cypher
  resource-neo4j query "MATCH (n) RETURN count(n)"

Default Port: 7474 (HTTP), 7687 (Bolt)
Web UI: http://localhost:7474
Features: Native graph database with Cypher query language
EOF
}

# Wrapper for logs (if not already defined)
neo4j::logs() {
    docker logs neo4j --tail 50 2>&1 || echo "Neo4j container not running"
}

################################################################################
# Register CLI commands
################################################################################

# Help
cli::register_command "help" "Show this help message with examples" "neo4j::show_help"

# Service management
cli::register_command "install" "Install Neo4j" "neo4j_install" "modifies-system"
cli::register_command "uninstall" "Uninstall Neo4j" "neo4j_uninstall" "modifies-system"
cli::register_command "start" "Start Neo4j" "neo4j_start" "modifies-system"
cli::register_command "stop" "Stop Neo4j" "neo4j_stop" "modifies-system"
cli::register_command "restart" "Restart Neo4j" "neo4j_restart" "modifies-system"

# Status & validation
cli::register_command "status" "Show service status" "neo4j_status"
cli::register_command "logs" "Show Neo4j logs" "neo4j::logs"

# Data management
cli::register_command "inject" "Inject Cypher file into Neo4j" "neo4j_inject" "modifies-system"
cli::register_command "list-injected" "List injected files" "neo4j_list_injected"
cli::register_command "query" "Execute Cypher query" "neo4j_query"

################################################################################
# Main execution - dispatch to framework
################################################################################

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi