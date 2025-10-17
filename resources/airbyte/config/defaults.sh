#!/bin/bash
# Airbyte Default Configuration

# Version
export AIRBYTE_VERSION="${AIRBYTE_VERSION:-latest}"

# Load ports from registry
PORT_REGISTRY="${VROOLI_ROOT:-${HOME}/Vrooli}/scripts/resources/port_registry.sh"
if [[ -f "$PORT_REGISTRY" ]]; then
    # Set flag to prevent port_registry from executing its CLI logic
    export SOURCED_PORT_REGISTRY=1
    source "$PORT_REGISTRY"
    export AIRBYTE_WEBAPP_PORT="${RESOURCE_PORTS[airbyte]:-8002}"
    export AIRBYTE_SERVER_PORT="${RESOURCE_PORTS[airbyte-server]:-8003}"
    export AIRBYTE_TEMPORAL_PORT="${RESOURCE_PORTS[airbyte-temporal]:-8006}"
else
    # Fallback to registry-defined ports if file not found
    export AIRBYTE_WEBAPP_PORT="${AIRBYTE_WEBAPP_PORT:-8002}"
    export AIRBYTE_SERVER_PORT="${AIRBYTE_SERVER_PORT:-8003}"
    export AIRBYTE_TEMPORAL_PORT="${AIRBYTE_TEMPORAL_PORT:-8006}"
fi

# Directories
export AIRBYTE_DATA_DIR="${AIRBYTE_DATA_DIR:-${RESOURCE_DIR}/data}"
export AIRBYTE_WORKSPACE_DIR="${AIRBYTE_WORKSPACE_DIR:-${AIRBYTE_DATA_DIR}/workspace}"

# Timeouts
export AIRBYTE_STARTUP_TIMEOUT="${AIRBYTE_STARTUP_TIMEOUT:-120}"
export AIRBYTE_HEALTH_CHECK_TIMEOUT="${AIRBYTE_HEALTH_CHECK_TIMEOUT:-5}"

# Resource limits
export AIRBYTE_MEMORY_LIMIT="${AIRBYTE_MEMORY_LIMIT:-4g}"
export AIRBYTE_CPU_LIMIT="${AIRBYTE_CPU_LIMIT:-2}"

# Debug mode
export DEBUG="${DEBUG:-false}"

# Deployment method
# Airbyte v1.x+ requires abctl (Kubernetes-in-Docker) deployment
# docker-compose was deprecated in August 2024
export AIRBYTE_USE_ABCTL="${AIRBYTE_USE_ABCTL:-true}"