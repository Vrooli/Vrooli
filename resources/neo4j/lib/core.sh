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
# Install APOC plugin
# Returns:
#   0 on success, 1 on failure
#######################################
neo4j_install_apoc() {
    echo "Installing APOC plugin for enhanced graph algorithms..."
    
    # Use latest compatible APOC version
    local apoc_version="2025.09.0"
    local apoc_url="https://github.com/neo4j/apoc/releases/download/${apoc_version}/apoc-${apoc_version}-core.jar"
    local temp_file="/tmp/apoc-${apoc_version}-core.jar"
    
    # Download APOC jar to temp location first
    if command -v wget >/dev/null 2>&1; then
        wget -q -O "$temp_file" "$apoc_url" || {
            echo "Error: Failed to download APOC plugin"
            return 1
        }
    elif command -v curl >/dev/null 2>&1; then
        curl -sL -o "$temp_file" "$apoc_url" || {
            echo "Error: Failed to download APOC plugin"
            return 1
        }
    else
        echo "Error: Neither wget nor curl available"
        return 1
    fi
    
    # Copy plugin to container if running
    if neo4j_is_running; then
        docker cp "$temp_file" \
            "$NEO4J_CONTAINER_NAME:/var/lib/neo4j/plugins/apoc-${apoc_version}-core.jar" || {
            echo "Error: Failed to copy APOC plugin to container"
            rm -f "$temp_file"
            return 1
        }
        
        # Add APOC configuration
        docker exec "$NEO4J_CONTAINER_NAME" bash -c "
            echo 'apoc.export.file.enabled=true' >> /var/lib/neo4j/conf/apoc.conf
            echo 'apoc.import.file.enabled=true' >> /var/lib/neo4j/conf/apoc.conf
            echo 'apoc.import.file.use_neo4j_config=true' >> /var/lib/neo4j/conf/apoc.conf
        " 2>/dev/null || true
        
        echo "APOC plugin installed. Restart Neo4j to activate: vrooli resource neo4j manage restart"
    else
        # Copy to local plugins dir for future container starts
        if [[ -d "$NEO4J_PLUGINS_DIR" ]]; then
            cp "$temp_file" "$NEO4J_PLUGINS_DIR/" 2>/dev/null || \
                echo "Note: Could not copy to local plugins directory (permission denied). Plugin will be activated when Neo4j starts."
        fi
        echo "APOC plugin downloaded. It will be activated when Neo4j starts."
    fi
    
    # Clean up temp file
    rm -f "$temp_file"
    
    return 0
}

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
    
    # Try cypher-shell first
    local result
    if [[ "$NEO4J_AUTH" == "none" ]]; then
        # No authentication
        result=$(echo "$query" | docker exec -i "$NEO4J_CONTAINER_NAME" cypher-shell \
            --format plain 2>/dev/null)
    else
        # With authentication
        result=$(echo "$query" | docker exec -i "$NEO4J_CONTAINER_NAME" cypher-shell \
            -u neo4j -p "${NEO4J_AUTH#*/}" \
            --format plain 2>/dev/null)
    fi
    
    if [[ -n "$result" ]]; then
        echo "$result"
        return 0
    fi
    
    # Fallback to HTTP API if cypher-shell fails
    local auth_header=""
    if [[ "$NEO4J_AUTH" != "none" ]]; then
        auth_header="-u neo4j:${NEO4J_AUTH#*/}"
    fi
    # Escape the query for JSON
    local escaped_query
    escaped_query=$(echo "$query" | sed 's/"/\\"/g' | tr '\n' ' ')
    local json_query
    json_query=$(cat <<EOF
{
  "statements": [{
    "statement": "${escaped_query}",
    "parameters": {}
  }]
}
EOF
    )
    
    local response
    response=$(timeout 5 curl -sf "$auth_header" \
        -H "Content-Type: application/json" \
        -d "$json_query" \
        "http://localhost:${NEO4J_HTTP_PORT}/db/neo4j/tx/commit" 2>/dev/null)
    
    if [[ -n "$response" ]]; then
        # Parse JSON response and extract data
        echo "$response" | jq -r '.results[0].data[].row | @csv' 2>/dev/null | tr -d '"' || echo "$response"
        return 0
    fi
    
    echo "Error: Failed to execute query"
    return 1
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
    
    # Neo4j --version just outputs the version number directly
    docker exec "$NEO4J_CONTAINER_NAME" neo4j --version 2>/dev/null || echo "unknown"
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
    
    local nodes
    nodes=$(neo4j_query "MATCH (n) RETURN count(n)" 2>/dev/null | tail -1 || echo "0")
    local rels
    rels=$(neo4j_query "MATCH ()-[r]->() RETURN count(r)" 2>/dev/null | tail -1 || echo "0")
    
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
    local backup_file
    local backup_dir="/var/lib/neo4j/data/dumps"
    
    # Parse backup path - if it contains a path separator, extract filename
    if [[ -n "$backup_path" ]]; then
        if [[ "$backup_path" == *"/"* ]]; then
            # Full path provided - extract just the filename for container use
            backup_file=$(basename "$backup_path")
        else
            # Just filename provided
            backup_file="$backup_path"
        fi
    else
        # No argument - generate default name
        backup_file="neo4j-backup-$(date +%Y%m%d-%H%M%S)"
    fi
    
    # Remove any extension - we'll add the appropriate one
    backup_file="${backup_file%.*}"
    
    if ! neo4j_is_running; then
        echo "Error: Neo4j is not running"
        return 1
    fi
    
    # Ensure backup directory exists in container
    docker exec "$NEO4J_CONTAINER_NAME" mkdir -p "$backup_dir" 2>/dev/null || true
    
    echo "Note: Community Edition requires stopping database for backup"
    echo "Creating backup: $backup_file"
    
    # For Community Edition, we need to export data via Cypher
    # This approach works while the database is running
    
    # Try APOC first if available (use relative path for APOC)
    local apoc_result
    apoc_result=$(docker exec "$NEO4J_CONTAINER_NAME" cypher-shell --format plain \
        'CALL apoc.export.json.all("'${backup_file}'.json", {useTypes:true}) YIELD file, nodes, relationships RETURN file' 2>&1 || echo "")
    
    if [[ "$apoc_result" =~ "file" ]] && [[ ! "$apoc_result" =~ "ProcedureNotFound" ]]; then
        echo "Backup completed successfully using APOC: ${backup_file}.json"
        # Copy backup to host (APOC exports to import directory)
        if [[ -n "$backup_path" ]]; then
            if [[ "$backup_path" == *"/"* ]]; then
                # Full path provided - copy to exact location
                docker cp "$NEO4J_CONTAINER_NAME:/var/lib/neo4j/import/${backup_file}.json" "${backup_path}.json" 2>/dev/null || true
            else
                # Just filename - copy to current directory
                docker cp "$NEO4J_CONTAINER_NAME:/var/lib/neo4j/import/${backup_file}.json" "${backup_path}.json" 2>/dev/null || true
            fi
        fi
        return 0
    else
        # Fallback: Create a simple JSON export with all data
        echo "APOC not available, using basic export..."
        
        # First, get all nodes
        local nodes_data
        nodes_data=$(docker exec "$NEO4J_CONTAINER_NAME" cypher-shell --format json \
            'MATCH (n) RETURN id(n) as id, labels(n) as labels, properties(n) as properties' 2>/dev/null || echo "[]")
        
        # Then get all relationships
        local rels_data
        rels_data=$(docker exec "$NEO4J_CONTAINER_NAME" cypher-shell --format json \
            'MATCH (a)-[r]->(b) RETURN id(a) as from, id(b) as to, type(r) as type, properties(r) as properties' 2>/dev/null || echo "[]")
        
        # Create the backup JSON file
        if docker exec "$NEO4J_CONTAINER_NAME" bash -c "cat > ${backup_dir}/${backup_file}.json" <<EOF
{
  "nodes": ${nodes_data:-[]},
  "relationships": ${rels_data:-[]},
  "metadata": {
    "backup_date": "$(date -Iseconds)",
    "format": "json",
    "neo4j_version": "5.15.0"
  }
}
EOF
        then
            echo "Basic backup completed: ${backup_file}.json"
            # Copy backup to host
            if [[ -n "$backup_path" ]]; then
                if [[ "$backup_path" == *"/"* ]]; then
                    # Full path provided - copy to exact location
                    docker cp "$NEO4J_CONTAINER_NAME:${backup_dir}/${backup_file}.json" "${backup_path}.json" 2>/dev/null || {
                        echo "Note: Could not copy backup to ${backup_path}.json"
                        echo "Backup available in container at ${backup_dir}/${backup_file}.json"
                    }
                else
                    # Just filename - copy to current directory
                    docker cp "$NEO4J_CONTAINER_NAME:${backup_dir}/${backup_file}.json" "${backup_path}.json" 2>/dev/null || {
                        echo "Note: Could not copy backup to ${backup_path}.json"
                        echo "Backup available in container at ${backup_dir}/${backup_file}.json"
                    }
                fi
            fi
        else
            echo "Error: Backup failed"
            return 1
        fi
    fi
    
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
    
    # Copy CSV to import directory using docker cp to handle permissions
    local csv_filename
    csv_filename=$(basename "$csv_file")
    docker cp "$csv_file" "$NEO4J_CONTAINER_NAME:/var/lib/neo4j/import/$csv_filename" || {
        echo "Error: Failed to copy CSV file to Neo4j import directory"
        return 1
    }
    
    # Execute import query
    neo4j_query "$import_query" || {
        echo "Error: Import failed"
        return 1
    }
    
    echo "CSV import completed successfully"
    return 0
}

