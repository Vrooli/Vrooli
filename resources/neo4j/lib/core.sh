#!/usr/bin/env bash
################################################################################
# Neo4j Resource - Core Library Functions
# 
# Central library for Neo4j graph database management operations
################################################################################

set -euo pipefail

# Source dependencies
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
NEO4J_RESOURCE_DIR="${APP_ROOT}/resources/neo4j"

# Source common utilities
source "${NEO4J_RESOURCE_DIR}/lib/common.sh"
source "${NEO4J_RESOURCE_DIR}/lib/install.sh"
source "${NEO4J_RESOURCE_DIR}/lib/start.sh"
source "${NEO4J_RESOURCE_DIR}/lib/status.sh"
source "${NEO4J_RESOURCE_DIR}/lib/inject.sh" 2>/dev/null || true

################################################################################
# Core Neo4j Operations
################################################################################

#######################################
# Execute a Cypher query
# Args:
#   $1 - Cypher query to execute
# Returns:
#   Query results or error
#######################################
neo4j_query() {
    local query="${1:-}"
    
    if [[ -z "$query" ]]; then
        echo "Error: No query provided"
        echo "Usage: neo4j_query 'MATCH (n) RETURN count(n)'"
        return 1
    fi
    
    if ! neo4j_is_running; then
        echo "Error: Neo4j is not running"
        return 1
    fi
    
    # Execute query using cypher-shell
    echo "$query" | docker exec -i "$NEO4J_CONTAINER_NAME" cypher-shell \
        -u neo4j -p "${NEO4J_AUTH#*/}" \
        --format plain 2>/dev/null || {
        echo "Error: Failed to execute query"
        return 1
    }
}

#######################################
# List injected content (Cypher files)
# Returns:
#   List of injected files
#######################################
neo4j_list_injected() {
    local inject_dir="${NEO4J_RESOURCE_DIR}/injected"
    
    if [[ ! -d "$inject_dir" ]]; then
        echo "No injected content found"
        return 0
    fi
    
    echo "Injected Cypher files:"
    find "$inject_dir" -name "*.cypher" -o -name "*.cql" | while read -r file; do
        echo "  - $(basename "$file")"
    done
}

#######################################
# Validate Neo4j health with timeout
# Returns:
#   0 if healthy, 1 otherwise
#######################################
neo4j_health_check() {
    if ! neo4j_is_running; then
        return 1
    fi
    
    # Check HTTP endpoint with timeout
    if timeout 5 curl -sf "http://localhost:${NEO4J_HTTP_PORT}/" &>/dev/null; then
        return 0
    fi
    
    return 1
}

#######################################
# Get Neo4j version
# Returns:
#   Version string or "unknown"
#######################################
neo4j_get_version() {
    if ! neo4j_is_running; then
        echo "unknown"
        return 1
    fi
    
    docker exec "$NEO4J_CONTAINER_NAME" neo4j --version 2>/dev/null | \
        grep -oP 'neo4j \K[0-9.]+' || echo "unknown"
}

#######################################
# Get node and relationship counts
# Returns:
#   JSON with counts
#######################################
neo4j_get_stats() {
    if ! neo4j_is_running; then
        echo '{"nodes": 0, "relationships": 0}'
        return 1
    fi
    
    local nodes=$(neo4j_query "MATCH (n) RETURN count(n)" 2>/dev/null | tail -1 || echo "0")
    local rels=$(neo4j_query "MATCH ()-[r]->() RETURN count(r)" 2>/dev/null | tail -1 || echo "0")
    
    echo "{\"nodes\": ${nodes:-0}, \"relationships\": ${rels:-0}}"
}

#######################################
# Wait for Neo4j to be ready
# Args:
#   $1 - Max wait time in seconds (default: 30)
# Returns:
#   0 if ready, 1 if timeout
#######################################
neo4j_wait_ready() {
    local max_wait="${1:-30}"
    local elapsed=0
    
    echo "Waiting for Neo4j to be ready..."
    while (( elapsed < max_wait )); do
        if neo4j_health_check; then
            echo "Neo4j is ready"
            return 0
        fi
        sleep 2
        ((elapsed += 2))
    done
    
    echo "Timeout: Neo4j did not become ready in ${max_wait} seconds"
    return 1
}

#######################################
# Create database backup
# Args:
#   $1 - Backup file path (optional)
# Returns:
#   0 on success, 1 on failure
#######################################
neo4j_backup() {
    local backup_path="${1:-}"
    
    if ! neo4j_is_running; then
        echo "Error: Neo4j is not running"
        return 1
    fi
    
    if [[ -z "$backup_path" ]]; then
        backup_path="${NEO4J_DATA_DIR}/backups/neo4j-$(date +%Y%m%d-%H%M%S).dump"
    fi
    
    # Ensure backup directory exists
    mkdir -p "$(dirname "$backup_path")"
    
    echo "Creating backup: $backup_path"
    docker exec "$NEO4J_CONTAINER_NAME" neo4j-admin database dump neo4j \
        --to-path=/data/backups 2>/dev/null || {
        echo "Error: Backup failed"
        return 1
    }
    
    echo "Backup completed successfully"
    return 0
}

#######################################
# Import CSV data
# Args:
#   $1 - CSV file path
#   $2 - Cypher import query
# Returns:
#   0 on success, 1 on failure
#######################################
neo4j_import_csv() {
    local csv_file="${1:-}"
    local import_query="${2:-}"
    
    if [[ -z "$csv_file" ]] || [[ -z "$import_query" ]]; then
        echo "Error: CSV file and import query required"
        echo "Usage: neo4j_import_csv /path/to/file.csv 'LOAD CSV...'"
        return 1
    fi
    
    if [[ ! -f "$csv_file" ]]; then
        echo "Error: CSV file not found: $csv_file"
        return 1
    fi
    
    # Copy CSV to import directory
    cp "$csv_file" "$NEO4J_IMPORT_DIR/"
    local csv_name=$(basename "$csv_file")
    
    # Execute import query
    neo4j_query "$import_query" || {
        echo "Error: Import failed"
        return 1
    }
    
    echo "CSV import completed successfully"
    return 0
}

# Export all functions
export -f neo4j_query neo4j_list_injected neo4j_health_check
export -f neo4j_get_version neo4j_get_stats neo4j_wait_ready
export -f neo4j_backup neo4j_import_csv