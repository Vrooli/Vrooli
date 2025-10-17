#!/usr/bin/env bash
# WireGuard Resource Default Configuration

# Port configuration - using dynamic allocation
# Port 51820 is the standard WireGuard port
export WIREGUARD_PORT="${WIREGUARD_PORT:-51820}"

# Network configuration
export WIREGUARD_NETWORK="${WIREGUARD_NETWORK:-10.13.13.0/24}"

# Container configuration
export CONTAINER_NAME="${CONTAINER_NAME:-vrooli-wireguard}"
export WIREGUARD_IMAGE="${WIREGUARD_IMAGE:-lscr.io/linuxserver/wireguard:latest}"

# Path configuration
export CONFIG_DIR="${CONFIG_DIR:-${HOME}/.vrooli/resources/wireguard}"

# Performance settings
export WIREGUARD_PEERS_MAX="${WIREGUARD_PEERS_MAX:-100}"
export WIREGUARD_KEEPALIVE="${WIREGUARD_KEEPALIVE:-25}"

# Security settings
export WIREGUARD_ALLOWED_IPS="${WIREGUARD_ALLOWED_IPS:-0.0.0.0/0}"
export WIREGUARD_DNS="${WIREGUARD_DNS:-1.1.1.1,8.8.8.8}"

# Logging
export WIREGUARD_LOG_LEVEL="${WIREGUARD_LOG_LEVEL:-info}"