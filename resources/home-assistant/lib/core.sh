#!/bin/bash
# Home Assistant Core Functions

# Define directory using cached APP_ROOT
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
HOME_ASSISTANT_CORE_DIR="${APP_ROOT}/resources/home-assistant/lib"

# Source dependencies
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"
source "${var_LIB_UTILS_DIR}/format.sh"
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/docker-utils.sh"
source "${APP_ROOT}/resources/home-assistant/config/defaults.sh"

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

#######################################
# Docker wrapper functions for v2.0 CLI
#######################################
home_assistant::docker::start() {
    home_assistant::init
    if docker::container_exists "$HOME_ASSISTANT_CONTAINER_NAME"; then
        log::info "Starting Home Assistant..."
        docker start "$HOME_ASSISTANT_CONTAINER_NAME"
        home_assistant::health::wait_for_healthy 60
    else
        log::error "Home Assistant is not installed. Run 'manage install' first."
        return 1
    fi
}

home_assistant::docker::stop() {
    home_assistant::init
    if docker::is_running "$HOME_ASSISTANT_CONTAINER_NAME"; then
        log::info "Stopping Home Assistant..."
        docker stop "$HOME_ASSISTANT_CONTAINER_NAME"
        log::success "Home Assistant stopped"
    else
        log::warning "Home Assistant is not running"
    fi
}

home_assistant::docker::restart() {
    home_assistant::init
    if docker::container_exists "$HOME_ASSISTANT_CONTAINER_NAME"; then
        log::info "Restarting Home Assistant..."
        docker restart "$HOME_ASSISTANT_CONTAINER_NAME"
        home_assistant::health::wait_for_healthy 60
    else
        log::error "Home Assistant is not installed. Run 'manage install' first."
        return 1
    fi
}

home_assistant::docker::logs() {
    home_assistant::init
    local tail_lines="${1:-50}"
    
    # Parse options
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --tail)
                tail_lines="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if docker::container_exists "$HOME_ASSISTANT_CONTAINER_NAME"; then
        docker logs "$HOME_ASSISTANT_CONTAINER_NAME" --tail "$tail_lines"
    else
        log::error "Home Assistant is not installed"
        return 1
    fi
}

# Export functions
export -f home_assistant::export_config
export -f home_assistant::init
export -f home_assistant::get_port
export -f home_assistant::get_api_info
export -f home_assistant::docker::start
export -f home_assistant::docker::stop
export -f home_assistant::docker::restart
export -f home_assistant::docker::logs