#!/usr/bin/env bash
# Network management functions for PostgreSQL resource

#######################################
# Check if a Docker network exists
# Arguments:
#   $1 - network name
# Returns: 0 if exists, 1 if not
#######################################
postgres::network::exists() {
    local network="$1"
    docker network ls --format '{{.Name}}' | grep -q "^${network}$"
}

#######################################
# Create a Docker network if it doesn't exist
# Arguments:
#   $1 - network name
# Returns: 0 on success, 1 on failure
#######################################
postgres::network::create() {
    local network="$1"
    
    if postgres::network::exists "$network"; then
        log::debug "Network already exists: $network"
        return 0
    fi
    
    log::info "Creating Docker network: $network"
    if docker network create "$network" >/dev/null 2>&1; then
        log::debug "Network created successfully: $network"
        return 0
    else
        log::error "Failed to create network: $network"
        return 1
    fi
}

#######################################
# Discover existing resource networks
# Returns: Space-separated list of discovered networks
#######################################
postgres::network::discover_resource_networks() {
    local discovered_networks=()
    local resource_networks=(
        "n8n-network"
        "node-red-network"
        "windmill-network"
        "huginn-network"
        "comfyui-network"
        "searxng-network"
        "minio-network"
        "qdrant-network"
    )
    
    for network in "${resource_networks[@]}"; do
        if postgres::network::exists "$network"; then
            discovered_networks+=("$network")
        fi
    done
    
    echo "${discovered_networks[@]}"
}

#######################################
# Validate network names
# Arguments:
#   $1 - comma-separated network names
# Returns: 0 if all valid, 1 if any invalid
#######################################
postgres::network::validate_names() {
    local networks="$1"
    
    # Empty is valid (no additional networks)
    [[ -z "$networks" ]] && return 0
    
    # Split comma-separated list
    IFS=',' read -ra network_array <<< "$networks"
    
    for network in "${network_array[@]}"; do
        # Trim whitespace
        network=$(echo "$network" | xargs)
        
        # Check for valid Docker network name
        if [[ ! "$network" =~ ^[a-zA-Z0-9][a-zA-Z0-9_.-]*$ ]]; then
            log::error "Invalid network name: $network"
            log::info "Network names must start with alphanumeric and contain only alphanumeric, underscore, period, or hyphen"
            return 1
        fi
    done
    
    return 0
}

#######################################
# Connect container to additional networks
# Arguments:
#   $1 - container name
#   $2 - comma-separated network names
# Returns: 0 on success, 1 on failure
#######################################
postgres::network::connect_container() {
    local container="$1"
    local networks="$2"
    
    # Empty networks means no additional connections needed
    [[ -z "$networks" ]] && return 0
    
    # Split comma-separated list
    IFS=',' read -ra network_array <<< "$networks"
    
    local failed=0
    for network in "${network_array[@]}"; do
        # Trim whitespace
        network=$(echo "$network" | xargs)
        
        log::info "Connecting container to network: $network"
        
        # Create network if it doesn't exist
        if ! postgres::network::exists "$network"; then
            log::warn "Network '$network' doesn't exist, creating it..."
            if ! postgres::network::create "$network"; then
                log::error "Failed to create network: $network"
                ((failed++))
                continue
            fi
        fi
        
        # Connect container to network
        if docker network connect "$network" "$container" 2>/dev/null; then
            log::debug "Successfully connected to network: $network"
        else
            # Check if already connected
            if docker inspect "$container" | grep -q "\"$network\""; then
                log::debug "Container already connected to network: $network"
            else
                log::error "Failed to connect container to network: $network"
                ((failed++))
            fi
        fi
    done
    
    [[ $failed -eq 0 ]] && return 0 || return 1
}

#######################################
# Disconnect container from additional networks
# Arguments:
#   $1 - container name
#   $2 - comma-separated network names
# Returns: 0 on success, 1 on failure
#######################################
postgres::network::disconnect_container() {
    local container="$1"
    local networks="$2"
    
    # Empty networks means no disconnections needed
    [[ -z "$networks" ]] && return 0
    
    # Split comma-separated list
    IFS=',' read -ra network_array <<< "$networks"
    
    for network in "${network_array[@]}"; do
        # Trim whitespace
        network=$(echo "$network" | xargs)
        
        # Skip primary network
        [[ "$network" == "$POSTGRES_NETWORK" ]] && continue
        
        log::debug "Disconnecting container from network: $network"
        docker network disconnect "$network" "$container" 2>/dev/null || true
    done
    
    return 0
}

