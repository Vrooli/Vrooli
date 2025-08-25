#!/usr/bin/env bash
# Geth Start Functions

# Get the script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
GETH_START_DIR="${APP_ROOT}/resources/geth/lib"

# Source common functions
# shellcheck disable=SC1091
source "${GETH_START_DIR}/common.sh"

# Start Geth
geth::start() {
    echo "[INFO] Starting Geth..."
    
    # Check if installed
    if ! geth::is_installed; then
        echo "[ERROR] Geth is not installed. Run 'install' first."
        return 1
    fi
    
    # Check if already running
    if geth::is_running; then
        echo "[INFO] Geth is already running"
        return 0
    fi
    
    # Start container
    if docker start "${GETH_CONTAINER_NAME}"; then
        echo "[INFO] Geth container started"
        
        # Wait for Geth to be ready
        echo "[INFO] Waiting for Geth to be ready..."
        local max_attempts=30
        local attempt=0
        
        while [[ $attempt -lt $max_attempts ]]; do
            if geth::health_check 2; then
                echo "[SUCCESS] Geth is ready!"
                
                # Display connection info
                echo ""
                echo "[INFO] Geth Connection Info:"
                echo "  JSON-RPC: http://localhost:${GETH_PORT}"
                echo "  WebSocket: ws://localhost:${GETH_WS_PORT}"
                echo "  Network: ${GETH_NETWORK}"
                
                if [[ "${GETH_NETWORK}" == "dev" ]]; then
                    echo "  Chain ID: ${GETH_CHAIN_ID}"
                    echo "  Dev Mode: Enabled (auto-mining)"
                fi
                
                return 0
            fi
            
            attempt=$((attempt + 1))
            sleep 2
        done
        
        echo "[WARN] Geth started but may not be fully ready"
        return 0
    else
        echo "[ERROR] Failed to start Geth container"
        return 1
    fi
}

# Main execution
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    geth::start
fi