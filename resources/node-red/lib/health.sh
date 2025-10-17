#!/usr/bin/env bash
# Node-RED Health Check Functions
# Delegates to health framework for all checks

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
NODE_RED_HEALTH_LIB_DIR="${APP_ROOT}/resources/node-red/lib"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/health-framework.sh"

#######################################
# Get Node-RED health check configuration
# Returns: JSON configuration for health framework
#######################################
node_red::get_health_config() {
    cat <<EOF
{
    "service_name": "Node-RED",
    "container_name": "$NODE_RED_CONTAINER_NAME",
    "port": $NODE_RED_PORT,
    "endpoints": {
        "main": "http://localhost:$NODE_RED_PORT",
        "flows": "http://localhost:$NODE_RED_PORT/flows",
        "settings": "http://localhost:$NODE_RED_PORT/settings"
    },
    "checks": {
        "basic": {
            "container_running": true,
            "port_accessible": true,
            "http_endpoint": "/"
        },
        "advanced": {
            "api_endpoints": ["/flows", "/settings"],
            "data_directory": "$NODE_RED_DATA_DIR",
            "required_files": ["$NODE_RED_DATA_DIR/$NODE_RED_FLOWS_FILE"]
        }
    },
    "thresholds": {
        "response_time_ms": 5000,
        "memory_limit_mb": 512
    }
}
EOF
}

#######################################
# Perform health check using framework
# Returns: 0 if healthy, 1 otherwise
#######################################
node_red::health() {
    local config
    config=$(node_red::get_health_config)
    
    # Let the framework handle all the actual checking
    health::tiered_check "$config"
}