#######################################
# Get performance metrics from Neo4j
# Returns:
#   JSON with performance metrics
#######################################
neo4j_get_performance_metrics() {
    if ! neo4j_is_running; then
        echo '{"error": "Neo4j is not running"}'
        return 1
    fi
    
    # Get basic database stats (Community Edition compatible)
    local node_count
    node_count=$(neo4j_query "MATCH (n) RETURN count(n) as count" 2>/dev/null | grep -oE '[0-9]+' | tail -1 || echo "0")
    local rel_count
    rel_count=$(neo4j_query "MATCH ()-[r]->() RETURN count(r) as count" 2>/dev/null | grep -oE '[0-9]+' | tail -1 || echo "0")
    
    # Get memory usage from container
    local memory_stats
    memory_stats=$(docker stats "$NEO4J_CONTAINER_NAME" --no-stream --format "{{.MemUsage}}" 2>/dev/null | awk '{print $1}' | sed 's/[^0-9.]//g' || echo "0")
    
    # Get database size
    local db_size
    db_size=$(docker exec "$NEO4J_CONTAINER_NAME" du -sh /data/databases/neo4j 2>/dev/null | awk '{print $1}' || echo "unknown")
    
    # Skip transaction info - SHOW TRANSACTIONS can hang in Community Edition
    local tx_count=0
    
    # Build metrics JSON
    cat <<EOF
{
    "node_count": ${node_count:-0},
    "relationship_count": ${rel_count:-0},
    "active_transactions": ${tx_count:-0},
    "memory_usage_mb": ${memory_stats:-0},
    "database_size": "${db_size}",
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
EOF
}

#######################################
# Monitor query performance
# Args:
#   $1 - Query to monitor
# Returns:
#   Execution time and plan
#######################################
neo4j_monitor_query() {
    local query="${1:-}"
    
    if [[ -z "$query" ]]; then
        echo "Error: No query provided"
        return 1
    fi
    
    if ! neo4j_is_running; then
        echo "Error: Neo4j is not running"
        return 1
    fi
    
    # Use EXPLAIN to get query plan and estimated cost
    local start_time
    start_time=$(date +%s%N)
    local explain_result
    explain_result=$(neo4j_query "EXPLAIN $query" 2>/dev/null)
    local end_time
    end_time=$(date +%s%N)
    
    # Calculate execution time in milliseconds
    local exec_time=$(( (end_time - start_time) / 1000000 ))
    
    # Run PROFILE for detailed metrics (note: this actually executes the query)
    local profile_result
    profile_result=$(neo4j_query "PROFILE $query" 2>/dev/null | head -20)
    
    cat <<EOF
Query Performance Analysis
==========================
Query: ${query:0:100}...
Execution Time: ${exec_time}ms

Query Plan:
-----------
$explain_result

Profile Summary:
---------------
$profile_result
EOF
}

#######################################
# Get slow queries from logs
# Args:
#   $1 - Threshold in ms (default: 1000)
# Returns:
#   List of slow queries
#######################################
neo4j_get_slow_queries() {
    local threshold="${1:-1000}"
    
    if ! neo4j_is_running; then
        echo "Error: Neo4j is not running"
        return 1
    fi
    
    echo "Slow Queries (>${threshold}ms)"
    echo "========================="
    
    # Check query log for slow queries
    docker exec "$NEO4J_CONTAINER_NAME" cat /logs/query.log 2>/dev/null | \
        grep -E "ms: [0-9]{4,}" | \
        tail -10 || echo "No slow queries found in logs"
}

################################################################################
# Graph Algorithms (Using APOC)
################################################################################

#######################################
# Run PageRank algorithm
# Args:
#   $1 - Node label (optional, default: all nodes)
#   $2 - Relationship type (optional, default: all)
#   $3 - Iterations (optional, default: 20)
# Returns:
#   Top ranked nodes
#######################################
neo4j_algo_pagerank() {
    local label="${1:-}"
    local rel_type="${2:-}"
    local iterations="${3:-20}"
    
    if ! neo4j_is_running; then
        echo "Error: Neo4j is not running"
        return 1
    fi
    
    # Build query based on parameters
    local query
    if [[ -n "$label" ]] && [[ -n "$rel_type" ]]; then
        query="CALL apoc.algo.pageRank(
            'MATCH (n:$label) RETURN id(n) as id',
            'MATCH (n:$label)-[:$rel_type]->(m:$label) RETURN id(n) as source, id(m) as target',
            {iterations: $iterations}
        ) YIELD node, score
        RETURN node.name as name, score
        ORDER BY score DESC LIMIT 10"
    else
        # Use graph data science procedure if available, fallback to APOC
        query="MATCH (n)
        WITH collect(n) as nodes
        CALL apoc.algo.pageRank(nodes) YIELD node, score
        RETURN node.name as name, node.id as id, score
        ORDER BY score DESC LIMIT 10"
    fi
    
    echo "Running PageRank Algorithm..."
    echo "=============================="
    if result=$(neo4j_query "$query" 2>&1); then
        echo "$result"
    else
        # Fallback to simple degree centrality if PageRank fails
        echo "PageRank not available, using degree centrality..."
        neo4j_query "
            MATCH (n)
            OPTIONAL MATCH (n)-[r]-()
            WITH n, count(r) as degree
            RETURN n.name as name, n.id as id, degree
            ORDER BY degree DESC LIMIT 10
        "
    fi
}

