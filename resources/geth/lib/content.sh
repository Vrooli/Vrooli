#!/usr/bin/env bash
# Geth Content Operations (Business Functionality)
# Handles blockchain operations, smart contracts, and transactions

# Deploy a smart contract
geth::content::add() {
    local contract_file="${1:-}"
    
    if [[ -z "$contract_file" ]]; then
        echo "[ERROR] Contract file required"
        echo "Usage: resource-geth content add <contract.sol>"
        return 1
    fi
    
    if [[ ! -f "$contract_file" ]]; then
        echo "[ERROR] Contract file not found: $contract_file"
        return 1
    fi
    
    # This would normally deploy the contract
    geth::inject deploy "$contract_file"
}

# List accounts or deployed contracts
geth::content::list() {
    geth::list_accounts
}

# Get balance or contract details
geth::content::get() {
    local address="${1:-}"
    
    if [[ -z "$address" ]]; then
        echo "[ERROR] Address required"
        echo "Usage: resource-geth content get <address>"
        return 1
    fi
    
    geth::get_balance "$address"
}

# Remove/clear accounts (dev mode only)
geth::content::remove() {
    if [[ "$GETH_NETWORK" != "dev" ]]; then
        echo "[ERROR] Account removal only available in dev mode"
        return 1
    fi
    
    echo "[WARNING] This will clear all dev accounts"
    read -p "Are you sure? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "[INFO] Operation cancelled"
        return 0
    fi
    
    # Clear dev accounts
    rm -rf "${GETH_DATA_DIR}/data/keystore"
    echo "[INFO] Dev accounts cleared"
}

# Execute transaction or script
geth::content::execute() {
    local script_file="${1:-}"
    
    if [[ -z "$script_file" ]]; then
        echo "[ERROR] Script file required"
        echo "Usage: resource-geth content execute <script.js>"
        return 1
    fi
    
    if [[ ! -f "$script_file" ]]; then
        echo "[ERROR] Script file not found: $script_file"
        return 1
    fi
    
    geth::inject script "$script_file"
}

# Custom command: Open Geth console
geth::content::console() {
    if ! geth::is_running; then
        echo "[ERROR] Geth is not running"
        return 1
    fi
    
    echo "[INFO] Attaching to Geth console..."
    echo "[INFO] Type 'exit' to leave the console"
    docker exec -it "${GETH_CONTAINER_NAME}" geth attach
}

# Custom command: Show current block
geth::content::block() {
    local block
    block=$(geth::get_block_number)
    echo "Current block: $block"
}

# Custom command: Show connected peers
geth::content::peers() {
    local count
    count=$(geth::get_peer_count)
    echo "Connected peers: $count"
    
    if [[ "$count" -gt 0 ]]; then
        local response
        response=$(curl -s -X POST \
            -H "Content-Type: application/json" \
            --data '{"jsonrpc":"2.0","method":"admin_peers","params":[],"id":1}' \
            "http://localhost:${GETH_PORT}" 2>/dev/null)
        
        if [[ -n "$response" ]]; then
            echo "$response" | jq '.result[]' 2>/dev/null | head -20
        fi
    fi
}

# Custom command: Show sync status
geth::content::sync() {
    local status
    status=$(geth::get_sync_status)
    echo "Sync status: $status"
    
    if [[ "$status" == "syncing" ]]; then
        local response
        response=$(curl -s -X POST \
            -H "Content-Type: application/json" \
            --data '{"jsonrpc":"2.0","method":"eth_syncing","params":[],"id":1}' \
            "http://localhost:${GETH_PORT}" 2>/dev/null)
        
        if [[ -n "$response" ]]; then
            echo "$response" | jq '.result' 2>/dev/null
        fi
    fi
}