#!/usr/bin/env bash
# Node-RED Installation Functions
# Install, uninstall, and configuration update functions

# Source var.sh first to get standard directory variables  
LIB_INSTALL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${LIB_INSTALL_DIR}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LIB_SERVICE_DIR}/secrets.sh"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh"

#######################################
# Pre-installation checks
#######################################
node_red::pre_install_checks() {
    # Check Docker availability
    if ! node_red::validate_docker_setup; then
        return 1
    fi
    
    # Create necessary directories
    if ! node_red::create_directories; then
        return 1
    fi
    
    return 0
}

#######################################
# Handle existing installation
#######################################
node_red::handle_existing_installation() {
    if node_red::is_installed; then
        node_red::show_already_installed_warning
        
        if [[ "$FORCE" != "yes" ]]; then
            if ! node_red::prompt_reinstall; then
                return 1  # User cancelled
            fi
        fi
        
        # Uninstall existing installation
        node_red::uninstall_without_prompts
    fi
    
    return 0
}

#######################################
# Complete Node-RED installation
#######################################
node_red::install() {
    # Check if already installed and running
    if node_red::is_installed && node_red::is_running && [[ "$FORCE" != "yes" ]]; then
        log::info "Node-RED is already installed and running"
        log::info "Use --force yes to reinstall, or --action status to check current state"
        node_red::update_resource_config  # Ensure config is up to date
        return 0
    fi
    
    node_red::show_installing
    
    # Pre-installation checks
    if ! node_red::pre_install_checks; then
        node_red::show_installation_failed_error
        return 1
    fi
    
    # Handle existing installation
    if ! node_red::handle_existing_installation; then
        return 0  # User cancelled, not an error
    fi
    
    # Start the container
    if ! node_red::start_container "$BUILD_IMAGE"; then
        node_red::show_installation_failed_error
        return 1
    fi
    
    # Wait for Node-RED to be ready
    if node_red::wait_for_ready; then
        node_red::show_installation_complete
        return 0
    else
        node_red::show_installation_failed_error
        return 1
    fi
}

#######################################
# Uninstall Node-RED with prompts
#######################################
node_red::uninstall() {
    if ! node_red::is_installed; then
        log::warning "Node-RED is not installed"
        return 0
    fi
    
    # Prompt for confirmation
    if ! node_red::prompt_uninstall; then
        return 0  # User cancelled
    fi
    
    node_red::uninstall_without_prompts
}

#######################################
# Uninstall Node-RED without prompts (internal use)
#######################################
node_red::uninstall_without_prompts() {
    node_red::show_uninstalling
    
    # Clean up all Docker resources
    node_red::cleanup_docker_resources
    
    # Remove from resource configuration
    node_red::remove_resource_config
    
    log::success "Node-RED uninstalled successfully"
}

#######################################
# Start Node-RED service
#######################################
node_red::start() {
    if ! node_red::is_installed; then
        node_red::show_not_installed_error
        return 1
    fi
    
    if node_red::is_running; then
        node_red::show_already_running_warning
        return 0
    fi
    
    node_red::start_existing_container
}

#######################################
# Stop Node-RED service
#######################################
node_red::stop() {
    node_red::stop_container
}

#######################################
# Restart Node-RED service
#######################################
node_red::restart() {
    node_red::restart_container
}

