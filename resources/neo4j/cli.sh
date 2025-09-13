#!/usr/bin/env bash
################################################################################
# Neo4j Resource CLI - v2.0 Universal Contract Compliant
# 
# Native property graph database with Cypher query language
#
# Usage:
#   resource-neo4j <command> [options]
#   resource-neo4j <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    NEO4J_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    APP_ROOT="$(builtin cd "${NEO4J_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
NEO4J_CLI_DIR="${APP_ROOT}/resources/neo4j"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"
# shellcheck disable=SC1091
source "${NEO4J_CLI_DIR}/config/defaults.sh"

# Source Neo4j libraries
for lib in common install start status inject core; do
    lib_file="${NEO4J_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework in v2.0 mode (auto-creates manage/test/content groups)
cli::init "neo4j" "Neo4j graph database management" "v2"

# Override default handlers to point directly to neo4j implementations
CLI_COMMAND_HANDLERS["manage::install"]="neo4j_install"
CLI_COMMAND_HANDLERS["manage::uninstall"]="neo4j_uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="neo4j_start"  
CLI_COMMAND_HANDLERS["manage::stop"]="neo4j_stop"
CLI_COMMAND_HANDLERS["manage::restart"]="neo4j_restart"
CLI_COMMAND_HANDLERS["test::smoke"]="neo4j_status"

# Override content handlers for Neo4j-specific graph database functionality
CLI_COMMAND_HANDLERS["content::add"]="neo4j_inject"
CLI_COMMAND_HANDLERS["content::list"]="neo4j_list_injected" 
CLI_COMMAND_HANDLERS["content::get"]="neo4j::content::get"
CLI_COMMAND_HANDLERS["content::remove"]="neo4j::content::remove"
CLI_COMMAND_HANDLERS["content::execute"]="neo4j_query"

# Add Neo4j-specific content subcommands for graph operations
cli::register_subcommand "content" "query" "Execute Cypher query" "neo4j_query"
cli::register_subcommand "content" "backup" "Backup graph database" "neo4j::content::backup" "modifies-system"
cli::register_subcommand "content" "restore" "Restore graph database" "neo4j::content::restore" "modifies-system"

# Performance monitoring commands
cli::register_subcommand "content" "metrics" "Get performance metrics" "neo4j_get_performance_metrics"
cli::register_subcommand "content" "monitor" "Monitor query performance" "neo4j_monitor_query"
cli::register_subcommand "content" "slow-queries" "Get slow queries" "neo4j_get_slow_queries"

# Additional information commands
cli::register_command "status" "Show detailed resource status" "neo4j_status"
cli::register_command "logs" "Show Neo4j logs" "neo4j_logs"

# Define missing content and docker functions
neo4j::content::get() {
    echo "Content retrieval not implemented for Neo4j"
    echo "Use 'content query' to run Cypher queries instead"
    return 1
}

neo4j::content::remove() {
    echo "Content removal not implemented for Neo4j"
    echo "Use 'content query' to run deletion Cypher queries instead"
    return 1
}

neo4j::content::backup() {
    # Use the core backup function
    neo4j_backup "$@"
}

neo4j::content::restore() {
    local backup_file="${1:-}"
    
    if [[ -z "$backup_file" ]]; then
        echo "Error: Backup file name required"
        echo "Usage: resource-neo4j content restore <backup-file>"
        echo ""
        echo "Available backups:"
        docker exec "$NEO4J_CONTAINER_NAME" ls -1 /var/lib/neo4j/data/dumps/ 2>/dev/null | head -10 || echo "  No backups found"
        return 1
    fi
    
    if ! neo4j_is_running; then
        echo "Error: Neo4j is not running"
        return 1
    fi
    
    # Check file extension to determine restore method
    if [[ "$backup_file" == *.cypher ]]; then
        echo "Restoring from Cypher backup: $backup_file"
        echo "Clearing existing data..."
        neo4j_query "MATCH (n) DETACH DELETE n" &>/dev/null
        
        echo "Importing backup data..."
        docker exec "$NEO4J_CONTAINER_NAME" bash -c "
            cat /var/lib/neo4j/data/dumps/$backup_file | cypher-shell -u neo4j -p '${NEO4J_AUTH#*/}'
        " 2>/dev/null || {
            echo "Error: Restore failed"
            return 1
        }
    elif [[ "$backup_file" == *.json ]]; then
        echo "Restoring from JSON backup: $backup_file"
        echo "Clearing existing data..."
        neo4j_query "MATCH (n) DETACH DELETE n" &>/dev/null
        
        echo "Importing JSON data..."
        neo4j_query "CALL apoc.import.json('/var/lib/neo4j/data/dumps/$backup_file')" || {
            echo "Error: JSON restore failed (APOC required)"
            return 1
        }
    else
        echo "Error: Unsupported backup format. Use .cypher or .json files"
        return 1
    fi
    
    # Verify restoration
    local node_count=$(neo4j_query "MATCH (n) RETURN count(n)" 2>/dev/null | tail -1)
    echo "Restore completed. Nodes in database: ${node_count:-unknown}"
    
    return 0
}

neo4j_logs() {
    docker logs "${NEO4J_CONTAINER_NAME}" --tail 50 2>&1 || echo "Neo4j container not running"
}

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi