#!/usr/bin/env bash
# Open MCT Configuration Defaults

# Get port from central registry
DEFAULTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${DEFAULTS_DIR}/../../../scripts/resources/port_registry.sh"

# Service Configuration
export OPENMCT_PORT="${OPENMCT_PORT:-$(ports::get_resource_port "openmct")}"
export OPENMCT_HOST="${OPENMCT_HOST:-0.0.0.0}"
export OPENMCT_CONTAINER_NAME="${OPENMCT_CONTAINER_NAME:-vrooli-openmct}"
export OPENMCT_IMAGE="${OPENMCT_IMAGE:-vrooli/openmct:latest}"

# Data Configuration
export OPENMCT_DATA_DIR="${OPENMCT_DATA_DIR:-${HOME}/.vrooli/openmct/data}"
export OPENMCT_CONFIG_DIR="${OPENMCT_CONFIG_DIR:-${HOME}/.vrooli/openmct/config}"
export OPENMCT_PLUGINS_DIR="${OPENMCT_PLUGINS_DIR:-${HOME}/.vrooli/openmct/plugins}"

# Telemetry Configuration
export OPENMCT_MAX_STREAMS="${OPENMCT_MAX_STREAMS:-100}"
export OPENMCT_HISTORY_DAYS="${OPENMCT_HISTORY_DAYS:-30}"
export OPENMCT_SAMPLE_RATE="${OPENMCT_SAMPLE_RATE:-1000}"  # ms between samples
export OPENMCT_COMPRESS_HISTORY="${OPENMCT_COMPRESS_HISTORY:-true}"

# WebSocket Configuration
export OPENMCT_WS_PORT="${OPENMCT_WS_PORT:-8198}"
export OPENMCT_WS_BUFFER_SIZE="${OPENMCT_WS_BUFFER_SIZE:-65536}"
export OPENMCT_WS_MAX_CONNECTIONS="${OPENMCT_WS_MAX_CONNECTIONS:-100}"

# Authentication Configuration
export OPENMCT_AUTH_ENABLED="${OPENMCT_AUTH_ENABLED:-false}"
export OPENMCT_AUTH_USERNAME="${OPENMCT_AUTH_USERNAME:-admin}"
export OPENMCT_AUTH_PASSWORD="${OPENMCT_AUTH_PASSWORD:-}"

# Performance Configuration
export OPENMCT_CACHE_ENABLED="${OPENMCT_CACHE_ENABLED:-true}"
export OPENMCT_CACHE_TTL="${OPENMCT_CACHE_TTL:-3600}"  # seconds
export OPENMCT_DB_WAL_MODE="${OPENMCT_DB_WAL_MODE:-true}"  # SQLite WAL mode

# Logging Configuration
export OPENMCT_LOG_LEVEL="${OPENMCT_LOG_LEVEL:-info}"
export OPENMCT_LOG_FILE="${OPENMCT_LOG_FILE:-${HOME}/.vrooli/openmct/logs/openmct.log}"
export OPENMCT_LOG_MAX_SIZE="${OPENMCT_LOG_MAX_SIZE:-10M}"
export OPENMCT_LOG_MAX_FILES="${OPENMCT_LOG_MAX_FILES:-5}"

# Docker Configuration
export OPENMCT_MEMORY_LIMIT="${OPENMCT_MEMORY_LIMIT:-1g}"
export OPENMCT_CPU_LIMIT="${OPENMCT_CPU_LIMIT:-1.0}"
export OPENMCT_RESTART_POLICY="${OPENMCT_RESTART_POLICY:-unless-stopped}"

# Telemetry Provider Defaults
export OPENMCT_ENABLE_DEMO="${OPENMCT_ENABLE_DEMO:-true}"
export OPENMCT_DEMO_SATELLITE="${OPENMCT_DEMO_SATELLITE:-true}"
export OPENMCT_DEMO_SENSORS="${OPENMCT_DEMO_SENSORS:-true}"
export OPENMCT_DEMO_METRICS="${OPENMCT_DEMO_METRICS:-true}"

# Integration Defaults
export OPENMCT_MQTT_ENABLED="${OPENMCT_MQTT_ENABLED:-false}"
export OPENMCT_MQTT_BROKER="${OPENMCT_MQTT_BROKER:-}"
export OPENMCT_TRACCAR_ENABLED="${OPENMCT_TRACCAR_ENABLED:-false}"
export OPENMCT_TRACCAR_URL="${OPENMCT_TRACCAR_URL:-}"

# Create required directories
mkdir -p "$OPENMCT_DATA_DIR" "$OPENMCT_CONFIG_DIR" "$OPENMCT_PLUGINS_DIR" "$(dirname "$OPENMCT_LOG_FILE")"