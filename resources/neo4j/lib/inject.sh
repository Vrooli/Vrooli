#!/usr/bin/env bash
# Neo4j Resource - Injection Functions

# Get script directory and source common
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
NEO4J_LIB_DIR="${APP_ROOT}/resources/neo4j/lib"
source "$NEO4J_LIB_DIR/common.sh"
source "$NEO4J_LIB_DIR/start.sh"

neo4j_inject() {
    local file_path="$1"
    
    if [[ -z "$file_path" ]]; then
        echo "Error: File path required"
        return 1
    fi
    
    if [[ ! -f "$file_path" ]]; then
        echo "Error: File not found: $file_path"
        return 1
    fi
    
    # Ensure Neo4j is running
    if ! neo4j_is_running; then
        echo "Error: Neo4j is not running. Start it with: vrooli resource start neo4j"
        return 1
    fi
    
    # Get filename
    local filename=$(basename "$file_path")
    local injected_dir="$NEO4J_RESOURCE_DIR/injected"
    
    # Create injected directory if needed
    mkdir -p "$injected_dir"
    
    # Copy file to injected directory
    cp "$file_path" "$injected_dir/$filename"
    
    # Execute Cypher file (use password from NEO4J_AUTH environment variable)
    local password="${NEO4J_AUTH#*/}"  # Extract password from neo4j/password format
    echo "Injecting Cypher queries from $filename..."
    cat "$file_path" | docker exec -i "$NEO4J_CONTAINER_NAME" cypher-shell \
        -u neo4j -p "$password" \
        --format plain || {
        echo "Error: Failed to execute Cypher queries"
        return 1
    }
    
    echo "Successfully injected: $filename"
    return 0
}

neo4j_list_injected() {
    local injected_dir="$NEO4J_RESOURCE_DIR/injected"
    
    if [[ ! -d "$injected_dir" ]]; then
        echo "No injected files found"
        return 0
    fi
    
    echo "Injected Cypher files:"
    ls -la "$injected_dir"/*.cypher 2>/dev/null || echo "  No .cypher files found"
}

neo4j_query() {
    local query="$1"
    
    if [[ -z "$query" ]]; then
        echo "Error: Query required"
        return 1
    fi
    
    if ! neo4j_is_running; then
        echo "Error: Neo4j is not running"
        return 1
    fi
    
    # Execute query (use password from NEO4J_AUTH environment variable)
    local password="${NEO4J_AUTH#*/}"  # Extract password from neo4j/password format
    echo "$query" | docker exec -i "$NEO4J_CONTAINER_NAME" cypher-shell \
        -u neo4j -p "$password" \
        --format plain || {
        echo "Error: Failed to execute query"
        return 1
    }
}