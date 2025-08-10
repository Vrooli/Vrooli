#!/usr/bin/env bash
# PostgreSQL Instance Management
# High-level functions for managing PostgreSQL instances

#######################################
# Create a new PostgreSQL instance
# Arguments:
#   $1 - instance name (default: main)
#   $2 - port (optional, auto-assigned if not specified)
#   $3 - template (default: development)
# Returns: 0 on success, 1 on failure
#######################################
postgres::instance::create() {
    local instance_name="${1:-main}"
    local port="${2:-}"
    local template="${3:-development}"
    local networks="${4:-}"
    
    # Validate instance name
    if ! postgres::instance::validate_name "$instance_name"; then
        return 1
    fi
    
    # Check if instance already exists
    if postgres::common::container_exists "$instance_name"; then
        log::error "PostgreSQL instance '$instance_name' already exists"
        return 1
    fi
    
    # Check maximum instances limit
    local current_count=$(postgres::common::count_running_instances)
    if [[ $current_count -ge $POSTGRES_MAX_INSTANCES ]]; then
        log::error "${MSG_MAX_INSTANCES} (current: $current_count, max: $POSTGRES_MAX_INSTANCES)"
        return 1
    fi
    
    # Allocate port if not specified
    if [[ -z "$port" ]]; then
        log::info "${MSG_FINDING_PORT}"
        port=$(postgres::instance::find_available_port)
        if [[ -z "$port" ]]; then
            log::error "${MSG_NO_AVAILABLE_PORT}"
            return 1
        fi
        log::info "Found available port: $port"
    else
        # Validate specified port
        if ! postgres::instance::validate_port "$port"; then
            return 1
        fi
        
        # Double-check port availability for manually specified ports
        if ! postgres::common::is_port_available "$port"; then
            log::error "Port $port is not available"
            # Show what's using the port if possible
            if command -v lsof >/dev/null 2>&1; then
                local port_user=$(lsof -i :${port} -sTCP:LISTEN -t 2>/dev/null | head -1)
                if [[ -n "$port_user" ]]; then
                    local process_info=$(ps -p "$port_user" -o comm= 2>/dev/null || echo "unknown")
                    log::info "Port $port is being used by process: $process_info (PID: $port_user)"
                fi
            fi
            return 1
        fi
    fi
    
    # Generate secure credentials
    local password=$(postgres::common::generate_password)
    
    # Handle networks - merge template networks with manually specified ones
    local template_networks=$(postgres::network::load_template_networks "$template")
    local all_networks=""
    
    if [[ -n "$template_networks" ]] && [[ -n "$networks" ]]; then
        # Merge both, removing duplicates
        all_networks=$(echo "$template_networks,$networks" | tr ',' '\n' | sort -u | tr '\n' ',' | sed 's/,$//')
    elif [[ -n "$template_networks" ]]; then
        all_networks="$template_networks"
    elif [[ -n "$networks" ]]; then
        all_networks="$networks"
    fi
    
    # Validate network names
    if [[ -n "$all_networks" ]]; then
        if ! postgres::network::validate_names "$all_networks"; then
            return 1
        fi
        log::info "Instance will join additional networks: $all_networks"
    fi
    
    log::info "Creating PostgreSQL instance '$instance_name' on port $port with template '$template'"
    
    # Final port availability check before creating any configuration
    if ! postgres::common::is_port_available "$port"; then
        log::error "Port $port is no longer available (may have been claimed by another process)"
        return 1
    fi
    
    # Save instance configuration (including networks)
    postgres::instance::save_config "$instance_name" "$port" "$password" "$template" "$all_networks"
    
    # Create and start PostgreSQL container
    if postgres::docker::create_container "$instance_name" "$port" "$password" "$template"; then
        # Connect to additional networks if specified
        if [[ -n "$all_networks" ]]; then
            local container_name="${POSTGRES_CONTAINER_PREFIX}-${instance_name}"
            if ! postgres::network::connect_container "$container_name" "$all_networks"; then
                log::warn "Some network connections failed, but container is running"
            fi
        fi
        
        # Wait for container to be ready
        if postgres::common::wait_for_ready "$instance_name"; then
            log::success "${MSG_CREATE_SUCCESS}"
            log::info "${MSG_INSTANCE_AVAILABLE}: localhost:$port"
            log::info "${MSG_HELP_CONNECTION}"
            return 0
        else
            log::error "${MSG_CREATE_FAILED}"
            # Clean up on failure
            postgres::instance::destroy "$instance_name" "true" >/dev/null 2>&1
            return 1
        fi
    else
        log::error "${MSG_CREATE_FAILED}"
        # Clean up configuration on Docker container creation failure
        local instance_dir="${POSTGRES_INSTANCES_DIR}/${instance_name}"
        if [[ -d "$instance_dir" ]]; then
            trash::safe_remove "$instance_dir" --no-confirm
            log::debug "Cleaned up instance configuration due to container creation failure"
        fi
        return 1
    fi
}

