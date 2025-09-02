#!/bin/bash
# Home Assistant Installation Functions

# Define directory using cached APP_ROOT
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
HOME_ASSISTANT_INSTALL_DIR="${APP_ROOT}/resources/home-assistant/lib"

# Source dependencies
source "${HOME_ASSISTANT_INSTALL_DIR}/core.sh"
source "${HOME_ASSISTANT_INSTALL_DIR}/health.sh"

#######################################
# Install Home Assistant
# Returns: 0 on success, 1 on failure
#######################################
home_assistant::install() {
    log::header "Installing Home Assistant"
    
    # Initialize environment
    home_assistant::init
    
    # Check if already installed
    if docker::container_exists "$HOME_ASSISTANT_CONTAINER_NAME"; then
        log::warning "Home Assistant container already exists"
        
        # Start if not running
        if ! docker::is_running "$HOME_ASSISTANT_CONTAINER_NAME"; then
            log::info "Starting existing Home Assistant container..."
            docker start "$HOME_ASSISTANT_CONTAINER_NAME"
        fi
        
        # Wait for healthy state
        if home_assistant::health::wait_for_healthy "$HOME_ASSISTANT_HEALTH_CHECK_TIMEOUT"; then
            log::success "Home Assistant is installed and healthy"
            return 0
        else
            log::error "Home Assistant is installed but not healthy"
            return 1
        fi
    fi
    
    # Pull the image
    log::info "Pulling Home Assistant image: $HOME_ASSISTANT_IMAGE"
    if ! docker pull "$HOME_ASSISTANT_IMAGE"; then
        log::error "Failed to pull Home Assistant image"
        return 1
    fi
    
    # Register port
    local port_registry="${var_SCRIPTS_RESOURCES_DIR}/port_registry.sh"
    if [[ -f "$port_registry" ]]; then
        log::info "Registering port $HOME_ASSISTANT_PORT for Home Assistant"
        "$port_registry" register home-assistant "$HOME_ASSISTANT_PORT"
    fi
    
    # Create container
    log::info "Creating Home Assistant container..."
    local docker_cmd="docker run -d"
    docker_cmd+=" --name $HOME_ASSISTANT_CONTAINER_NAME"
    docker_cmd+=" --restart=$HOME_ASSISTANT_RESTART_POLICY"
    docker_cmd+=" -v $HOME_ASSISTANT_CONFIG_DIR:/config"
    docker_cmd+=" -v /etc/localtime:/etc/localtime:ro"
    docker_cmd+=" -p ${HOME_ASSISTANT_PORT}:8123"
    docker_cmd+=" -e TZ=$HOME_ASSISTANT_TIME_ZONE"
    docker_cmd+=" --privileged"  # Required for certain integrations
    docker_cmd+=" --network host"  # Recommended for discovery features
    docker_cmd+=" $HOME_ASSISTANT_IMAGE"
    
    if ! eval "$docker_cmd"; then
        log::error "Failed to create Home Assistant container"
        return 1
    fi
    
    # Wait for healthy state
    if home_assistant::health::wait_for_healthy "$HOME_ASSISTANT_HEALTH_CHECK_TIMEOUT"; then
        log::success "Home Assistant installed successfully"
        log::info "Access Home Assistant at: $HOME_ASSISTANT_BASE_URL"
        
        # Register with Vrooli CLI
        local install_cli_script="${var_SCRIPTS_RESOURCES_LIB_DIR}/install-resource-cli.sh"
        if [[ -f "$install_cli_script" ]]; then
            log::info "Registering Home Assistant CLI..."
            bash "$install_cli_script" "home-assistant" "${HOME_ASSISTANT_INSTALL_DIR}/../cli.sh"
        fi
        
        return 0
    else
        log::error "Home Assistant installed but not healthy"
        return 1
    fi
}

#######################################
# Uninstall Home Assistant
# Arguments:
#   --force: Force uninstall without confirmation
# Returns: 0 on success, 1 on failure
#######################################
home_assistant::uninstall() {
    local force="false"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --force)
                force="true"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ "$force" != "true" ]]; then
        log::error "Uninstall requires --force flag for safety"
        return 1
    fi
    
    log::header "Uninstalling Home Assistant"
    
    # Stop and remove container
    if docker::container_exists "$HOME_ASSISTANT_CONTAINER_NAME"; then
        log::info "Stopping Home Assistant container..."
        docker stop "$HOME_ASSISTANT_CONTAINER_NAME" >/dev/null 2>&1
        
        log::info "Removing Home Assistant container..."
        docker rm "$HOME_ASSISTANT_CONTAINER_NAME" >/dev/null 2>&1
    fi
    
    # Unregister port
    local port_registry="${var_SCRIPTS_RESOURCES_DIR}/port_registry.sh"
    if [[ -f "$port_registry" ]]; then
        log::info "Unregistering port for Home Assistant"
        "$port_registry" unregister home-assistant
    fi
    
    # Note: We don't remove data directories by default
    log::warning "Data directories preserved at: $HOME_ASSISTANT_DATA_DIR"
    log::info "To remove data, manually delete: rm -rf $HOME_ASSISTANT_DATA_DIR"
    
    log::success "Home Assistant uninstalled"
    return 0
}

# Export functions
export -f home_assistant::install
export -f home_assistant::uninstall