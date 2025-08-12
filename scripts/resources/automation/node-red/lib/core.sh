#!/usr/bin/env bash
# Node-RED Core Functions - Consolidated Node-RED-specific logic
# All generic operations delegated to shared libraries

# Source shared libraries
NODE_RED_LIB_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${NODE_RED_LIB_DIR}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/docker-utils.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/http-utils.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/status-engine.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/health-framework.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/backup-framework.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/init-framework.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/wait-utils.sh"
# shellcheck disable=SC1091
source "${var_LIB_SERVICE_DIR}/secrets.sh"

#######################################
# Node-RED Configuration Constants
# These are set in config/defaults.sh as readonly
# Only set non-readonly variables here
#######################################
# Variables that aren't set as readonly in defaults.sh
: "${NODE_RED_FLOWS_FILE:=flows.json}"
: "${NODE_RED_CREDENTIALS_FILE:=flows_cred.json}"
: "${NODE_RED_SETTINGS_FILE:=settings.js}"
: "${BASIC_AUTH:=yes}"
: "${AUTH_USERNAME:=admin}"
: "${BUILD_IMAGE:=no}"

#######################################
# Get Node-RED initialization configuration
# Returns: JSON configuration for init framework
#######################################
node_red::get_init_config() {
    local auth_password="${1:-}"
    
    # Select image (custom if available)
    local image_to_use="$NODE_RED_IMAGE"
    if [[ "$BUILD_IMAGE" == "yes" ]] && docker images --format "{{.Repository}}:{{.Tag}}" | grep -q "^${NODE_RED_CUSTOM_IMAGE}$"; then
        image_to_use="$NODE_RED_CUSTOM_IMAGE"
    fi
    
    # Build environment variables
    local timezone
    timezone=$(timedatectl show -p Timezone --value 2>/dev/null || echo 'UTC')
    
    # Build volumes list
    local volumes_array="[\"${NODE_RED_DATA_DIR}:/data\""
    if [[ "$BUILD_IMAGE" == "yes" ]]; then
        volumes_array+=",\"/var/run/docker.sock:/var/run/docker.sock:rw\""
        volumes_array+=",\"${PWD}:/workspace:rw\""
        volumes_array+=",\"$HOME:/host/home:rw\""
    fi
    volumes_array+="]"
    
    # Build init config
    local config='{
        "resource_name": "node-red",
        "container_name": "'$NODE_RED_CONTAINER_NAME'",
        "data_dir": "'$NODE_RED_DATA_DIR'",
        "port": '$NODE_RED_PORT',
        "image": "'$image_to_use'",
        "env_vars": {
            "TZ": "'$timezone'",
            "NODE_RED_ENABLE_PROJECTS": "false",
            "NODE_RED_ENABLE_SAFE_MODE": "false"
        },
        "volumes": '$volumes_array',
        "networks": ["'$NODE_RED_NETWORK_NAME'"],
        "first_run_check": "node_red::is_first_run",
        "setup_func": "node_red::first_time_setup",
        "wait_for_ready": "node_red::wait_for_ready"
    }'
    
    # Add authentication if enabled
    if [[ "$BASIC_AUTH" == "yes" && -n "$auth_password" ]]; then
        config=$(echo "$config" | jq --arg username "$AUTH_USERNAME" --arg password "$auth_password" '
            .env_vars += {
                "NODE_RED_CREDENTIAL_SECRET": $password
            }
        ')
    fi
    
    echo "$config"
}

#######################################
# Check if this is first run (no flows exist)
# Returns: 0 if first run, 1 otherwise
#######################################
node_red::is_first_run() {
    [[ ! -f "$NODE_RED_DATA_DIR/$NODE_RED_FLOWS_FILE" ]]
}

#######################################
# First-time setup for Node-RED
# Sets up initial flows and configuration
#######################################
node_red::first_time_setup() {
    log::info "Setting up Node-RED for first run..."
    
    # Create data directory
    mkdir -p "$NODE_RED_DATA_DIR"
    
    # Copy default settings if it doesn't exist
    if [[ ! -f "$NODE_RED_DATA_DIR/$NODE_RED_SETTINGS_FILE" ]]; then
        if [[ -f "${NODE_RED_SCRIPT_DIR}/config/settings.js" ]]; then
            cp "${NODE_RED_SCRIPT_DIR}/config/settings.js" "$NODE_RED_DATA_DIR/$NODE_RED_SETTINGS_FILE"
        fi
    fi
    
    # Copy default flows if available
    local default_flows="${NODE_RED_SCRIPT_DIR}/examples/default-flows.json"
    if [[ -f "$default_flows" && ! -f "$NODE_RED_DATA_DIR/$NODE_RED_FLOWS_FILE" ]]; then
        cp "$default_flows" "$NODE_RED_DATA_DIR/$NODE_RED_FLOWS_FILE"
        log::info "Installed default flows"
    fi
    
    # Set appropriate permissions
    chown -R 1000:1000 "$NODE_RED_DATA_DIR" 2>/dev/null || true
}

