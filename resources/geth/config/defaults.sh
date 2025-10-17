#!/usr/bin/env bash
# Geth Resource Configuration Defaults

# Container settings
export GETH_VERSION="${GETH_VERSION:-v1.13.14}"
export GETH_CONTAINER_NAME="${GETH_CONTAINER_NAME:-vrooli-geth}"
export GETH_IMAGE="${GETH_IMAGE:-ethereum/client-go:${GETH_VERSION}}"

# Network ports
export GETH_PORT="${GETH_PORT:-8545}"            # JSON-RPC port
export GETH_WS_PORT="${GETH_WS_PORT:-8546}"      # WebSocket port
export GETH_P2P_PORT="${GETH_P2P_PORT:-30303}"   # P2P port

# Data directories
export GETH_DATA_DIR="${GETH_DATA_DIR:-${HOME}/.vrooli/geth}"

# Network configuration
export GETH_NETWORK="${GETH_NETWORK:-dev}"       # Network mode: dev, mainnet, goerli, sepolia
export GETH_CHAIN_ID="${GETH_CHAIN_ID:-1337}"    # Chain ID for dev network