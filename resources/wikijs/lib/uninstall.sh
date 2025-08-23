#!/bin/bash

# Wiki.js Uninstall Functions

set -euo pipefail

# Source common functions
WIKIJS_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$WIKIJS_LIB_DIR/common.sh"

# Uninstall Wiki.js
uninstall_wikijs() {
    local force=false
    local dry_run=false
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --force)
                force=true
                shift
                ;;
            --dry-run)
                dry_run=true
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Require --force flag
    if [[ "$force" != true ]] && [[ "$dry_run" != true ]]; then
        echo "[ERROR] Uninstall requires --force flag to confirm"
        echo "Usage: vrooli resource wikijs uninstall --force"
        return 1
    fi
    
    echo "[WARNING] Uninstalling Wiki.js..."
    
    if [[ "$dry_run" == true ]]; then
        echo "[DRY-RUN] Would uninstall Wiki.js:"
        echo "  - Stop and remove container: $WIKIJS_CONTAINER"
        echo "  - Remove data directory: $WIKIJS_DATA_DIR"
        echo "  - Drop database: $WIKIJS_DB_NAME (if exists)"
        return 0
    fi
    
    # Stop container if running
    if is_running; then
        echo "[INFO] Stopping Wiki.js container..."
        docker stop "$WIKIJS_CONTAINER"
    fi
    
    # Remove container
    if docker ps -a --format "{{.Names}}" | grep -q "^${WIKIJS_CONTAINER}$"; then
        echo "[INFO] Removing Wiki.js container..."
        docker rm "$WIKIJS_CONTAINER"
    fi
    
    # Remove data directory
    if [[ -d "$WIKIJS_DATA_DIR" ]]; then
        echo "[INFO] Removing data directory..."
        rm -rf "$WIKIJS_DATA_DIR"
    fi
    
    # Drop database (optional, only if PostgreSQL resource is available)
    if command -v resource-postgres &>/dev/null; then
        echo "[INFO] Dropping Wiki.js database..."
        resource-postgres query "DROP DATABASE IF EXISTS $WIKIJS_DB_NAME" 2>/dev/null || true
        resource-postgres query "DROP USER IF EXISTS $WIKIJS_DB_USER" 2>/dev/null || true
    fi
    
    echo "[SUCCESS] Wiki.js uninstalled successfully"
}