#######################################
# Wait for Node-RED to be ready
# Returns: 0 if ready, 1 if timeout
#######################################
node_red::wait_for_ready() {
    local max_attempts=30
    local attempt=1
    
    log::info "Waiting for Node-RED to be ready..."
    
    while [[ $attempt -le $max_attempts ]]; do
        if node_red::is_responding; then
            log::success "Node-RED is ready!"
            return 0
        fi
        
        log::info "Attempt $attempt/$max_attempts: Node-RED not ready yet..."
        sleep 2
        ((attempt++))
    done
    
    log::error "Node-RED failed to become ready within timeout"
    return 1
}

#######################################
# Check if Node-RED is responding to requests
# Returns: 0 if responding, 1 otherwise
#######################################
node_red::is_responding() {
    local url="http://localhost:$NODE_RED_PORT"
    
    # Try to get the Node-RED editor page
    if curl -s -f "$url" >/dev/null 2>&1; then
        return 0
    fi
    
    return 1
}

#######################################
# Install Node-RED using init framework
# Globals: All Node-RED configuration variables
# Returns: 0 on success, 1 on failure
#######################################
node_red::install() {
    log::info "Installing Node-RED using init framework..."
    
    if ! docker::check_daemon; then
        return 1
    fi
    
    # Get authentication password if basic auth is enabled
    local auth_password=""
    if [[ "$BASIC_AUTH" == "yes" ]]; then
        auth_password=$(openssl rand -hex 8)
        log::info "Generated authentication password for admin user"
    fi
    
    local init_config
    init_config=$(node_red::get_init_config "$auth_password")
    
    if init::setup_resource "$init_config"; then
        if [[ "$BASIC_AUTH" == "yes" ]]; then
            log::success "Node-RED installed successfully!"
            log::info "Access URL: http://localhost:$NODE_RED_PORT"
            log::info "Username: $AUTH_USERNAME"
            log::info "Password: $auth_password"
            log::warn "Save these credentials - they won't be shown again"
        else
            log::success "Node-RED installed successfully!"
            log::info "Access URL: http://localhost:$NODE_RED_PORT"
            log::warn "No authentication configured - consider enabling it for security"
        fi
        return 0
    else
        log::error "Node-RED installation failed"
        return 1
    fi
}

#######################################
# Uninstall Node-RED
# Returns: 0 on success, 1 on failure
#######################################
node_red::uninstall() {
    log::info "Uninstalling Node-RED..."
    
    if ! docker::check_daemon; then
        return 1
    fi
    
    # Stop and remove container
    if docker::container_exists "$NODE_RED_CONTAINER_NAME"; then
        docker::stop_container "$NODE_RED_CONTAINER_NAME"
        docker::remove_container "$NODE_RED_CONTAINER_NAME"
    fi
    
    # Remove network if it exists and is empty
    docker::remove_network_if_empty "$NODE_RED_NETWORK_NAME"
    
    log::success "Node-RED uninstalled successfully"
    return 0
}

#######################################
# Start Node-RED service
# Returns: 0 on success, 1 on failure
#######################################
node_red::start() {
    if ! docker::check_daemon; then
        return 1
    fi
    
    if ! docker::container_exists "$NODE_RED_CONTAINER_NAME"; then
        log::error "Node-RED is not installed. Run: $0 --action install"
        return 1
    fi
    
    if docker::container_running "$NODE_RED_CONTAINER_NAME"; then
        log::info "Node-RED is already running"
        return 0
    fi
    
    log::info "Starting Node-RED..."
    docker::start_container "$NODE_RED_CONTAINER_NAME"
    
    if node_red::wait_for_ready; then
        log::success "Node-RED started successfully"
        log::info "Access URL: http://localhost:$NODE_RED_PORT"
        return 0
    else
        log::error "Node-RED failed to start properly"
        return 1
    fi
}

