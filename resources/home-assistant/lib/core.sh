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
    # For now, just return the configured port
    # The port registry integration needs to be fixed separately
    echo "$HOME_ASSISTANT_PORT"
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

#######################################
# Backup Home Assistant configuration
#######################################
home_assistant::backup() {
    home_assistant::init
    local backup_dir="${HOME_ASSISTANT_DATA_DIR}/backups"
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local backup_file="${backup_dir}/backup_${timestamp}.tar.gz"
    
    # Create backup directory if it doesn't exist
    mkdir -p "$backup_dir"
    
    log::info "Creating backup of Home Assistant configuration..."
    
    # Use docker exec to create backup from inside container with proper permissions
    if docker::is_running "$HOME_ASSISTANT_CONTAINER_NAME"; then
        # Create backup inside container
        if docker exec "$HOME_ASSISTANT_CONTAINER_NAME" tar -czf "/tmp/backup_${timestamp}.tar.gz" -C /config .; then
            # Copy backup out of container
            docker cp "$HOME_ASSISTANT_CONTAINER_NAME:/tmp/backup_${timestamp}.tar.gz" "$backup_file"
            # Clean up temp file in container
            docker exec "$HOME_ASSISTANT_CONTAINER_NAME" rm -f "/tmp/backup_${timestamp}.tar.gz"
            
            log::success "Backup created: $backup_file"
            
            # Keep only last 5 backups
            local backup_count=$(ls -1 "$backup_dir"/backup_*.tar.gz 2>/dev/null | wc -l)
            if [[ $backup_count -gt 5 ]]; then
                log::info "Cleaning old backups (keeping last 5)..."
                ls -1t "$backup_dir"/backup_*.tar.gz | tail -n +6 | xargs rm -f
            fi
            
            echo "$backup_file"
            return 0
        else
            log::error "Failed to create backup inside container"
            return 1
        fi
    else
        # Fallback to direct tar with sudo if container not running
        if sudo tar -czf "$backup_file" -C "$HOME_ASSISTANT_CONFIG_DIR" . 2>/dev/null; then
            sudo chmod 644 "$backup_file" 2>/dev/null || true
            sudo chown "$USER:$USER" "$backup_file" 2>/dev/null || true
            log::success "Backup created: $backup_file"
            
            # Keep only last 5 backups
            local backup_count=$(ls -1 "$backup_dir"/backup_*.tar.gz 2>/dev/null | wc -l)
            if [[ $backup_count -gt 5 ]]; then
                log::info "Cleaning old backups (keeping last 5)..."
                ls -1t "$backup_dir"/backup_*.tar.gz | tail -n +6 | xargs rm -f
            fi
            
            echo "$backup_file"
            return 0
        else
            log::error "Failed to create backup"
            return 1
        fi
    fi
}

#######################################
# Restore Home Assistant configuration
#######################################
home_assistant::restore() {
    home_assistant::init
    local backup_file="$1"
    
    if [[ -z "$backup_file" ]]; then
        # List available backups
        local backup_dir="${HOME_ASSISTANT_DATA_DIR}/backups"
        if [[ -d "$backup_dir" ]]; then
            log::info "Available backups:"
            ls -1t "$backup_dir"/backup_*.tar.gz 2>/dev/null || {
                log::warning "No backups found"
                return 1
            }
        else
            log::warning "No backup directory found"
            return 1
        fi
        return 0
    fi
    
    if [[ ! -f "$backup_file" ]]; then
        log::error "Backup file not found: $backup_file"
        return 1
    fi
    
    # Stop Home Assistant if running
    if docker::is_running "$HOME_ASSISTANT_CONTAINER_NAME"; then
        log::info "Stopping Home Assistant for restore..."
        home_assistant::docker::stop
    fi
    
    # Backup current config before restore
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local temp_backup="${HOME_ASSISTANT_DATA_DIR}/backups/pre_restore_${timestamp}.tar.gz"
    mkdir -p "${HOME_ASSISTANT_DATA_DIR}/backups"
    
    log::info "Creating safety backup before restore..."
    tar -czf "$temp_backup" -C "$HOME_ASSISTANT_CONFIG_DIR" . 2>/dev/null || true
    
    # Extract backup
    log::info "Restoring from backup: $backup_file"
    if tar -xzf "$backup_file" -C "$HOME_ASSISTANT_CONFIG_DIR"; then
        log::success "Configuration restored successfully"
        
        # Restart Home Assistant
        log::info "Starting Home Assistant with restored configuration..."
        home_assistant::docker::start
        return 0
    else
        log::error "Failed to restore backup"
        
        # Try to restore from safety backup
        if [[ -f "$temp_backup" ]]; then
            log::warning "Attempting to restore previous configuration..."
            tar -xzf "$temp_backup" -C "$HOME_ASSISTANT_CONFIG_DIR" 2>/dev/null || true
        fi
        return 1
    fi
}

#######################################
# List available backups
#######################################
home_assistant::backup::list() {
    home_assistant::init
    local backup_dir="${HOME_ASSISTANT_DATA_DIR}/backups"
    
    if [[ -d "$backup_dir" ]]; then
        log::header "Available Home Assistant Backups"
        local backups=$(ls -1t "$backup_dir"/backup_*.tar.gz 2>/dev/null)
        if [[ -n "$backups" ]]; then
            echo "$backups" | while read -r backup; do
                local size=$(du -h "$backup" | cut -f1)
                local date=$(basename "$backup" | sed 's/backup_\([0-9]*\)_\([0-9]*\).tar.gz/\1 \2/' | sed 's/\([0-9]\{4\}\)\([0-9]\{2\}\)\([0-9]\{2\}\) \([0-9]\{2\}\)\([0-9]\{2\}\)\([0-9]\{2\}\)/\1-\2-\3 \4:\5:\6/')
                log::info "📦 $(basename "$backup") - Size: $size - Date: $date"
            done
        else
            log::warning "No backups found"
        fi
    else
        log::warning "Backup directory does not exist"
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
export -f home_assistant::backup
export -f home_assistant::restore
export -f home_assistant::backup::list