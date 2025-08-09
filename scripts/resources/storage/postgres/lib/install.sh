#!/usr/bin/env bash
# PostgreSQL Installation Functions
# Handles installation and uninstallation of PostgreSQL resource

#######################################
# Install PostgreSQL resource
# Returns: 0 on success, 1 on failure
#######################################
postgres::install() {
    log::info "Installing PostgreSQL storage resource..."
    
    # Pre-installation checks
    if ! postgres::install::pre_checks; then
        return 1
    fi
    
    # Check if already installed (look for any existing instances)
    local instances=($(postgres::common::list_instances))
    if [[ ${#instances[@]} -gt 0 ]]; then
        log::info "PostgreSQL resource is already installed with ${#instances[@]} instance(s)"
        log::info "Use 'manage.sh --action list' to see all instances"
        return 0
    fi
    
    # Pull Docker image
    if ! postgres::docker::pull_image; then
        log::error "${MSG_INSTALL_FAILED}: Failed to pull Docker image"
        return 1
    fi
    
    # Create configuration directories
    if ! postgres::install::create_directories; then
        log::error "${MSG_INSTALL_FAILED}: Failed to create directories"
        return 1
    fi
    
    # Create templates
    if ! postgres::install::create_templates; then
        log::error "${MSG_INSTALL_FAILED}: Failed to create configuration templates"
        return 1
    fi
    
    # Update Vrooli configuration
    if ! postgres::install::update_vrooli_config; then
        log::warn "Failed to update Vrooli configuration"
        log::info "You may need to manually add PostgreSQL to ~/.vrooli/service.json"
    fi
    
    # Show success message
    log::success "${MSG_INSTALL_SUCCESS}"
    log::info ""
    log::info "PostgreSQL resource is now available for creating instances:"
    log::info "  • Create instance: manage.sh --action create --instance <name>"
    log::info "  • List instances:  manage.sh --action list"
    log::info "  • Get help:        manage.sh --action help"
    log::info ""
    log::info "Port range available: ${POSTGRES_INSTANCE_PORT_RANGE_START}-${POSTGRES_INSTANCE_PORT_RANGE_END}"
    log::info "Maximum instances: ${POSTGRES_MAX_INSTANCES}"
    
    return 0
}

#######################################
# Uninstall PostgreSQL resource
# Returns: 0 on success, 1 on failure
#######################################
postgres::uninstall() {
    log::info "Uninstalling PostgreSQL storage resource..."
    
    # Get all instances
    local instances=($(postgres::common::list_instances))
    
    if [[ ${#instances[@]} -gt 0 ]]; then
        log::warn "Found ${#instances[@]} PostgreSQL instance(s). These will be destroyed:"
        for instance in "${instances[@]}"; do
            log::info "  • $instance"
        done
        
        if ! flow::confirm "Are you sure you want to destroy all instances and uninstall?"; then
            log::info "Operation cancelled"
            return 1
        fi
        
        # Destroy all instances
        log::info "Destroying all PostgreSQL instances..."
        for instance in "${instances[@]}"; do
            if postgres::instance::destroy "$instance" "true"; then
                log::success "Destroyed instance: $instance"
            else
                log::error "Failed to destroy instance: $instance"
            fi
        done
    fi
    
    # Remove Docker network if no instances remain
    if docker network ls | grep -q "^${POSTGRES_NETWORK}"; then
        if docker network rm "${POSTGRES_NETWORK}" >/dev/null 2>&1; then
            log::debug "Removed Docker network: ${POSTGRES_NETWORK}"
        else
            log::warn "Failed to remove Docker network (may be in use by other containers)"
        fi
    fi
    
    # Remove instance directories
    if [[ -d "$POSTGRES_INSTANCES_DIR" ]]; then
        rm -rf "$POSTGRES_INSTANCES_DIR"
        log::debug "Removed instances directory"
    fi
    
    # Remove configuration directories
    if [[ -d "$POSTGRES_CONFIG_DIR" ]]; then
        rm -rf "$POSTGRES_CONFIG_DIR"
        log::debug "Removed configuration directory"
    fi
    
    # Remove from Vrooli configuration
    if ! postgres::install::remove_vrooli_config; then
        log::warn "Failed to remove from Vrooli configuration"
        log::info "You may need to manually remove PostgreSQL from ~/.vrooli/service.json"
    fi
    
    log::success "${MSG_UNINSTALL_SUCCESS}"
    return 0
}

#######################################
# Pre-installation checks
# Returns: 0 if all checks pass, 1 if any fail
#######################################
postgres::install::pre_checks() {
    local checks_passed=true
    
    log::info "${MSG_CHECKING_DOCKER}"
    if ! postgres::docker::check_docker; then
        log::error "${MSG_DOCKER_NOT_FOUND}"
        checks_passed=false
    fi
    
    log::info "Checking disk space..."
    if ! postgres::common::check_disk_space; then
        log::error "${MSG_INSUFFICIENT_DISK}"
        checks_passed=false
    fi
    
    log::info "${MSG_CHECKING_PORTS}"
    # Check if any ports in our range are blocked by critical services
    local blocked_ports=()
    for port in $(seq $POSTGRES_INSTANCE_PORT_RANGE_START $((POSTGRES_INSTANCE_PORT_RANGE_START + 5))); do
        if ! postgres::common::is_port_available "$port"; then
            blocked_ports+=("$port")
        fi
    done
    
    if [[ ${#blocked_ports[@]} -gt 3 ]]; then
        log::warn "Many ports in PostgreSQL range are in use: ${blocked_ports[*]}"
        log::info "Installation will continue, but fewer instances may be available"
    fi
    
    [[ "$checks_passed" == "true" ]] && return 0 || return 1
}

#######################################
# Create required directories
# Returns: 0 on success, 1 on failure
#######################################
postgres::install::create_directories() {
    log::info "${MSG_CREATING_DIRECTORIES}"
    
    # Create main instances directory
    if [[ ! -d "$POSTGRES_INSTANCES_DIR" ]]; then
        mkdir -p "$POSTGRES_INSTANCES_DIR"
        log::debug "Created instances directory: $POSTGRES_INSTANCES_DIR"
    fi
    
    # Create configuration directory
    if [[ ! -d "$POSTGRES_CONFIG_DIR" ]]; then
        mkdir -p "$POSTGRES_CONFIG_DIR"
        log::debug "Created config directory: $POSTGRES_CONFIG_DIR"
    fi
    
    # Create backup directory
    if [[ ! -d "$POSTGRES_BACKUP_DIR" ]]; then
        mkdir -p "$POSTGRES_BACKUP_DIR"
        log::debug "Created backup directory: $POSTGRES_BACKUP_DIR"
    fi
    
    # Set appropriate permissions
    chmod 700 "$POSTGRES_INSTANCES_DIR" "$POSTGRES_CONFIG_DIR" "$POSTGRES_BACKUP_DIR"
    
    return 0
}

#######################################
# Create configuration templates
# Returns: 0 on success, 1 on failure
#######################################
postgres::install::create_templates() {
    log::info "${MSG_APPLYING_TEMPLATE}"
    
    # Templates will be created in Phase 3, for now just create the directory
    if [[ ! -d "$POSTGRES_TEMPLATE_DIR" ]]; then
        mkdir -p "$POSTGRES_TEMPLATE_DIR"
        log::debug "Created template directory: $POSTGRES_TEMPLATE_DIR"
    fi
    
    # Create basic template files (minimal versions for now)
    cat > "${POSTGRES_TEMPLATE_DIR}/development.conf" << 'EOF'
# PostgreSQL Development Template
# Optimized for development work with verbose logging
max_connections = 100
shared_buffers = 128MB
effective_cache_size = 256MB
work_mem = 4MB
maintenance_work_mem = 64MB
log_statement = 'all'
log_min_duration_statement = 0
EOF

    cat > "${POSTGRES_TEMPLATE_DIR}/production.conf" << 'EOF'
# PostgreSQL Production Template
# Optimized for production workloads
max_connections = 200
shared_buffers = 256MB
effective_cache_size = 1GB
work_mem = 8MB
maintenance_work_mem = 128MB
checkpoint_completion_target = 0.9
wal_buffers = 16MB
default_statistics_target = 100
EOF

    cat > "${POSTGRES_TEMPLATE_DIR}/testing.conf" << 'EOF'
# PostgreSQL Testing Template
# Optimized for fast, ephemeral testing
max_connections = 50
shared_buffers = 64MB
effective_cache_size = 128MB
work_mem = 2MB
fsync = off
synchronous_commit = off
full_page_writes = off
EOF

    cat > "${POSTGRES_TEMPLATE_DIR}/minimal.conf" << 'EOF'
# PostgreSQL Minimal Template
# Minimal resource usage
max_connections = 20
shared_buffers = 32MB
effective_cache_size = 64MB
work_mem = 1MB
maintenance_work_mem = 16MB
EOF

    log::debug "Created configuration templates"
    return 0
}

#######################################
# Update Vrooli configuration
# Returns: 0 on success, 1 on failure
#######################################
postgres::install::update_vrooli_config() {
    local config_file
    config_file="$var_SERVICE_JSON_FILE"
    local base_url="http://localhost:${POSTGRES_DEFAULT_PORT}"
    
    # Create configuration JSON
    local postgres_config='{
        "enabled": true,
        "baseUrl": "'$base_url'",
        "description": "PostgreSQL instances for client isolation",
        "category": "storage",
        "portRange": {
            "start": '$POSTGRES_INSTANCE_PORT_RANGE_START',
            "end": '$POSTGRES_INSTANCE_PORT_RANGE_END'
        },
        "maxInstances": '$POSTGRES_MAX_INSTANCES',
        "healthCheck": {
            "intervalMs": 60000,
            "timeoutMs": 5000
        }
    }'
    
    # Use the common resource update function if available
    if command -v resources::update_config >/dev/null 2>&1; then
        resources::update_config "storage" "postgres" "$base_url" "$postgres_config"
    else
        # Fallback to manual configuration
        log::debug "Using manual configuration update"
        
        # Ensure directory exists
        mkdir -p "$(dirname "$config_file")"
        
        # Create basic config if it doesn't exist
        if [[ ! -f "$config_file" ]]; then
            echo '{"version": "1.0.0", "enabled": true, "services": {"storage": {}}}' > "$config_file"
        fi
        
        # Add PostgreSQL configuration using jq if available
        if command -v jq >/dev/null 2>&1; then
            local temp_config=$(mktemp)
            jq --argjson config "$postgres_config" '.services.storage.postgres = $config' "$config_file" > "$temp_config"
            mv "$temp_config" "$config_file"
        else
            log::warn "jq not available, skipping configuration update"
            return 1
        fi
    fi
    
    return 0
}

#######################################
# Remove from Vrooli configuration
# Returns: 0 on success, 1 on failure
#######################################
postgres::install::remove_vrooli_config() {
    local config_file
    config_file="$var_SERVICE_JSON_FILE"
    
    if [[ ! -f "$config_file" ]]; then
        return 0  # Nothing to remove
    fi
    
    # Use common resource remove function if available
    if command -v resources::remove_config >/dev/null 2>&1; then
        resources::remove_config "storage" "postgres"
    else
        # Fallback to manual removal
        if command -v jq >/dev/null 2>&1; then
            local temp_config=$(mktemp)
            jq 'del(.services.storage.postgres)' "$config_file" > "$temp_config"
            mv "$temp_config" "$config_file"
        else
            log::warn "jq not available, skipping configuration removal"
            return 1
        fi
    fi
    
    return 0
}