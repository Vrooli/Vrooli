#!/bin/bash

# Wiki.js Backup Functions

set -euo pipefail

# Source common functions
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
WIKIJS_LIB_DIR="${APP_ROOT}/resources/wikijs/lib"
source "$WIKIJS_LIB_DIR/common.sh"

# Backup Wiki.js data
backup_wikijs() {
    local backup_dir="$WIKIJS_DATA_DIR/backup"
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local backup_file="$backup_dir/wikijs_backup_${timestamp}.tar.gz"
    
    echo "[INFO] Creating Wiki.js backup..."
    
    # Create backup directory if it doesn't exist
    mkdir -p "$backup_dir"
    
    # Create backup archive
    tar -czf "$backup_file" \
        -C "$WIKIJS_DATA_DIR" \
        data content config.yml \
        2>/dev/null || true
    
    echo "[SUCCESS] Backup created: $backup_file"
    echo "[INFO] Size: $(du -h "$backup_file" | cut -f1)"
}

# Restore Wiki.js backup
restore_wikijs() {
    local backup_file="$1"
    
    if [[ -z "$backup_file" ]]; then
        echo "[ERROR] Backup file required"
        echo "Usage: vrooli resource wikijs restore <backup_file>"
        return 1
    fi
    
    if [[ ! -f "$backup_file" ]]; then
        echo "[ERROR] Backup file not found: $backup_file"
        return 1
    fi
    
    echo "[WARNING] Restoring will overwrite current data"
    echo "[INFO] Restoring from: $backup_file"
    
    # Stop Wiki.js if running
    if is_running; then
        echo "[INFO] Stopping Wiki.js for restore..."
        docker stop "$WIKIJS_CONTAINER"
    fi
    
    # Extract backup
    tar -xzf "$backup_file" -C "$WIKIJS_DATA_DIR"
    
    # Restart Wiki.js if it was running
    echo "[INFO] Starting Wiki.js..."
    docker start "$WIKIJS_CONTAINER" 2>/dev/null || start_wikijs
    
    echo "[SUCCESS] Restore completed from: $backup_file"
}