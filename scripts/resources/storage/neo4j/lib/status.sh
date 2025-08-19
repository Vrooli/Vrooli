#!/usr/bin/env bash
# Neo4j Resource - Status Functions

# Get script directory and source common
NEO4J_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$NEO4J_LIB_DIR/common.sh"
source "$NEO4J_LIB_DIR/start.sh"

# Source format utility for consistent output
FORMAT_UTIL="${NEO4J_LIB_DIR}/../../../lib/utils/format.sh"
[[ -f "$FORMAT_UTIL" ]] && source "$FORMAT_UTIL"

neo4j_status() {
    local format="text"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --format)
                format="${2:-text}"
                shift 2
                ;;
            json|text)
                # Support positional format argument for backward compatibility
                format="$1"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    local status_data=""
    
    # Check if installed
    local installed="false"
    if neo4j_is_installed; then
        installed="true"
    fi
    
    # Check if running
    local running="false"
    if neo4j_is_running; then
        running="true"
    fi
    
    # Check health
    local healthy="false"
    local message="Not running"
    local version=""
    local nodes="0"
    local relationships="0"
    
    if [[ "$running" == "true" ]]; then
        # Try to get basic status
        if curl -s "http://localhost:$NEO4J_HTTP_PORT/" 2>/dev/null | grep -q '"neo4j_version"'; then
            healthy="true"
            message="Neo4j is healthy and accepting connections"
            
            # Try to get version
            version=$(docker exec "$NEO4J_CONTAINER_NAME" neo4j --version 2>/dev/null | grep -oP 'neo4j \K[0-9.]+' || echo "unknown")
            
            # Try to get node/relationship count (requires auth)
            local cypher_result=$(echo 'MATCH (n) RETURN count(n) as nodes' | \
                docker exec -i "$NEO4J_CONTAINER_NAME" cypher-shell \
                -u neo4j -p "VrooliNeo4j2024!" \
                --format plain 2>/dev/null | tail -1)
            if [[ -n "$cypher_result" ]]; then
                nodes="$cypher_result"
            fi
        else
            message="Neo4j is running but not responding"
        fi
    elif [[ "$installed" == "true" ]]; then
        message="Neo4j is installed but not running"
    else
        message="Neo4j is not installed"
    fi
    
    # Determine status string
    local status="stopped"
    if [[ "$running" == "true" ]]; then
        status="running"
    elif [[ "$installed" == "true" ]]; then
        status="stopped"
    else
        status="not_installed"
    fi
    
    # Build the key-value pairs
    local kv_pairs=(
        "name" "neo4j"
        "status" "$status"
        "installed" "$installed"
        "running" "$running"
        "health" "$healthy"
        "healthy" "$healthy"
        "version" "$version"
        "http_port" "$NEO4J_HTTP_PORT"
        "bolt_port" "$NEO4J_BOLT_PORT"
        "nodes" "$nodes"
        "relationships" "$relationships"
        "message" "$message"
        "description" "Native property graph database"
        "category" "storage"
    )
    
    # Use format utility if available, fall back to simple output
    if declare -f format::output >/dev/null 2>&1; then
        format::output "$format" "kv" "${kv_pairs[@]}"
    else
        # Fallback to simple format
        if [[ "$format" == "json" ]]; then
            cat <<EOF
{
  "name": "neo4j",
  "status": "$status",
  "installed": $installed,
  "running": $running,
  "health": $healthy,
  "healthy": $healthy,
  "version": "$version",
  "http_port": $NEO4J_HTTP_PORT,
  "bolt_port": $NEO4J_BOLT_PORT,
  "nodes": $nodes,
  "relationships": $relationships,
  "message": "$message",
  "description": "Native property graph database",
  "category": "storage"
}
EOF
        else
            cat <<EOF
Installed: $installed
Running: $running
Healthy: $healthy
Version: $version
Http Port: $NEO4J_HTTP_PORT
Bolt Port: $NEO4J_BOLT_PORT
Nodes: $nodes
Relationships: $relationships
Message: $message
Description: Native property graph database
Category: storage
EOF
        fi
    fi
}