#######################################
# Find shortest path between nodes
# Args:
#   $1 - Start node ID or property match
#   $2 - End node ID or property match
#   $3 - Max hops (optional, default: 15)
# Returns:
#   Shortest path details
#######################################
neo4j_algo_shortest_path() {
    local start="${1:-}"
    local end="${2:-}"
    local max_hops="${3:-15}"
    
    if [[ -z "$start" ]] || [[ -z "$end" ]]; then
        echo "Error: Start and end nodes required"
        echo "Usage: neo4j_algo_shortest_path 'name:\"Node1\"' 'name:\"Node2\"'"
        return 1
    fi
    
    if ! neo4j_is_running; then
        echo "Error: Neo4j is not running"
        return 1
    fi
    
    # Build query for shortest path
    local query="
        MATCH (start {$start}), (end {$end})
        CALL apoc.algo.dijkstra(start, end, '', 'weight', 1.0) 
        YIELD path, weight
        RETURN [n in nodes(path) | n.name] as path, 
               length(path) as hops,
               weight
        LIMIT 1
    "
    
    # Fallback to native shortest path if APOC dijkstra fails
    local fallback_query="
        MATCH (start {$start}), (end {$end})
        MATCH path = shortestPath((start)-[*..${max_hops}]-(end))
        RETURN [n in nodes(path) | coalesce(n.name, n.id, id(n))] as path,
               length(path) as hops
    "
    
    echo "Finding Shortest Path..."
    echo "========================"
    neo4j_query "$query" 2>/dev/null || neo4j_query "$fallback_query"
}

