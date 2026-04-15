#!/usr/bin/env bash
# Neo4j Configuration Defaults

set +u  # Temporarily disable unbound variable check for environment testing

# Set APP_ROOT if not already set
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"

# Service configuration
export NEO4J_NAME="neo4j"
export NEO4J_DISPLAY_NAME="Neo4j Graph Database"
export NEO4J_CATEGORY="storage"
export NEO4J_DESCRIPTION="Native property graph database with Cypher query language"

# Container configuration
export NEO4J_CONTAINER_NAME="${NEO4J_CONTAINER_NAME:-vrooli-neo4j}"
export NEO4J_VERSION="${NEO4J_VERSION:-5.15.0}"
export NEO4J_IMAGE="${NEO4J_IMAGE:-neo4j:${NEO4J_VERSION}}"

# Port configuration - use standardized port registry access
# Source the port registry to get RESOURCE_PORTS array
if [[ -f "${APP_ROOT}/scripts/resources/port_registry.sh" ]]; then
    source "${APP_ROOT}/scripts/resources/port_registry.sh"
fi

# Get ports from registry if not already set
if [[ -z "${NEO4J_HTTP_PORT}" ]]; then
    NEO4J_HTTP_PORT="${RESOURCE_PORTS["neo4j"]}"
fi

if [[ -z "${NEO4J_BOLT_PORT}" ]]; then
    NEO4J_BOLT_PORT="${RESOURCE_PORTS["neo4j-bolt"]}"
fi

# Fail if ports not available
if [[ -z "${NEO4J_HTTP_PORT}" ]]; then
    echo "ERROR: NEO4J_HTTP_PORT not set and not available from port registry" >&2
    exit 1
fi
if [[ -z "${NEO4J_BOLT_PORT}" ]]; then
    echo "ERROR: NEO4J_BOLT_PORT not set and not available from port registry" >&2
    exit 1
fi

export NEO4J_HTTP_PORT
export NEO4J_BOLT_PORT

# Authentication - use environment variable, no hardcoded defaults
# Note: Neo4j Community Edition v5 has authentication issues with NEO4J_AUTH env var
# For development, we default to no auth. For production, set NEO4J_AUTH explicitly
if [[ -z "${NEO4J_AUTH}" ]]; then
    # Default to no authentication for development
    # Set NEO4J_AUTH="neo4j/yourpassword" for production
    export NEO4J_AUTH="none"
else
    export NEO4J_AUTH
fi

# Memory configuration
export NEO4J_HEAP_SIZE="${NEO4J_HEAP_SIZE:-512M}"

# Canonical resource storage directories.
# RESOURCE_* is injected by the Go control plane; XDG fallbacks keep shell use off repo-local paths.
neo4j_xdg_config_home="${XDG_CONFIG_HOME:-${HOME}/.config}"
neo4j_xdg_data_home="${XDG_DATA_HOME:-${HOME}/.local/share}"
neo4j_xdg_state_home="${XDG_STATE_HOME:-${HOME}/.local/state}"
neo4j_base_data_dir="${RESOURCE_DATA_DIR:-${neo4j_xdg_data_home}/vrooli/resources/neo4j}"
neo4j_base_logs_dir="${RESOURCE_LOGS_DIR:-${neo4j_xdg_state_home}/logs/vrooli/resources/neo4j}"
neo4j_base_config_dir="${RESOURCE_CONFIG_DIR:-${neo4j_xdg_config_home}/vrooli/resources/neo4j}"
neo4j_base_state_dir="${RESOURCE_STATE_DIR:-${neo4j_xdg_state_home}/vrooli/resources/neo4j}"

export NEO4J_DATA_DIR="${NEO4J_DATA_DIR:-${neo4j_base_data_dir}/data}"
export NEO4J_LOGS_DIR="${NEO4J_LOGS_DIR:-${neo4j_base_logs_dir}}"
export NEO4J_IMPORT_DIR="${NEO4J_IMPORT_DIR:-${neo4j_base_data_dir}/import}"
export NEO4J_PLUGINS_DIR="${NEO4J_PLUGINS_DIR:-${neo4j_base_data_dir}/plugins}"
export NEO4J_CONF_DIR="${NEO4J_CONF_DIR:-${neo4j_base_config_dir}}"
export NEO4J_STATE_DIR="${NEO4J_STATE_DIR:-${neo4j_base_state_dir}}"

# Re-enable strict mode after environment checks
set -u

# Export config function
neo4j::export_config() {
    export NEO4J_NAME NEO4J_DISPLAY_NAME NEO4J_CATEGORY NEO4J_DESCRIPTION
    export NEO4J_CONTAINER_NAME NEO4J_VERSION NEO4J_IMAGE
    export NEO4J_HTTP_PORT NEO4J_BOLT_PORT NEO4J_AUTH NEO4J_HEAP_SIZE
    export NEO4J_DATA_DIR NEO4J_LOGS_DIR NEO4J_IMPORT_DIR NEO4J_PLUGINS_DIR NEO4J_CONF_DIR NEO4J_STATE_DIR
}