#######################################
# Get networks for a container
# Arguments:
#   $1 - container name
# Returns: Comma-separated list of networks
#######################################
postgres::network::get_container_networks() {
    local container="$1"
    local networks=$(docker inspect "$container" 2>/dev/null | jq -r '.[0].NetworkSettings.Networks | keys | join(",")')
    echo "$networks"
}

#######################################
# Load network profile from template
# Arguments:
#   $1 - template name
# Returns: Comma-separated list of networks
#######################################
postgres::network::load_template_networks() {
    local template="$1"
    local template_network_file="${POSTGRES_TEMPLATE_DIR}/${template}.networks"
    
    if [[ -f "$template_network_file" ]]; then
        # Read networks from file, filter comments and empty lines
        local networks=$(grep -v '^#' "$template_network_file" 2>/dev/null | grep -v '^$' | tr '\n' ',' | sed 's/,$//')
        echo "$networks"
    else
        # Return empty for templates without network profiles
        echo ""
    fi
}

#######################################
# Suggest networks based on running resources
# Returns: Formatted suggestion message
#######################################
postgres::network::suggest_networks() {
    local discovered=$(postgres::network::discover_resource_networks)
    
    if [[ -n "$discovered" ]]; then
        log::info "Discovered resource networks: $discovered"
        log::info "To connect to these resources, use: --networks \"$discovered\""
    else
        log::debug "No resource networks discovered"
    fi
}

#######################################
# Update networks for existing instance
# Arguments:
#   $1 - instance name
# Returns: 0 on success, 1 on failure
#######################################
postgres::network::update_instance() {
    local instance_name="${1:-main}"
    
    if ! postgres::common::container_exists "$instance_name"; then
        log::error "${MSG_INSTANCE_NOT_FOUND}: $instance_name"
        return 1
    fi
    
    if ! postgres::common::is_running "$instance_name"; then
        log::error "Instance '$instance_name' must be running to update networks"
        return 1
    fi
    
    # Get template from instance config
    local template=$(postgres::common::get_instance_config "$instance_name" "template")
    if [[ -z "$template" ]]; then
        template="development"  # Default fallback
        log::warn "No template found for instance '$instance_name', using 'development'"
    fi
    
    local template_networks=$(postgres::network::load_template_networks "$template")
    
    if [[ -n "$template_networks" ]]; then
        log::info "Connecting instance '$instance_name' to networks: $template_networks"
        local container_name="${POSTGRES_CONTAINER_PREFIX}-${instance_name}"
        
        if postgres::network::connect_container "$container_name" "$template_networks"; then
            # Update instance config to record networks
            postgres::common::set_instance_config "$instance_name" "networks" "$template_networks"
            log::success "Network update completed for instance: $instance_name"
            return 0
        else
            log::error "Failed to update networks for instance: $instance_name"
            return 1
        fi
    else
        log::info "No additional networks configured for template: $template"
        return 0
    fi
}

#######################################
# Migrate networks for all instances
# Returns: 0 on success, 1 if any failures
#######################################
postgres::network::migrate_all_instances() {
    local instances=($(postgres::common::list_instances))
    local updated=0
    local failed=0
    local skipped=0
    
    if [[ ${#instances[@]} -eq 0 ]]; then
        log::info "No instances found to migrate"
        return 0
    fi
    
    log::info "Migrating networks for ${#instances[@]} instance(s)..."
    
    for instance in "${instances[@]}"; do
        local current_networks=$(postgres::common::get_instance_config "$instance" "networks")
        
        # Skip if already has automation networks
        if [[ "$current_networks" == *"n8n-network"* ]]; then
            log::debug "Instance '$instance' already has updated networks"
            ((skipped++))
            continue
        fi
        
        # Skip if not running
        if ! postgres::common::is_running "$instance"; then
            log::warn "Instance '$instance' is not running, skipping network update"
            ((skipped++))
            continue
        fi
        
        log::info "Updating networks for instance: $instance"
        if postgres::network::update_instance "$instance"; then
            ((updated++))
        else
            ((failed++))
        fi
    done
    
    log::info "Network migration completed: $updated updated, $skipped skipped, $failed failed"
    [[ $failed -eq 0 ]] && return 0 || return 1
}