#######################################
# Detect communities using Louvain
# Args:
#   $1 - Node label (optional)
#   $2 - Relationship type (optional)
# Returns:
#   Community assignments
#######################################
neo4j_algo_community_detection() {
    local label="${1:-}"
    local rel_type="${2:-}"
    
    if ! neo4j_is_running; then
        echo "Error: Neo4j is not running"
        return 1
    fi
    
    # Build query for community detection
    local query
    if [[ -n "$label" ]]; then
        query="
            MATCH (n:$label)
            WITH collect(n) as nodes
            CALL apoc.algo.community(nodes, 'community') 
            YIELD node, community
            WITH community, collect(node.name) as members, count(*) as size
            RETURN community, size, members[0..5] as sample_members
            ORDER BY size DESC
            LIMIT 10
        "
    else
        # Use label propagation as fallback
        query="
            CALL apoc.algo.labelPropagation(
                'MATCH (n) RETURN id(n) as id',
                'MATCH (n)-[r]-(m) RETURN id(n) as source, id(m) as target',
                'OUTGOING',
                {iterations: 10}
            )
            YIELD node, label
            WITH label as community, collect(node.name) as members, count(*) as size
            RETURN community, size, members[0..5] as sample_members
            ORDER BY size DESC
            LIMIT 10
        "
    fi
    
    echo "Detecting Communities..."
    echo "========================"
    neo4j_query "$query" 2>/dev/null || {
        echo "Community detection algorithms not available"
        echo "Using connected components as fallback..."
        neo4j_query "
            MATCH (n)
            WITH n, id(n) as component
            WITH component, collect(n.name) as members, count(*) as size
            RETURN component, size, members[0..5] as sample_members
            ORDER BY size DESC
            LIMIT 10
        "
    }
}