#######################################
# Post-installation setup
#######################################
node_red::post_install_setup() {
    # Update resource configuration
    node_red::update_resource_config
    
    # Import example flows if they exist
    if [[ -d "$SCRIPT_DIR/flows" ]] && ls "$SCRIPT_DIR/flows"/*.json >/dev/null 2>&1; then
        node_red::show_importing_flows
        for flow in "$SCRIPT_DIR/flows"/*.json; do
            node_red::import_flow_file "$flow" || log::warning "Failed to import $(basename "$flow")"
        done
    fi
}

#######################################
# Reinstall Node-RED (force reinstall)
#######################################
node_red::reinstall() {
    log::info "Reinstalling Node-RED..."
    
    # Force uninstall
    if node_red::is_installed; then
        node_red::uninstall_without_prompts
    fi
    
    # Install fresh
    node_red::install
}

#######################################
# Update Node-RED installation
#######################################
node_red::update() {
    if ! node_red::is_installed; then
        node_red::show_not_installed_error
        return 1
    fi
    
    log::info "Updating Node-RED..."
    
    # Pull latest official image if not using custom
    if [[ "$BUILD_IMAGE" != "yes" ]]; then
        node_red::pull_official_image
    fi
    
    # Restart with updated image
    node_red::restart_container
    
    log::success "Node-RED updated successfully"
}

#######################################
# Backup Node-RED data
#######################################
node_red::backup() {
    local backup_dir="${1:-$HOME/.node-red-backup}"
    local timestamp=$(date +%Y%m%d-%H%M%S)
    local backup_path="$backup_dir/node-red-backup-$timestamp"
    
    log::info "Creating backup at: $backup_path"
    
    # Create backup directory
    mkdir -p "$backup_path"
    
    # Backup flows
    if [[ -d "$SCRIPT_DIR/flows" ]]; then
        cp -r "$SCRIPT_DIR/flows" "$backup_path/"
    fi
    
    # Backup settings
    if [[ -f "$SCRIPT_DIR/settings.js" ]]; then
        cp "$SCRIPT_DIR/settings.js" "$backup_path/"
    fi
    
    # Backup custom nodes if any
    if [[ -d "$SCRIPT_DIR/nodes" ]] && [[ -n "$(ls -A "$SCRIPT_DIR/nodes" 2>/dev/null)" ]]; then
        cp -r "$SCRIPT_DIR/nodes" "$backup_path/"
    fi
    
    # Export flows from running container if possible
    if node_red::is_running; then
        local flows_export="$backup_path/flows-export.json"
        if node_red::export_flows_to_file "$flows_export"; then
            log::info "Exported running flows to backup"
        fi
    fi
    
    log::success "Backup created at: $backup_path"
    echo "Backup contents:"
    ls -la "$backup_path"
}

#######################################
# Restore Node-RED data from backup
#######################################
node_red::restore() {
    local backup_path="$1"
    
    if [[ -z "$backup_path" ]]; then
        log::error "Backup path is required"
        log::info "Usage: $0 --action restore --backup-path /path/to/backup"
        return 1
    fi
    
    if [[ ! -d "$backup_path" ]]; then
        log::error "Backup directory not found: $backup_path"
        return 1
    fi
    
    log::info "Restoring Node-RED from backup: $backup_path"
    
    # Stop Node-RED if running
    local was_running=false
    if node_red::is_running; then
        was_running=true
        node_red::stop_container
    fi
    
    # Restore flows
    if [[ -d "$backup_path/flows" ]]; then
        log::info "Restoring flows..."
        trash::safe_remove "$SCRIPT_DIR/flows" --no-confirm
        cp -r "$backup_path/flows" "$SCRIPT_DIR/"
    fi
    
    # Restore settings
    if [[ -f "$backup_path/settings.js" ]]; then
        log::info "Restoring settings..."
        cp "$backup_path/settings.js" "$SCRIPT_DIR/"
    fi
    
    # Restore custom nodes
    if [[ -d "$backup_path/nodes" ]]; then
        log::info "Restoring custom nodes..."
        trash::safe_remove "$SCRIPT_DIR/nodes" --no-confirm
        cp -r "$backup_path/nodes" "$SCRIPT_DIR/"
    fi
    
    # Import flows if export file exists
    if [[ -f "$backup_path/flows-export.json" ]]; then
        log::info "Importing flows from export..."
        # Start Node-RED first if it was running
        if [[ "$was_running" == "true" ]]; then
            node_red::start_existing_container
            sleep 5  # Give it time to start
        fi
        
        if node_red::is_running; then
            node_red::import_flow_file "$backup_path/flows-export.json"
        fi
    elif [[ "$was_running" == "true" ]]; then
        # Restart if it was running before
        node_red::start_existing_container
    fi
    
    log::success "Node-RED restored from backup"
}

#######################################
# Check for Node-RED updates
#######################################
node_red::check_updates() {
    if ! node_red::is_installed; then
        node_red::show_not_installed_error
        return 1
    fi
    
    log::info "Checking for Node-RED updates..."
    
    # Get current image ID
    local current_image_id
    current_image_id=$(docker inspect --format='{{.Image}}' "$CONTAINER_NAME" 2>/dev/null)
    
    # Pull latest image
    docker pull "$OFFICIAL_IMAGE" >/dev/null 2>&1
    
    # Get new image ID
    local latest_image_id
    latest_image_id=$(docker inspect --format='{{.Id}}' "$OFFICIAL_IMAGE" 2>/dev/null)
    
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
# Validate installation integrity
#######################################
node_red::validate_installation() {
    local issues=0
    
    log::info "Validating Node-RED installation..."
    
    # Check if container exists
    if ! node_red::is_installed; then
        log::error "Container does not exist"
        ((issues++))
    fi
    
    # Check if running
    if ! node_red::is_running; then
        log::warning "Container is not running"
        ((issues++))
    fi
    
    # Check if responsive
    if ! node_red::is_healthy; then
        log::error "Node-RED is not responding to HTTP requests"
        ((issues++))
    fi
    
    # Check settings file
    if [[ ! -f "$SCRIPT_DIR/settings.js" ]]; then
        log::warning "Settings file missing"
        ((issues++))
    fi
    
    # Check flows directory
    if [[ ! -d "$SCRIPT_DIR/flows" ]]; then
        log::warning "Flows directory missing"
        ((issues++))
    fi
    
    # Check resource configuration
    local config_file
    config_file="$(secrets::get_project_config_file)"
    if [[ ! -f "$config_file" ]] || ! jq -e '.services.automation."node-red"' "$config_file" >/dev/null 2>&1; then
        log::warning "Resource configuration missing or invalid"
        ((issues++))
    fi
    
    if [[ $issues -eq 0 ]]; then
        log::success "Installation validation passed"
        return 0
    else
        log::error "Found $issues issues with the installation"
        return 1
    fi
}