#!/usr/bin/env bash
# Geth Docker Operations

# Start Geth container
geth::docker::start() {
    echo "[INFO] Starting Geth..."
    
    # Check if installed
    if ! geth::is_installed; then
        echo "[ERROR] Geth is not installed. Run 'manage install' first."
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

# Stop Geth container
geth::docker::stop() {
    echo "[INFO] Stopping Geth..."
    
    if ! geth::is_running; then
        echo "[INFO] Geth is not running"
        return 0
    fi
    
    if docker stop "${GETH_CONTAINER_NAME}"; then
        echo "[SUCCESS] Geth stopped successfully"
        return 0
    else
        echo "[ERROR] Failed to stop Geth"
        return 1
    fi
}

# Restart Geth container
geth::docker::restart() {
    echo "[INFO] Restarting Geth..."
    
    geth::docker::stop
    sleep 2
    geth::docker::start
}

# Show Geth logs
geth::docker::logs() {
    local lines="${1:-100}"
    local follow="${2:-}"
    
    if ! docker::container_exists "${GETH_CONTAINER_NAME}"; then
        echo "[ERROR] Geth container does not exist"
        return 1
    fi
    
    if [[ "$follow" == "follow" ]] || [[ "$follow" == "-f" ]]; then
        docker logs --tail "$lines" -f "${GETH_CONTAINER_NAME}"
    else
        docker logs --tail "$lines" "${GETH_CONTAINER_NAME}"
    fi
}