#######################################
# Calculate centrality metrics
# Args:
#   $1 - Centrality type (degree|betweenness|closeness)
#   $2 - Node label (optional)
# Returns:
#   Top central nodes
#######################################
neo4j_algo_centrality() {
    local centrality_type="${1:-degree}"
    local label="${2:-}"
    
    if ! neo4j_is_running; then
        echo "Error: Neo4j is not running"
        return 1
    fi
    
    local query
    case "$centrality_type" in
        degree)
            if [[ -n "$label" ]]; then
                query="
                    MATCH (n:$label)
                    OPTIONAL MATCH (n)-[r]-()
                    WITH n, count(r) as degree
                    RETURN n.name as name, id(n) as id, degree as score
                    ORDER BY score DESC
                    LIMIT 10
                "
            else
                query="
                    MATCH (n)
                    OPTIONAL MATCH (n)-[r]-()
                    WITH n, count(r) as degree
                    RETURN n.name as name, id(n) as id, degree as score
                    ORDER BY score DESC
                    LIMIT 10
                "
            fi
            ;;
        betweenness)
            query="
                CALL apoc.algo.betweenness(
                    '${label:-}',
                    '',
                    'BOTH'
                ) YIELD node, score
                RETURN node.name as name, node.id as id, score
                ORDER BY score DESC
                LIMIT 10
            "
            ;;
        closeness)
            query="
                CALL apoc.algo.closeness(
                    '${label:-}',
                    '',
                    'BOTH'
                ) YIELD node, score  
                RETURN node.name as name, node.id as id, score
                ORDER BY score DESC
                LIMIT 10
            "
            ;;
        *)
            echo "Error: Unknown centrality type: $centrality_type"
            echo "Use: degree, betweenness, or closeness"
            return 1
            ;;
    esac
    
    echo "Calculating ${centrality_type} Centrality..."
    echo "==========================================="
    neo4j_query "$query" 2>/dev/null || {
        if [[ "$centrality_type" == "degree" ]]; then
            echo "Error: Failed to calculate centrality"
            return 1
        else
            echo "Advanced centrality not available, falling back to degree..."
            neo4j_algo_centrality "degree" "$label"
        fi
    }
}

#######################################
# Run similarity algorithms
# Args:
#   $1 - Algorithm (jaccard|cosine|overlap)
#   $2 - Node label (optional)
# Returns:
#   Similar node pairs
#######################################
neo4j_algo_similarity() {
    local algo="${1:-jaccard}"
    local label="${2:-}"
    
    if ! neo4j_is_running; then
        echo "Error: Neo4j is not running"
        return 1
    fi
    
    local query
    case "$algo" in
        jaccard)
            query="
                MATCH (n1${label:+:$label})--(neighbor)--(n2${label:+:$label})
                WHERE id(n1) < id(n2)
                WITH n1, n2, count(distinct neighbor) as intersection
                MATCH (n1)--(n1_neighbor)
                WITH n1, n2, intersection, count(distinct n1_neighbor) as n1_degree
                MATCH (n2)--(n2_neighbor) 
                WITH n1, n2, intersection, n1_degree, count(distinct n2_neighbor) as n2_degree
                WITH n1, n2, intersection, n1_degree + n2_degree - intersection as union
                WITH n1, n2, intersection, union,
                     toFloat(intersection) / toFloat(union) as similarity
                WHERE similarity > 0.5
                RETURN n1.name as node1, n2.name as node2, similarity
                ORDER BY similarity DESC
                LIMIT 10
            "
            ;;
        cosine|overlap)
            # For these, we'd need vector embeddings or more complex calculations
            # Fallback to Jaccard
            echo "Note: $algo similarity requires embeddings, using Jaccard instead"
            neo4j_algo_similarity "jaccard" "$label"
            return $?
            ;;
        *)
            echo "Error: Unknown similarity algorithm: $algo"
            echo "Use: jaccard, cosine, or overlap"
            return 1
            ;;
    esac
    
    echo "Calculating ${algo} Similarity..."
    echo "================================="
    neo4j_query "$query"
}

