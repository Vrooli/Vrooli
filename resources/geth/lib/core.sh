#!/usr/bin/env bash
# Geth Core Operations

# Show Geth credentials/connection info
geth::core::credentials() {
    if ! geth::is_running; then
        echo "[WARNING] Geth is not running. Start it first with 'manage start'"
        echo ""
    fi
    
    echo "Geth Connection Information:"
    echo "============================"
    echo ""
    echo "JSON-RPC Endpoint:"
    echo "  URL: http://localhost:${GETH_PORT}"
    echo "  Port: ${GETH_PORT}"
    echo ""
    echo "WebSocket Endpoint:"
    echo "  URL: ws://localhost:${GETH_WS_PORT}"  
    echo "  Port: ${GETH_WS_PORT}"
    echo ""
    echo "Network Configuration:"
    echo "  Network: ${GETH_NETWORK}"
    
    if [[ "${GETH_NETWORK}" == "dev" ]]; then
        echo "  Chain ID: ${GETH_CHAIN_ID}"
        echo "  Mode: Development (auto-mining enabled)"
    else
        echo "  Mode: ${GETH_NETWORK} network"
    fi
    
    echo ""
    echo "P2P Port: ${GETH_P2P_PORT}"
    echo "Data Directory: ${GETH_DATA_DIR}"
    echo ""
    echo "Example Usage:"
    echo "  curl -X POST -H \"Content-Type: application/json\" \\"
    echo "    --data '{\"jsonrpc\":\"2.0\",\"method\":\"eth_blockNumber\",\"params\":[],\"id\":1}' \\"
    echo "    http://localhost:${GETH_PORT}"
}