#######################################
# Stop Node-RED service
# Returns: 0 on success, 1 on failure
#######################################
node_red::stop() {
    if ! docker::check_daemon; then
        return 1
    fi
    
    if ! docker::container_exists "$NODE_RED_CONTAINER_NAME"; then
        log::info "Node-RED is not installed"
        return 0
    fi
    
    if ! docker::container_running "$NODE_RED_CONTAINER_NAME"; then
        log::info "Node-RED is already stopped"
        return 0
    fi
    
    log::info "Stopping Node-RED..."
    docker::stop_container "$NODE_RED_CONTAINER_NAME"
    log::success "Node-RED stopped successfully"
    return 0
}

#######################################
# Restart Node-RED service
# Returns: 0 on success, 1 on failure
#######################################
node_red::restart() {
    log::info "Restarting Node-RED..."
    node_red::stop
    sleep 2
    node_red::start
}

#######################################
# Show Node-RED logs
# Arguments:
#   $1 - lines to show (default: 100)
#   $2 - follow logs (yes/no, default: no)
# Returns: 0 on success, 1 on failure
#######################################
node_red::view_logs() {
    local lines="${1:-100}"
    local follow="${2:-no}"
    
    if ! docker::check_daemon; then
        return 1
    fi
    
    if ! docker::container_exists "$NODE_RED_CONTAINER_NAME"; then
        log::error "Node-RED is not installed"
        return 1
    fi
    
    local args="--tail $lines"
    if [[ "$follow" == "yes" ]]; then
        args+=" --follow"
    fi
    
    # shellcheck disable=SC2086
    docker logs $args "$NODE_RED_CONTAINER_NAME"
}

#######################################
# Get Node-RED status information for status engine
# Returns: JSON configuration for status display
#######################################
node_red::get_status_config() {
    echo '{
        "resource": {
            "name": "node-red",
            "category": "automation",
            "description": "Flow-based programming for event-driven applications",
            "port": '$NODE_RED_PORT',
            "container_name": "'$NODE_RED_CONTAINER_NAME'",
            "data_dir": "'$NODE_RED_DATA_DIR'"
        },
        "endpoints": {
            "ui": "'$NODE_RED_BASE_URL'",
            "api": "'$NODE_RED_BASE_URL'/flows",
            "health": "'$NODE_RED_BASE_URL'"
        },
        "health_tiers": {
            "healthy": "All systems operational",
            "degraded": "Node-RED responding but flows may need configuration",
            "unhealthy": "Service not responding - Try: ./manage.sh --action restart"
        }
    }'
}

#######################################
# Node-RED Utility Functions
# Consolidated from lib/common.sh
#######################################

#######################################
# Generate secure random secret
# Returns: 64-character hexadecimal secret
#######################################
node_red::generate_secret() {
    if command -v openssl >/dev/null 2>&1; then
        openssl rand -hex 32 2>/dev/null
    elif [[ -r /dev/urandom ]]; then
        tr -dc 'A-Za-z0-9' < /dev/urandom | head -c 64
    else
        # Fallback to timestamp-based secret
        echo "node-red-$(date +%s)-$RANDOM" | sha256sum | cut -c1-64
    fi
}

#######################################
# Create default settings.js if it doesn't exist
#######################################
node_red::create_default_settings() {
    local settings_file="${NODE_RED_SCRIPT_DIR}/config/settings.js"
    
    if [[ -f "$settings_file" ]]; then
        return 0  # Settings already exist
    fi
    
    log::info "Creating default Node-RED settings..."
    
    cat > "$settings_file" << 'EOF'
module.exports = {
    // Flow file settings
    flowFile: 'flows.json',
    flowFilePretty: true,
    
    // User directory
    userDir: '/data/',
    
    // Node-RED settings
    uiPort: process.env.PORT || 1880,
    
    // Logging
    logging: {
        console: {
            level: "info",
            metrics: false,
            audit: false
        }
    },
    
    // Editor theme
    editorTheme: {
        theme: "dark",
        projects: {
            enabled: false
        }
    },
    
    // Function node settings
    functionGlobalContext: {
        // Add global libraries here
        // os: require('os'),
    },
    
    // Allow external npm modules in function nodes
    functionExternalModules: true,
    
    // Debug settings
    debugMaxLength: 1000,
    
    // Exec node settings
    execMaxBufferSize: 10000000, // 10MB
    
    // HTTP request timeout
    httpRequestTimeout: 120000, // 2 minutes
}
EOF
}