################################################################################
# WebSocket and Real-time Features
################################################################################

#######################################
# Enable transaction event monitoring
# Args:
#   $1 - Event type (node_created|relationship_created|all)
# Returns:
#   Instructions for subscription
#######################################
neo4j_enable_change_stream() {
    # event_type parameter reserved for future use
    # local event_type="${1:-all}"
    
    if ! neo4j_is_running; then
        echo "Error: Neo4j is not running"
        return 1
    fi
    
    echo "Change Streams Configuration"
    echo "============================="
    echo ""
    echo "Neo4j Community Edition does not have built-in WebSocket support."
    echo "To enable real-time subscriptions, consider these approaches:"
    echo ""
    echo "1. **Polling Method** (Available now):"
    echo "   Use the following command to monitor changes:"
    echo "   watch -n 1 'vrooli resource neo4j content query \"MATCH (n) RETURN count(n)\"'"
    echo ""
    echo "2. **Transaction Logs** (Available now):"
    echo "   Monitor transaction logs for changes:"
    echo "   docker exec vrooli-neo4j tail -f /logs/query.log"
    echo ""
    echo "3. **External Solutions** (Requires setup):"
    echo "   - neo4j-streams plugin for Kafka integration"
    echo "   - Custom Node.js/Python service with bolt-driver subscriptions"
    echo "   - GraphQL subscriptions via neo4j-graphql-js"
    echo ""
    echo "4. **Upgrade Path**:"
    echo "   Neo4j Enterprise Edition includes:"
    echo "   - Change Data Capture (CDC)"
    echo "   - Causal clustering with read replicas"
    echo "   - Advanced monitoring capabilities"
    
    return 0
}

#######################################
# Create a simple change monitor
# Args:
#   $1 - Interval in seconds (default: 5)
#   $2 - Query to monitor (default: node count)
# Returns:
#   Continuous output of changes
#######################################
neo4j_monitor_changes() {
    local interval="${1:-5}"
    local query="${2:-MATCH (n) RETURN count(n) as nodes, datetime() as timestamp}"
    
    if ! neo4j_is_running; then
        echo "Error: Neo4j is not running"
        return 1
    fi
    
    echo "Starting change monitor (interval: ${interval}s)"
    echo "Press Ctrl+C to stop"
    echo "=================================="
    
    local last_result=""
    while true; do
        local current_result
        current_result=$(neo4j_query "$query" 2>/dev/null | tail -n +2)
        
        if [[ "$current_result" != "$last_result" ]] && [[ -n "$current_result" ]]; then
            echo "[$(date '+%Y-%m-%d %H:%M:%S')] Change detected:"
            echo "$current_result"
            echo "----------------------------------"
            last_result="$current_result"
        fi
        
        sleep "$interval"
    done
}

#######################################
# Create trigger simulation using APOC
# Args:
#   $1 - Trigger name
#   $2 - Cypher condition
#   $3 - Action query
# Returns:
#   Trigger ID or error
#######################################
neo4j_create_trigger() {
    local trigger_name="${1:-}"
    local condition="${2:-}"
    local action="${3:-}"
    
    if [[ -z "$trigger_name" ]] || [[ -z "$condition" ]] || [[ -z "$action" ]]; then
        echo "Error: All parameters required"
        echo "Usage: neo4j_create_trigger 'name' 'condition' 'action'"
        return 1
    fi
    
    if ! neo4j_is_running; then
        echo "Error: Neo4j is not running"
        return 1
    fi
    
    # APOC triggers are available in newer versions
    local create_trigger_query="
        CALL apoc.trigger.add(
            '$trigger_name',
            '$condition',
            '$action',
            {phase: 'after'}
        ) YIELD name
        RETURN name
    "
    
    echo "Creating trigger: $trigger_name"
    neo4j_query "$create_trigger_query" 2>/dev/null || {
        echo "Note: APOC triggers require apoc.trigger.enabled=true in neo4j.conf"
        echo "Alternative: Use transaction event handlers in your application code"
        return 1
    }
}

