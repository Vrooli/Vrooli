#!/bin/bash

# Wiki.js Installation Functions

set -euo pipefail

# Source common functions
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
WIKIJS_LIB_DIR="${APP_ROOT}/resources/wikijs/lib"
source "$WIKIJS_LIB_DIR/common.sh"

# Install Wiki.js
install_wikijs() {
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
    
    echo "[INFO] Installing Wiki.js..."
    
    # Check if already installed
    if is_installed; then
        echo "[INFO] Wiki.js is already installed"
        return 0
    fi
    
    if [[ "$dry_run" == true ]]; then
        echo "[DRY-RUN] Would install Wiki.js:"
        echo "  - Create data directory: $WIKIJS_DATA_DIR"
        echo "  - Pull Docker image: $WIKIJS_IMAGE"
        echo "  - Create database: $WIKIJS_DB_NAME"
        echo "  - Start container: $WIKIJS_CONTAINER"
        return 0
    fi
    
    # Create data directory
    echo "[INFO] Creating data directory..."
    mkdir -p "$WIKIJS_DATA_DIR"/{data,content,backup}
    
    # Ensure database exists
    ensure_database
    
    # Get configuration
    local port=$(get_wikijs_port)
    local db_config=$(get_db_config)
    local db_host=$(echo "$db_config" | grep -oP 'host=\K\S+')
    local db_port=$(echo "$db_config" | grep -oP 'port=\K\d+')
    local db_pass=$(echo "$db_config" | grep -oP 'pass=\K\S+')
    
    # Create configuration file
    echo "[INFO] Creating configuration..."
    cat > "$WIKIJS_CONFIG_FILE" << EOF
# Wiki.js Configuration
port: $port
db:
  type: postgres
  host: $db_host
  port: $db_port
  user: $WIKIJS_DB_USER
  pass: $db_pass
  db: $WIKIJS_DB_NAME
  ssl: false
trustProxy: true
logLevel: info
uploads:
  maxFileSize: 100
  maxFiles: 20
EOF
    
    # Pull Docker image
    echo "[INFO] Pulling Wiki.js Docker image..."
    docker pull "$WIKIJS_IMAGE"
    
    # Start Wiki.js container
    echo "[INFO] Starting Wiki.js container..."
    docker run -d \
        --name "$WIKIJS_CONTAINER" \
        --restart unless-stopped \
        -p "${port}:3000" \
        -e "DB_TYPE=postgres" \
        -e "DB_HOST=$db_host" \
        -e "DB_PORT=$db_port" \
        -e "DB_USER=$WIKIJS_DB_USER" \
        -e "DB_PASS=$db_pass" \
        -e "DB_NAME=$WIKIJS_DB_NAME" \
        -v "$WIKIJS_DATA_DIR/data:/wiki/data" \
        -v "$WIKIJS_DATA_DIR/content:/wiki/content" \
        "$WIKIJS_IMAGE"
    
    # Wait for Wiki.js to be ready
    if wait_for_ready; then
        echo "[SUCCESS] Wiki.js installed successfully!"
        echo "[INFO] Access Wiki.js at: http://localhost:${port}"
        
        # Register CLI
        register_cli
    else
        echo "[ERROR] Wiki.js installation completed but service is not ready"
        return 1
    fi
}

# Register Wiki.js CLI with Vrooli
register_cli() {
    local install_script="${var_SCRIPT_DIR:-/root/Vrooli/scripts}/lib/utils/install-resource-cli.sh"
    
    if [[ -f "$install_script" ]]; then
        echo "[INFO] Registering Wiki.js CLI..."
        "$install_script" "wikijs" "$WIKIJS_DIR/cli.sh" || {
            echo "[WARNING] Failed to register CLI with Vrooli"
        }
    fi
}