#######################################
# Destroy PostgreSQL instance
# Arguments:
#   $1 - instance name
#   $2 - force flag (optional, default: false)
# Returns: 0 on success, 1 on failure
#######################################
postgres::instance::destroy() {
    local instance_name="${1:-main}"
    local force="${2:-false}"
    
    if ! postgres::common::container_exists "$instance_name"; then
        postgres::common::show_instance_not_found_error "$instance_name"
        return 0
    fi
    
    # Confirm destruction unless forced
    if [[ "$force" != "true" && "$force" != "yes" ]]; then
        read -p "Are you sure you want to destroy instance '$instance_name'? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log::info "Operation cancelled"
            return 1
        fi
    fi
    
    log::info "Destroying PostgreSQL instance '$instance_name'..."
    
    # Remove Docker container
    if postgres::docker::remove "$instance_name" "$force"; then
        # Remove instance directory with proper permission handling
        local instance_dir="${POSTGRES_INSTANCES_DIR}/${instance_name}"
        if [[ -d "$instance_dir" ]]; then
            # First try normal removal
            if trash::safe_remove "$instance_dir" --no-confirm 2>/dev/null; then
                log::debug "Removed instance directory: $instance_dir"
            else
                # If that fails, try using Docker to remove root-owned files
                log::debug "Normal removal failed, using Docker to clean up root-owned files"
                
                # Use a temporary container to remove files with proper permissions
                if docker run --rm -v "${instance_dir}:/cleanup" alpine:latest sh -c "rm -rf /cleanup/*" 2>/dev/null; then
                    # Now remove the empty directory
                    rmdir "$instance_dir" 2>/dev/null || trash::safe_remove "$instance_dir" --no-confirm
                    log::debug "Removed instance directory using Docker: $instance_dir"
                else
                    log::warn "Could not fully remove instance directory. Manual cleanup may be required: $instance_dir"
                    log::warn "You can try: sudo rm -rf $instance_dir"
                fi
            fi
        fi
        
        log::success "${MSG_DESTROY_SUCCESS}"
        return 0
    else
        log::error "Failed to destroy PostgreSQL instance '$instance_name'"
        return 1
    fi
}

#######################################
# Start PostgreSQL instance
# Arguments:
#   $1 - instance name
# Returns: 0 on success, 1 on failure
#######################################
postgres::instance::start() {
    local instance_name="${1:-main}"
    
    if [[ "$instance_name" == "all" ]]; then
        postgres::instance::start_all
        return $?
    fi
    
    postgres::docker::start "$instance_name"
}

#######################################
# Stop PostgreSQL instance
# Arguments:
#   $1 - instance name
# Returns: 0 on success, 1 on failure
#######################################
postgres::instance::stop() {
    local instance_name="${1:-main}"
    
    if [[ "$instance_name" == "all" ]]; then
        postgres::instance::stop_all
        return $?
    fi
    
    postgres::docker::stop "$instance_name"
}