#######################################
# List active triggers
# Returns:
#   List of triggers or instructions
#######################################
neo4j_list_triggers() {
    if ! neo4j_is_running; then
        echo "Error: Neo4j is not running"
        return 1
    fi
    
    echo "Checking for triggers..."
    neo4j_query "CALL apoc.trigger.list() YIELD name, query RETURN name, query" 2>/dev/null || {
        echo "Triggers not available in Community Edition."
        echo ""
        echo "For real-time event handling, consider:"
        echo "1. Application-level polling with neo4j_monitor_changes"
        echo "2. Transaction log monitoring"
        echo "3. External event streaming solutions"
        return 0
    }
}

################################################################################
# Clustering Support
################################################################################

#######################################
# Configure clustering (Enterprise only)
# Args:
#   $1 - Mode (single|core|read-replica)
#   $2 - Cluster size (for core mode)
# Returns:
#   Configuration instructions
#######################################
neo4j_configure_cluster() {
    local mode="${1:-single}"
    local size="${2:-3}"
    
    echo "Neo4j Clustering Configuration"
    echo "=============================="
    echo ""
    
    case "$mode" in
        single)
            echo "Current mode: Single Instance (Community Edition)"
            echo "This is the default configuration."
            ;;
        core|read-replica)
            echo "Clustering requires Neo4j Enterprise Edition."
            echo ""
            echo "To enable clustering:"
            echo ""
            echo "1. **Upgrade to Enterprise Edition:**"
            echo "   - Update Docker image to neo4j:5.15.0-enterprise"
            echo "   - Obtain valid license key"
            echo ""
            echo "2. **Core Server Configuration:**"
            echo "   For a ${size}-node cluster, add to neo4j.conf:"
            echo '   ```'
            echo "   dbms.mode=CORE"
            echo "   causal_clustering.minimum_core_cluster_size=${size}"
            echo "   causal_clustering.discovery_type=LIST"
            echo "   causal_clustering.initial_discovery_members=server1:5000,server2:5000,server3:5000"
            echo '   ```'
            echo ""
            echo "3. **Read Replica Configuration:**"
            echo '   ```'
            echo "   dbms.mode=READ_REPLICA"
            echo "   causal_clustering.discovery_type=LIST"
            echo "   causal_clustering.initial_discovery_members=core1:5000,core2:5000,core3:5000"
            echo '   ```'
            echo ""
            echo "4. **Docker Compose Example:**"
            cat <<'EOF'
version: '3'
services:
  neo4j-core-1:
    image: neo4j:5.15.0-enterprise
    environment:
      - NEO4J_dbms_mode=CORE
      - NEO4J_causal__clustering_minimum__core__cluster__size=3
      - NEO4J_causal__clustering_initial__discovery__members=neo4j-core-1:5000,neo4j-core-2:5000,neo4j-core-3:5000
      - NEO4J_AUTH=neo4j/password
    networks:
      - neo4j-cluster
    
  neo4j-core-2:
    # Similar configuration...
    
  neo4j-read-1:
    image: neo4j:5.15.0-enterprise
    environment:
      - NEO4J_dbms_mode=READ_REPLICA
      - NEO4J_causal__clustering_initial__discovery__members=neo4j-core-1:5000,neo4j-core-2:5000,neo4j-core-3:5000
EOF
            ;;
        *)
            echo "Unknown mode: $mode"
            echo "Valid modes: single, core, read-replica"
            return 1
            ;;
    esac
    
    return 0
}

#######################################
# Check cluster status (simulation)
# Returns:
#   Cluster status or single node info
#######################################
neo4j_cluster_status() {
    if ! neo4j_is_running; then
        echo "Error: Neo4j is not running"
        return 1
    fi
    
    echo "Cluster Status"
    echo "=============="
    echo ""
    
    # Try to get cluster info (Enterprise only)
    local cluster_info
    cluster_info=$(neo4j_query "CALL dbms.cluster.overview()" 2>/dev/null)
    
    if [[ -n "$cluster_info" ]] && [[ ! "$cluster_info" =~ "Error" ]]; then
        echo "$cluster_info"
    else
        echo "Mode: Single Instance (Community Edition)"
        echo ""
        echo "Node Information:"
        echo "  Instance: vrooli-neo4j"
        echo "  Role: LEADER (single node)"
        echo "  Status: $(neo4j_is_running && echo 'ONLINE' || echo 'OFFLINE')"
        echo ""
        local stats
        stats=$(neo4j_get_stats)
        echo "Database Statistics:"
        echo "  $stats"
        echo ""
        echo "To enable clustering, upgrade to Enterprise Edition."
    fi
    
    return 0
}

