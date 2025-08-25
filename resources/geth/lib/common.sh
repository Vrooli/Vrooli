#!/usr/bin/env bash
# Geth Common Utilities
# Shared functions used across Geth management modules

# Source required utilities
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
GETH_LIB_DIR="${APP_ROOT}/resources/geth/lib"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/docker-utils.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/format.sh" 2>/dev/null || true

# Configuration
export GETH_VERSION="${GETH_VERSION:-v1.13.14}"
export GETH_PORT="${GETH_PORT:-8545}"
export GETH_WS_PORT="${GETH_WS_PORT:-8546}"
export GETH_P2P_PORT="${GETH_P2P_PORT:-30303}"
export GETH_CONTAINER_NAME="${GETH_CONTAINER_NAME:-vrooli-geth}"
export GETH_IMAGE="${GETH_IMAGE:-ethereum/client-go:${GETH_VERSION}}"
export GETH_DATA_DIR="${GETH_DATA_DIR:-${HOME}/.vrooli/geth}"
export GETH_NETWORK="${GETH_NETWORK:-dev}"
export GETH_CHAIN_ID="${GETH_CHAIN_ID:-1337}"

# Create required directories
geth::init_directories() {
    mkdir -p "${GETH_DATA_DIR}/data"
    mkdir -p "${GETH_DATA_DIR}/contracts"
    mkdir -p "${GETH_DATA_DIR}/scripts"
    mkdir -p "${GETH_DATA_DIR}/logs"
}

# Check if Geth is installed (container exists)
geth::is_installed() {
    docker::container_exists "${GETH_CONTAINER_NAME}"
}

# Check if Geth is running
geth::is_running() {
    docker::is_running "${GETH_CONTAINER_NAME}"
}

# Get container status
geth::container_status() {
    docker inspect "${GETH_CONTAINER_NAME}" 2>/dev/null | jq -r '.[0].State.Status' 2>/dev/null || echo "not_found"
}

# Health check
geth::health_check() {
    local timeout="${1:-5}"
    if ! geth::is_running; then
        return 1
    fi
    
    # Check JSON-RPC endpoint
    timeout "${timeout}" curl -s -X POST \
        -H "Content-Type: application/json" \
        --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
        "http://localhost:${GETH_PORT}" >/dev/null 2>&1
}

# Get sync status
geth::get_sync_status() {
    if ! geth::is_running; then
        echo "not_running"
        return 1
    fi
    
    local response
    response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        --data '{"jsonrpc":"2.0","method":"eth_syncing","params":[],"id":1}' \
        "http://localhost:${GETH_PORT}" 2>/dev/null)
    
    if [[ -z "$response" ]]; then
        echo "unreachable"
        return 1
    fi
    
    local result
    result=$(echo "$response" | jq -r '.result' 2>/dev/null)
    
    if [[ "$result" == "false" ]]; then
        echo "synced"
    else
        echo "syncing"
    fi
}

# Get block number
geth::get_block_number() {
    if ! geth::is_running; then
        echo "0"
        return 1
    fi
    
    local response
    response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
        "http://localhost:${GETH_PORT}" 2>/dev/null)
    
    if [[ -n "$response" ]]; then
        local hex_number
        hex_number=$(echo "$response" | jq -r '.result' 2>/dev/null)
        if [[ -n "$hex_number" ]] && [[ "$hex_number" != "null" ]]; then
            # Convert hex to decimal
            printf "%d\n" "$hex_number" 2>/dev/null || echo "0"
        else
            echo "0"
        fi
    else
        echo "0"
    fi
}

# Get network ID
geth::get_network_id() {
    if ! geth::is_running; then
        echo "unknown"
        return 1
    fi
    
    local response
    response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        --data '{"jsonrpc":"2.0","method":"net_version","params":[],"id":1}' \
        "http://localhost:${GETH_PORT}" 2>/dev/null)
    
    if [[ -n "$response" ]]; then
        echo "$response" | jq -r '.result' 2>/dev/null || echo "unknown"
    else
        echo "unknown"
    fi
}

# Get peer count
geth::get_peer_count() {
    if ! geth::is_running; then
        echo "0"
        return 1
    fi
    
    local response
    response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        --data '{"jsonrpc":"2.0","method":"net_peerCount","params":[],"id":1}' \
        "http://localhost:${GETH_PORT}" 2>/dev/null)
    
    if [[ -n "$response" ]]; then
        local hex_count
        hex_count=$(echo "$response" | jq -r '.result' 2>/dev/null)
        if [[ -n "$hex_count" ]] && [[ "$hex_count" != "null" ]]; then
            # Convert hex to decimal
            printf "%d\n" "$hex_count" 2>/dev/null || echo "0"
        else
            echo "0"
        fi
    else
        echo "0"
    fi
}

# Export configuration for other scripts
geth::export_config() {
    export GETH_VERSION
    export GETH_PORT
    export GETH_WS_PORT
    export GETH_P2P_PORT
    export GETH_CONTAINER_NAME
    export GETH_IMAGE
    export GETH_DATA_DIR
    export GETH_NETWORK
    export GETH_CHAIN_ID
}