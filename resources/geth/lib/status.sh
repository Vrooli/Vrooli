#!/usr/bin/env bash
# Geth Status Functions

# Get the script directory
GETH_STATUS_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# Source common functions
# shellcheck disable=SC1091
source "${GETH_STATUS_DIR}/common.sh"

# Get Geth status
geth::status() {
    local format="${1:-text}"
    
    # Gather status information
    local installed="false"
    local running="false"
    local healthy="false"
    local status_message="Geth not installed"
    local container_status="not_found"
    local sync_status="unknown"
    local block_number="0"
    local network_id="unknown"
    local peer_count="0"
    
    if geth::is_installed; then
        installed="true"
        container_status=$(geth::container_status)
        
        if geth::is_running; then
            running="true"
            
            if geth::health_check 2; then
                healthy="true"
                status_message="Geth is running and healthy"
                sync_status=$(geth::get_sync_status)
                block_number=$(geth::get_block_number)
                network_id=$(geth::get_network_id)
                peer_count=$(geth::get_peer_count)
            else
                status_message="Geth is running but not responding"
            fi
        else
            status_message="Geth is installed but not running"
        fi
    fi
    
    # Format output based on requested format
    if [[ "$format" == "json" ]]; then
        format::output json kv \
            "installed" "$installed" \
            "running" "$running" \
            "healthy" "$healthy" \
            "status_message" "$status_message" \
            "container_status" "$container_status" \
            "network" "${GETH_NETWORK}" \
            "chain_id" "${GETH_CHAIN_ID}" \
            "network_id" "$network_id" \
            "sync_status" "$sync_status" \
            "block_number" "$block_number" \
            "peer_count" "$peer_count" \
            "json_rpc_endpoint" "http://localhost:${GETH_PORT}" \
            "websocket_endpoint" "ws://localhost:${GETH_WS_PORT}" \
            "p2p_port" "${GETH_P2P_PORT}" \
            "version" "${GETH_VERSION}" \
            "container_name" "${GETH_CONTAINER_NAME}" \
            "data_directory" "${GETH_DATA_DIR}"
    else
        echo "⚡ Geth Status"
        echo "============="
        echo ""
        echo "Basic Status:"
        echo "  Installed:        $installed"
        echo "  Running:          $running"
        echo "  Health:           $([[ "$healthy" == "true" ]] && echo "Healthy" || echo "Unhealthy")"
        echo ""
        echo "Blockchain Info:"
        echo "  Network:          ${GETH_NETWORK}"
        echo "  Chain ID:         ${GETH_CHAIN_ID}"
        echo "  Network ID:       $network_id"
        echo "  Sync Status:      $sync_status"
        echo "  Block Number:     $block_number"
        echo "  Peer Count:       $peer_count"
        echo ""
        echo "Service Endpoints:"
        echo "  JSON-RPC:         http://localhost:${GETH_PORT}"
        echo "  WebSocket:        ws://localhost:${GETH_WS_PORT}"
        echo "  P2P Port:         ${GETH_P2P_PORT}"
        echo ""
        echo "Configuration:"
        echo "  Version:          ${GETH_VERSION}"
        echo "  Container:        ${GETH_CONTAINER_NAME}"
        echo "  Data Directory:   ${GETH_DATA_DIR}"
        echo ""
        echo "Status: $status_message"
    fi
}

# Main execution
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    geth::status "${1:-text}"
fi