#######################################
# Simulate load balancing configuration
# Args:
#   $1 - Strategy (round-robin|least-connections|ip-hash)
# Returns:
#   Load balancing configuration
#######################################
neo4j_configure_load_balancing() {
    local strategy="${1:-round-robin}"
    
    echo "Load Balancing Configuration"
    echo "============================"
    echo ""
    echo "Strategy: $strategy"
    echo ""
    
    echo "For Community Edition (current):"
    echo "  - Single instance only"
    echo "  - No load balancing needed"
    echo ""
    
    echo "For Enterprise Edition clustering:"
    echo ""
    echo "1. **Bolt+Routing Protocol:**"
    echo "   Connection string: neo4j+s://cluster-endpoint:7687"
    echo "   Automatic routing to appropriate cluster member"
    echo ""
    echo "2. **HAProxy Configuration Example:**"
    cat <<'EOF'
global
    daemon
    
defaults
    mode tcp
    timeout connect 5000ms
    timeout client 30000ms
    timeout server 30000ms
    
frontend neo4j_bolt
    bind *:7687
    default_backend neo4j_bolt_backend
    
backend neo4j_bolt_backend
    balance roundrobin
    server neo4j1 neo4j-core-1:7687 check
    server neo4j2 neo4j-core-2:7687 check
    server neo4j3 neo4j-core-3:7687 check
EOF
    echo ""
    echo "3. **Application-Level Routing:**"
    echo "   Use Neo4j drivers with routing support"
    echo "   Example: neo4j-driver in Node.js, py2neo in Python"
    
    return 0
}

#######################################
# Backup coordination for cluster
# Returns:
#   Backup strategy for clustering
#######################################
neo4j_cluster_backup_strategy() {
    echo "Cluster Backup Strategy"
    echo "======================="
    echo ""
    
    echo "Current Environment: Community Edition (Single Node)"
    echo "  - Use: neo4j_backup command"
    echo "  - Schedule: Cron job or external scheduler"
    echo ""
    
    echo "Enterprise Cluster Backup Strategy:"
    echo ""
    echo "1. **Online Backup (Recommended):**"
    echo "   neo4j-admin backup --from=neo4j-core-1:6362 --to=/backup/path"
    echo "   - No downtime required"
    echo "   - Consistent across cluster"
    echo ""
    echo "2. **Incremental Backups:**"
    echo "   neo4j-admin backup --from=neo4j-core-1:6362 --to=/backup/path --incremental"
    echo "   - Faster subsequent backups"
    echo "   - Less storage required"
    echo ""
    echo "3. **Backup Rotation Script:**"
    cat <<'EOF'
#!/bin/bash
BACKUP_DIR="/backups"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_PATH="${BACKUP_DIR}/neo4j_${TIMESTAMP}"

# Perform backup from any core member
neo4j-admin backup \
    --from=neo4j-core-1:6362 \
    --to="${BACKUP_PATH}" \
    --consistency-check

# Keep only last 7 days of backups
find ${BACKUP_DIR} -name "neo4j_*" -mtime +7 -exec rm -rf {} \;
EOF
    echo ""
    echo "4. **Disaster Recovery:**"
    echo "   - Maintain backups in multiple locations"
    echo "   - Test restore procedures regularly"
    echo "   - Document RTO/RPO requirements"
    
    return 0
}

# Export all functions
export -f neo4j_query neo4j_list_injected neo4j_health_check
export -f neo4j_get_version neo4j_get_stats neo4j_wait_ready
export -f neo4j_backup neo4j_import_csv neo4j_install_apoc
export -f neo4j_get_performance_metrics neo4j_monitor_query neo4j_get_slow_queries
export -f neo4j_algo_pagerank neo4j_algo_shortest_path neo4j_algo_community_detection
export -f neo4j_algo_centrality neo4j_algo_similarity
export -f neo4j_enable_change_stream neo4j_monitor_changes neo4j_create_trigger neo4j_list_triggers
export -f neo4j_configure_cluster neo4j_cluster_status neo4j_configure_load_balancing neo4j_cluster_backup_strategy