#!/usr/bin/env bash
# Geth Stop Functions

# Get the script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
GETH_STOP_DIR="${APP_ROOT}/resources/geth/lib"

# Source common functions
# shellcheck disable=SC1091
source "${GETH_STOP_DIR}/common.sh"

# Stop Geth
geth::stop() {
    echo "[INFO] Stopping Geth..."
    
    if ! geth::is_running; then
        echo "[INFO] Geth is not running"
        return 0
    fi
    
    # Stop container gracefully
    if docker stop "${GETH_CONTAINER_NAME}" --time 10; then
        echo "[SUCCESS] Geth stopped successfully"
        return 0
    else
        echo "[ERROR] Failed to stop Geth"
        return 1
    fi
}

# Main execution
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    geth::stop
fi