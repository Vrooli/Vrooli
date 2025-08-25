#!/bin/bash
# Home Assistant Core Functions

# Define directory using cached APP_ROOT
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
HOME_ASSISTANT_CORE_DIR="${APP_ROOT}/resources/home-assistant/lib"

# Source dependencies
source "${HOME_ASSISTANT_CORE_DIR}/../../../../lib/utils/var.sh"
source "${var_LOG_FILE}"
source "${var_LIB_UTILS_DIR}/format.sh"
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/docker-utils.sh"
source "${HOME_ASSISTANT_CORE_DIR}/../config/defaults.sh"

#######################################
# Export Home Assistant configuration
#######################################
home_assistant::export_config() {
    # Export all configuration variables
    export HOME_ASSISTANT_CONTAINER_NAME
    export HOME_ASSISTANT_IMAGE
    export HOME_ASSISTANT_PORT
    export HOME_ASSISTANT_BASE_URL
    export HOME_ASSISTANT_DATA_DIR
    export HOME_ASSISTANT_CONFIG_DIR
    export HOME_ASSISTANT_TIME_ZONE
    export HOME_ASSISTANT_RESTART_POLICY
}

#######################################
# Initialize Home Assistant environment
#######################################
home_assistant::init() {
    home_assistant::export_config
    
    # Create data directories if they don't exist
    mkdir -p "$HOME_ASSISTANT_CONFIG_DIR"
}

#######################################
# Get Home Assistant port from port registry
#######################################
home_assistant::get_port() {
    local port_registry="${var_SCRIPTS_RESOURCES_DIR}/port-registry.sh"
    
    if [[ -f "$port_registry" ]]; then
        local registered_port
        registered_port=$("$port_registry" get home-assistant 2>/dev/null)
        
        if [[ -n "$registered_port" ]]; then
            echo "$registered_port"
        else
            # Register the default port if not registered
            "$port_registry" register home-assistant "$HOME_ASSISTANT_PORT" >/dev/null 2>&1
            echo "$HOME_ASSISTANT_PORT"
        fi
    else
        echo "$HOME_ASSISTANT_PORT"
    fi
}

#######################################
# Get API information
#######################################
home_assistant::get_api_info() {
    echo "{
        \"url\": \"$HOME_ASSISTANT_BASE_URL\",
        \"port\": \"$HOME_ASSISTANT_PORT\",
        \"health_endpoint\": \"$HOME_ASSISTANT_BASE_URL/api/\",
        \"auth_endpoint\": \"$HOME_ASSISTANT_BASE_URL/auth/token\"
    }"
}

# Export functions
export -f home_assistant::export_config
export -f home_assistant::init
export -f home_assistant::get_port
export -f home_assistant::get_api_info