#######################################
# Restart PostgreSQL instance
# Arguments:
#   $1 - instance name
# Returns: 0 on success, 1 on failure
#######################################
postgres::instance::restart() {
    local instance_name="${1:-main}"
    
    if [[ "$instance_name" == "all" ]]; then
        postgres::instance::restart_all
        return $?
    fi
    
    postgres::docker::restart "$instance_name"
}

#######################################
# Start all PostgreSQL instances
# Returns: 0 on success, 1 if any failed
#######################################
postgres::instance::start_all() {
    local instances=($(postgres::common::list_instances))
    local failed=0
    
    if [[ ${#instances[@]} -eq 0 ]]; then
        log::info "No PostgreSQL instances found"
        return 0
    fi
    
    log::info "Starting all PostgreSQL instances..."
    
    for instance in "${instances[@]}"; do
        if postgres::docker::start "$instance"; then
            log::success "Started instance: $instance"
        else
            log::error "Failed to start instance: $instance"
            ((failed++))
        fi
    done
    
    if [[ $failed -eq 0 ]]; then
        log::success "All instances started successfully"
        return 0
    else
        log::error "$failed instance(s) failed to start"
        return 1
    fi
}

#######################################
# Stop all PostgreSQL instances
# Returns: 0 on success, 1 if any failed
#######################################
postgres::instance::stop_all() {
    local instances=($(postgres::common::list_instances))
    local failed=0
    
    if [[ ${#instances[@]} -eq 0 ]]; then
        log::info "No PostgreSQL instances found"
        return 0
    fi
    
    log::info "Stopping all PostgreSQL instances..."
    
    for instance in "${instances[@]}"; do
        if postgres::docker::stop "$instance"; then
            log::success "Stopped instance: $instance"
        else
            log::error "Failed to stop instance: $instance"
            ((failed++))
        fi
    done
    
    if [[ $failed -eq 0 ]]; then
        log::success "All instances stopped successfully"
        return 0
    else
        log::error "$failed instance(s) failed to stop"
        return 1
    fi
}

#######################################
# Restart all PostgreSQL instances
# Returns: 0 on success, 1 if any failed
#######################################
postgres::instance::restart_all() {
    local instances=($(postgres::common::list_instances))
    local failed=0
    
    if [[ ${#instances[@]} -eq 0 ]]; then
        log::info "No PostgreSQL instances found"
        return 0
    fi
    
    log::info "Restarting all PostgreSQL instances..."
    
    for instance in "${instances[@]}"; do
        if postgres::docker::restart "$instance"; then
            log::success "Restarted instance: $instance"
        else
            log::error "Failed to restart instance: $instance"
            ((failed++))
        fi
    done
    
    if [[ $failed -eq 0 ]]; then
        log::success "All instances restarted successfully"
        return 0
    else
        log::error "$failed instance(s) failed to restart"
        return 1
    fi
}

#######################################
# List all PostgreSQL instances with status
# Returns: 0 on success
#######################################
postgres::instance::list() {
    log::info "${MSG_LISTING_INSTANCES}"
    echo
    
    local instances=($(postgres::common::list_instances))
    
    if [[ ${#instances[@]} -eq 0 ]]; then
        log::info "No PostgreSQL instances found"
        echo
        log::info "${MSG_HELP_CREATE}"
        return 0
    fi
    
    printf "%-15s %-8s %-6s %-10s %-15s\n" "NAME" "STATUS" "PORT" "TEMPLATE" "CREATED"
    printf "%-15s %-8s %-6s %-10s %-15s\n" "----" "------" "----" "--------" "-------"
    
    for instance in "${instances[@]}"; do
        local status="stopped"
        local port=$(postgres::common::get_instance_config "$instance" "port")
        local template=$(postgres::common::get_instance_config "$instance" "template")
        local created=$(postgres::common::get_instance_config "$instance" "created")
        
        if postgres::common::is_running "$instance"; then
            if postgres::common::health_check "$instance"; then
                status="healthy"
            else
                status="unhealthy"
            fi
        fi
        
        printf "%-15s %-8s %-6s %-10s %-15s\n" "$instance" "$status" "${port:-N/A}" "${template:-dev}" "${created:-N/A}"
    done
    
    echo
    log::info "${MSG_HELP_CONNECTION}"
}

#######################################
# Get connection string for instance
# Arguments:
#   $1 - instance name
# Returns: 0 on success, 1 on failure
#######################################
postgres::instance::get_connection_string() {
    local instance_name="${1:-main}"
    
    if ! postgres::common::container_exists "$instance_name"; then
        log::error "${MSG_INSTANCE_NOT_FOUND}: $instance_name"
        echo ""  # Return empty string for command substitution
        return 1
    fi
    
    local port=$(postgres::common::get_instance_config "$instance_name" "port")
    local password=$(postgres::common::get_instance_config "$instance_name" "password")
    
    if [[ -z "$port" || -z "$password" ]]; then
        log::error "Instance configuration incomplete"
        echo ""  # Return empty string for command substitution
        return 1
    fi
    
    echo "postgresql://${POSTGRES_DEFAULT_USER}:${password}@localhost:${port}/${POSTGRES_DEFAULT_DB}"
}

#######################################
# Find available port in PostgreSQL range
# Returns: Available port number or empty if none available
#######################################
postgres::instance::find_available_port() {
    postgres::common::find_available_port
}

#######################################
# Save instance configuration
# Arguments:
#   $1 - instance name
#   $2 - port
#   $3 - password
#   $4 - template
#   $5 - networks (optional)
#######################################
postgres::instance::save_config() {
    local instance_name="$1"
    local port="$2"
    local password="$3"
    local template="$4"
    local networks="${5:-}"
    local created=$(date -Iseconds)
    
    postgres::common::set_instance_config "$instance_name" "port" "$port"
    postgres::common::set_instance_config "$instance_name" "password" "$password"
    postgres::common::set_instance_config "$instance_name" "template" "$template"
    postgres::common::set_instance_config "$instance_name" "created" "$created"
    postgres::common::set_instance_config "$instance_name" "user" "$POSTGRES_DEFAULT_USER"
    postgres::common::set_instance_config "$instance_name" "database" "$POSTGRES_DEFAULT_DB"
    
    # Save networks if specified
    if [[ -n "$networks" ]]; then
        postgres::common::set_instance_config "$instance_name" "networks" "$networks"
    fi
}

#######################################
# Validate instance name
# Arguments:
#   $1 - instance name
# Returns: 0 if valid, 1 if invalid
#######################################
postgres::instance::validate_name() {
    local instance_name="$1"
    
    # Check if name is provided
    if [[ -z "$instance_name" ]]; then
        log::error "Instance name cannot be empty"
        return 1
    fi
    
    # Check for valid characters (alphanumeric, hyphens, underscores)
    if [[ ! "$instance_name" =~ ^[a-zA-Z0-9_-]+$ ]]; then
        log::error "Instance name can only contain letters, numbers, hyphens, and underscores"
        return 1
    fi
    
    # Check length
    if [[ ${#instance_name} -gt 50 ]]; then
        log::error "Instance name cannot exceed 50 characters"
        return 1
    fi
    
    return 0
}

#######################################
# Validate port number
# Arguments:
#   $1 - port number
# Returns: 0 if valid, 1 if invalid
#######################################
postgres::instance::validate_port() {
    local port="$1"
    
    # Check if port is in valid range
    if [[ "$port" -lt $POSTGRES_INSTANCE_PORT_RANGE_START || "$port" -gt $POSTGRES_INSTANCE_PORT_RANGE_END ]]; then
        log::error "Port must be between $POSTGRES_INSTANCE_PORT_RANGE_START and $POSTGRES_INSTANCE_PORT_RANGE_END"
        return 1
    fi
    
    # Check if port is available
    if ! postgres::common::is_port_available "$port"; then
        log::error "Port $port is already in use"
        return 1
    fi
    
    return 0
}