#######################################
# Update Vrooli resource configuration
# SAFE VERSION - Uses proper ConfigurationManager to prevent data loss
#######################################
node_red::update_resource_config() {
    log::info "Updating Node-RED resource configuration..."
    
    # Source the common resources functions for safe config management
    if [[ -f "${var_SCRIPTS_RESOURCES_DIR}/common.sh" ]]; then
        # shellcheck disable=SC1091
        source "${var_SCRIPTS_RESOURCES_DIR}/common.sh"
    else
        log::error "Cannot find resources common.sh script for safe configuration management"
        return 1
    fi
    
    # Create Node-RED specific configuration
    local node_red_config=$(cat << EOF
{
    "enabled": true,
    "baseUrl": "http://localhost:$NODE_RED_PORT",
    "adminUrl": "http://localhost:$NODE_RED_PORT/admin",
    "healthCheck": {
        "endpoint": "/flows",
        "intervalMs": 60000,
        "timeoutMs": 5000
    },
    "flows": {
        "directory": "/data/flows",
        "autoBackup": true,
        "backupInterval": "1h"
    }
}
EOF
    )
    
    # Use the safe configuration update function from common.sh
    if resources::update_config "automation" "node-red" "http://localhost:$NODE_RED_PORT" "$node_red_config"; then
        log::success "Node-RED configuration updated safely"
        return 0
    else
        log::error "Failed to update Node-RED configuration using safe method"
        return 1
    fi
}

#######################################
# Validate JSON file
# Arguments: file path
# Returns: 0 if valid, 1 if invalid
#######################################
node_red::validate_json() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    jq . "$file" >/dev/null 2>&1
}

#######################################
# Update Node-RED to latest version
# Returns: 0 on success, 1 on failure
#######################################
node_red::update() {
    if ! docker::container_exists "$NODE_RED_CONTAINER_NAME"; then
        log::error "Node-RED is not installed. Run: $0 --action install"
        return 1
    fi
    
    log::info "Updating Node-RED to latest version..."
    
    # Pull latest official image if not using custom
    if [[ "$BUILD_IMAGE" != "yes" ]]; then
        log::info "Pulling latest Node-RED image..."
        docker::pull_image "$NODE_RED_IMAGE"
    fi
    
    # Restart with updated image
    log::info "Restarting Node-RED with updated image..."
    node_red::restart
    
    log::success "Node-RED updated successfully"
    return 0
}

#######################################
# Check for Node-RED updates
# Returns: 0 if updates available, 1 if up to date
#######################################
node_red::check_updates() {
    if ! docker::container_exists "$NODE_RED_CONTAINER_NAME"; then
        log::error "Node-RED is not installed"
        return 1
    fi
    
    log::info "Checking for Node-RED updates..."
    
    # Get current image ID
    local current_image_id
    current_image_id=$(docker inspect --format='{{.Image}}' "$NODE_RED_CONTAINER_NAME" 2>/dev/null)
    
    # Pull latest image quietly
    docker pull "$NODE_RED_IMAGE" >/dev/null 2>&1
    
    # Get new image ID
    local latest_image_id
    latest_image_id=$(docker inspect --format='{{.Id}}' "$NODE_RED_IMAGE" 2>/dev/null)
    
    if [[ "$current_image_id" != "$latest_image_id" ]]; then
        log::info "Updates available for Node-RED"
        echo "Current image: ${current_image_id:7:12}"
        echo "Latest image:  ${latest_image_id:7:12}"
        echo "Run '$0 --action update' to update"
        return 0
    else
        log::success "Node-RED is up to date"
        return 1
    fi
}

#######################################
# Validate Node-RED installation
# Returns: 0 if valid, 1 if issues found
#######################################
node_red::validate_installation() {
    local issues=0
    
    log::info "Validating Node-RED installation..."
    
    # Check if container exists
    if ! docker::container_exists "$NODE_RED_CONTAINER_NAME"; then
        log::error "Container does not exist"
        ((issues++))
    fi
    
    # Check if running
    if ! docker::container_running "$NODE_RED_CONTAINER_NAME"; then
        log::warning "Container is not running"
        ((issues++))
    fi
    
    # Check if responsive
    if ! node_red::is_responding; then
        log::error "Node-RED is not responding to HTTP requests"
        ((issues++))
    fi
    
    # Check data directory
    if [[ ! -d "$NODE_RED_DATA_DIR" ]]; then
        log::warning "Data directory missing: $NODE_RED_DATA_DIR"
        ((issues++))
    fi
    
    # Check if flows file exists
    if [[ ! -f "$NODE_RED_DATA_DIR/$NODE_RED_FLOWS_FILE" ]]; then
        log::info "No flows file found (this is normal for new installations)"
    fi
    
    if [[ $issues -eq 0 ]]; then
        log::success "Installation validation passed"
        return 0
    else
        log::error "Found $issues issues with the installation"
        return 1
    fi
}