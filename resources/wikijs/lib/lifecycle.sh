#!/bin/bash

# Wiki.js Lifecycle Management Functions

set -euo pipefail

# Source common functions
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
WIKIJS_LIB_DIR="${APP_ROOT}/resources/wikijs/lib"
source "$WIKIJS_LIB_DIR/common.sh"

# Start Wiki.js
start_wikijs() {
    local dry_run=false
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --dry-run)
                dry_run=true
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    echo "[INFO] Starting Wiki.js..."
    
    # Check if installed
    if ! is_installed; then
        echo "[ERROR] Wiki.js is not installed. Run: vrooli resource wikijs install"
        return 1
    fi
    
    # Check if already running
    if is_running; then
        echo "[INFO] Wiki.js is already running"
        return 0
    fi
    
    if [[ "$dry_run" == true ]]; then
        echo "[DRY-RUN] Would start Wiki.js container"
        return 0
    fi
    
    # Check container status
    local status=$(get_container_status)
    
    if [[ "$status" == "stopped" ]]; then
        # Container exists but stopped
        echo "[INFO] Starting existing container..."
        docker start "$WIKIJS_CONTAINER"
    else
        # Container doesn't exist, create it
        echo "[INFO] Creating and starting container..."
        local port=$(get_wikijs_port)
        local db_config=$(get_db_config)
        local db_host=$(echo "$db_config" | grep -oP 'host=\K\S+')
        local db_port=$(echo "$db_config" | grep -oP 'port=\K\d+')
        local db_pass=$(echo "$db_config" | grep -oP 'pass=\K\S+')
        
        # Use host network mode to access host's PostgreSQL
        docker run -d \
            --name "$WIKIJS_CONTAINER" \
            --restart unless-stopped \
            --network host \
            -e "DB_TYPE=postgres" \
            -e "DB_HOST=localhost" \
            -e "DB_PORT=$db_port" \
            -e "DB_USER=$WIKIJS_DB_USER" \
            -e "DB_PASS=$db_pass" \
            -e "DB_NAME=$WIKIJS_DB_NAME" \
            -v "$WIKIJS_DATA_DIR/data:/wiki/data" \
            -v "$WIKIJS_DATA_DIR/content:/wiki/content" \
            "$WIKIJS_IMAGE"
    fi
    
    # Wait for Wiki.js to be ready
    if wait_for_ready; then
        echo "[SUCCESS] Wiki.js started successfully!"
        local port=$(get_wikijs_port)
        echo "[INFO] Access Wiki.js at: http://localhost:${port}"
    else
        echo "[ERROR] Wiki.js started but is not responding"
        return 1
    fi
}

# Stop Wiki.js
stop_wikijs() {
    local dry_run=false
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --dry-run)
                dry_run=true
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    echo "[INFO] Stopping Wiki.js..."
    
    # Check if running
    if ! is_running; then
        echo "[INFO] Wiki.js is not running"
        return 0
    fi
    
    if [[ "$dry_run" == true ]]; then
        echo "[DRY-RUN] Would stop Wiki.js container"
        return 0
    fi
    
    # Stop container
    echo "[INFO] Stopping container..."
    docker stop "$WIKIJS_CONTAINER"
    
    echo "[SUCCESS] Wiki.js stopped successfully"
}

# Restart Wiki.js
restart_wikijs() {
    local dry_run=false
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --dry-run)
                dry_run=true
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    echo "[INFO] Restarting Wiki.js..."
    
    if [[ "$dry_run" == true ]]; then
        echo "[DRY-RUN] Would restart Wiki.js"
        return 0
    fi
    
    # Stop if running
    if is_running; then
        stop_wikijs
        sleep 2
    fi
    
    # Start Wiki.js
    start_wikijs
}