#!/bin/bash

# OpenEMS Default Configuration
# Define all environment variables and defaults

# Source port registry
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../../../scripts/resources/port_registry.sh"

# Service ports from registry
export OPENEMS_PORT="${RESOURCE_PORTS[openems]}"
export OPENEMS_JSONRPC_PORT="${RESOURCE_PORTS[openems-jsonrpc]}"
export OPENEMS_MODBUS_PORT="${RESOURCE_PORTS[openems-modbus]}"
export OPENEMS_BACKEND_PORT="${RESOURCE_PORTS[openems-backend]}"

# Validate required ports
if [[ -z "$OPENEMS_PORT" ]]; then
    echo "❌ OPENEMS_PORT not set in port_registry.sh"
    exit 1
fi

if [[ -z "$OPENEMS_JSONRPC_PORT" ]]; then
    echo "❌ OPENEMS_JSONRPC_PORT not set in port_registry.sh"
    exit 1
fi

if [[ -z "$OPENEMS_MODBUS_PORT" ]]; then
    echo "❌ OPENEMS_MODBUS_PORT not set in port_registry.sh"
    exit 1
fi

if [[ -z "$OPENEMS_BACKEND_PORT" ]]; then
    echo "❌ OPENEMS_BACKEND_PORT not set in port_registry.sh"
    exit 1
fi

# Docker configuration
export OPENEMS_IMAGE="${OPENEMS_IMAGE:-openems/edge:latest}"
export OPENEMS_BACKEND_IMAGE="${OPENEMS_BACKEND_IMAGE:-openems/backend:latest}"
export OPENEMS_CONTAINER="${OPENEMS_CONTAINER:-openems-edge}"
export OPENEMS_BACKEND_CONTAINER="${OPENEMS_BACKEND_CONTAINER:-openems-backend}"

# Memory limits
export OPENEMS_MEMORY="${OPENEMS_MEMORY:-512m}"
export OPENEMS_BACKEND_MEMORY="${OPENEMS_BACKEND_MEMORY:-512m}"

# Telemetry configuration
export OPENEMS_TELEMETRY_INTERVAL="${OPENEMS_TELEMETRY_INTERVAL:-1000}"
export OPENEMS_TELEMETRY_BATCH_SIZE="${OPENEMS_TELEMETRY_BATCH_SIZE:-100}"

# QuestDB integration
export QUESTDB_HOST="${QUESTDB_HOST:-localhost}"
export QUESTDB_PORT="${QUESTDB_PORT:-9010}"

# Redis integration
export REDIS_HOST="${REDIS_HOST:-localhost}"
export REDIS_PORT="${REDIS_PORT:-6379}"

# Timeouts
export OPENEMS_STARTUP_TIMEOUT="${OPENEMS_STARTUP_TIMEOUT:-60}"
export OPENEMS_SHUTDOWN_TIMEOUT="${OPENEMS_SHUTDOWN_TIMEOUT:-30}"
export OPENEMS_HEALTH_TIMEOUT="${OPENEMS_HEALTH_TIMEOUT:-5}"

# Logging
export OPENEMS_LOG_LEVEL="${OPENEMS_LOG_LEVEL:-INFO}"
export OPENEMS_LOG_MAX_SIZE="${OPENEMS_LOG_MAX_SIZE:-100m}"

# DER simulation defaults
export OPENEMS_SIMULATION_ENABLED="${OPENEMS_SIMULATION_ENABLED:-true}"
export OPENEMS_DEFAULT_SOLAR_CAPACITY="${OPENEMS_DEFAULT_SOLAR_CAPACITY:-10}"
export OPENEMS_DEFAULT_BATTERY_CAPACITY="${OPENEMS_DEFAULT_BATTERY_CAPACITY:-20}"

# Security
export OPENEMS_AUTH_ENABLED="${OPENEMS_AUTH_ENABLED:-true}"
export OPENEMS_DEFAULT_USER="${OPENEMS_DEFAULT_USER:-admin}"
export OPENEMS_DEFAULT_PASSWORD="${OPENEMS_DEFAULT_PASSWORD:-admin}"