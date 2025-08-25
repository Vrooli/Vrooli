#!/usr/bin/env bash
# Neo4j Resource - Installation Functions

# Get script directory and source common
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
NEO4J_LIB_DIR="${APP_ROOT}/resources/neo4j/lib"
source "$NEO4J_LIB_DIR/common.sh"

neo4j_is_installed() {
    docker image inspect "$NEO4J_IMAGE" >/dev/null 2>&1
}

neo4j_install() {
    local skip_pull="${1:-false}"
    
    # Create directories
    mkdir -p "$NEO4J_DATA_DIR" "$NEO4J_LOGS_DIR" "$NEO4J_IMPORT_DIR" "$NEO4J_PLUGINS_DIR" "$NEO4J_CONF_DIR"
    
    # Pull image unless skipped
    if [[ "$skip_pull" != "true" ]]; then
        echo "Pulling Neo4j image..."
        docker pull "$NEO4J_IMAGE" || {
            echo "Error: Failed to pull Neo4j image"
            return 1
        }
    fi
    
    echo "Neo4j installed successfully"
    return 0
}

neo4j_uninstall() {
    # Stop container if running
    if docker ps -q -f name="$NEO4J_CONTAINER_NAME" | grep -q .; then
        echo "Stopping Neo4j container..."
        docker stop "$NEO4J_CONTAINER_NAME" >/dev/null 2>&1
    fi
    
    # Remove container
    if docker ps -aq -f name="$NEO4J_CONTAINER_NAME" | grep -q .; then
        echo "Removing Neo4j container..."
        docker rm -f "$NEO4J_CONTAINER_NAME" >/dev/null 2>&1
    fi
    
    # Remove image
    if neo4j_is_installed; then
        echo "Removing Neo4j image..."
        docker rmi "$NEO4J_IMAGE" >/dev/null 2>&1
    fi
    
    echo "Neo4j uninstalled successfully"
    return 0
}