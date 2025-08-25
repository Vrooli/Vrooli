#!/usr/bin/env bash
# Neo4j Resource - Start/Stop Functions

# Get script directory and source common
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
NEO4J_LIB_DIR="${APP_ROOT}/resources/neo4j/lib"
source "$NEO4J_LIB_DIR/common.sh"
source "$NEO4J_LIB_DIR/install.sh"

neo4j_is_running() {
    docker ps -q -f name="$NEO4J_CONTAINER_NAME" | grep -q .
}

neo4j_start() {
    # Check if already running
    if neo4j_is_running; then
        echo "Neo4j is already running"
        return 0
    fi
    
    # Install if not installed
    if ! neo4j_is_installed; then
        echo "Neo4j not installed. Installing..."
        neo4j_install || return 1
    fi
    
    # Remove old container if exists
    if docker ps -aq -f name="$NEO4J_CONTAINER_NAME" | grep -q .; then
        docker rm -f "$NEO4J_CONTAINER_NAME" >/dev/null 2>&1
    fi
    
    echo "Starting Neo4j..."
    docker run -d \
        --name "$NEO4J_CONTAINER_NAME" \
        --restart unless-stopped \
        -p "$NEO4J_HTTP_PORT:7474" \
        -p "$NEO4J_BOLT_PORT:7687" \
        -v "$NEO4J_DATA_DIR:/data" \
        -v "$NEO4J_LOGS_DIR:/logs" \
        -v "$NEO4J_IMPORT_DIR:/var/lib/neo4j/import" \
        -v "$NEO4J_PLUGINS_DIR:/plugins" \
        --env NEO4J_AUTH="$NEO4J_AUTH" \
        --env NEO4J_server_memory_heap_initial__size="$NEO4J_HEAP_SIZE" \
        --env NEO4J_server_memory_heap_max__size="$NEO4J_HEAP_SIZE" \
        --env NEO4J_PLUGINS='["apoc"]' \
        "$NEO4J_IMAGE" >/dev/null 2>&1 || {
        echo "Error: Failed to start Neo4j"
        return 1
    }
    
    # Wait for Neo4j to be ready
    echo "Waiting for Neo4j to be ready..."
    local max_attempts=30
    local attempt=0
    while (( attempt < max_attempts )); do
        if curl -s "http://localhost:$NEO4J_HTTP_PORT/" | grep -q "neo4j_version"; then
            echo "Neo4j is ready"
            return 0
        fi
        sleep 2
        ((attempt++))
    done
    
    echo "Warning: Neo4j may not be fully ready yet"
    return 0
}

neo4j_stop() {
    if ! neo4j_is_running; then
        echo "Neo4j is not running"
        return 0
    fi
    
    echo "Stopping Neo4j..."
    docker stop "$NEO4J_CONTAINER_NAME" >/dev/null 2>&1 || {
        echo "Error: Failed to stop Neo4j"
        return 1
    }
    
    echo "Neo4j stopped successfully"
    return 0
}

neo4j_restart() {
    neo4j_stop
    neo4j_start
}