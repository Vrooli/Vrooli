#!/usr/bin/env bash
# mcrcon default configuration

# Server configuration
export MCRCON_HOST="${MCRCON_HOST:-localhost}"
export MCRCON_PORT="${MCRCON_PORT:-25575}"
export MCRCON_PASSWORD="${MCRCON_PASSWORD:-}"

# Connection settings
export MCRCON_TIMEOUT="${MCRCON_TIMEOUT:-30}"
export MCRCON_RETRY_ATTEMPTS="${MCRCON_RETRY_ATTEMPTS:-3}"
export MCRCON_AUTO_DISCOVER="${MCRCON_AUTO_DISCOVER:-true}"

# Runtime configuration
export MCRCON_DATA_DIR="${MCRCON_DATA_DIR:-${HOME}/.mcrcon}"
export MCRCON_CONFIG_FILE="${MCRCON_CONFIG_FILE:-${MCRCON_DATA_DIR}/servers.json}"
export MCRCON_LOG_FILE="${MCRCON_LOG_FILE:-${MCRCON_DATA_DIR}/mcrcon.log}"

# Binary paths
export MCRCON_BINARY="${MCRCON_BINARY:-${MCRCON_DATA_DIR}/bin/mcrcon}"

# Health check configuration
export MCRCON_HEALTH_PORT="${MCRCON_HEALTH_PORT:-8025}"
export MCRCON_HEALTH_TIMEOUT="${MCRCON_HEALTH_TIMEOUT:-5}"

# Debug settings
export MCRCON_DEBUG="${MCRCON_DEBUG:-false}"
export MCRCON_VERBOSE="${MCRCON_VERBOSE:-false}"