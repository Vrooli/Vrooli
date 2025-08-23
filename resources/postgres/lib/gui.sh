#!/usr/bin/env bash
# PostgreSQL GUI (pgweb) Management
# Functions for managing pgweb GUI containers for PostgreSQL instances

#######################################
# Check if GUI container exists for instance
# Arguments:
#   $1 - instance name
# Returns: 0 if exists, 1 if not
#######################################
postgres::gui::container_exists() {
    local instance_name="${1:-main}"
    local container_name="${POSTGRES_GUI_CONTAINER_PREFIX}-${instance_name}"
    docker ps -a --format "{{.Names}}" | grep -q "^${container_name}$"
}

#######################################
# Check if GUI container is running for instance
# Arguments:
#   $1 - instance name
# Returns: 0 if running, 1 if not
#######################################
postgres::gui::is_running() {
    local instance_name="${1:-main}"
    local container_name="${POSTGRES_GUI_CONTAINER_PREFIX}-${instance_name}"
    docker ps --format "{{.Names}}" | grep -q "^${container_name}$"
}

#######################################
# Find available port for GUI
# Returns: Available port number or empty if none available
#######################################
postgres::gui::find_available_port() {
    for port in $(seq $POSTGRES_GUI_PORT_RANGE_START $POSTGRES_GUI_PORT_RANGE_END); do
        if postgres::common::is_port_available "$port"; then
            echo "$port"
            return 0
        fi
    done
    return 1
}

#######################################
# Validate GUI port number
# Arguments:
#   $1 - port number
# Returns: 0 if valid, 1 if invalid
#######################################
postgres::gui::validate_port() {
    local port=$1
    
    if [[ ! "$port" =~ ^[0-9]+$ ]]; then
        log::error "Invalid port number: $port"
        return 1
    fi
    
    if [[ $port -lt $POSTGRES_GUI_PORT_RANGE_START ]] || [[ $port -gt $POSTGRES_GUI_PORT_RANGE_END ]]; then
        log::error "Port $port is outside allowed range ($POSTGRES_GUI_PORT_RANGE_START-$POSTGRES_GUI_PORT_RANGE_END)"
        return 1
    fi
    
    if ! postgres::common::is_port_available "$port"; then
        log::error "Port $port is already in use"
        return 1
    fi
    
    return 0
}

#######################################
# Get PostgreSQL connection details for GUI
# Arguments:
#   $1 - instance name
# Returns: Connection string for pgweb
#######################################
postgres::gui::get_connection_string() {
    local instance_name="${1:-main}"
    local instance_config_file="${POSTGRES_INSTANCES_DIR}/${instance_name}/config/instance.conf"
    
    if [[ ! -f "$instance_config_file" ]]; then
        log::error "Instance configuration not found: $instance_name"
        return 1
    fi
    
    # Read instance configuration
    local user password database
    # shellcheck disable=SC1090
    source "$instance_config_file"
    
    local container_name="${POSTGRES_CONTAINER_PREFIX}-${instance_name}"
    echo "postgres://${user}:${password}@${container_name}:5432/${database}?sslmode=disable"
}

#######################################
# Get networks for instance
# Arguments:
#   $1 - instance name
# Returns: 0 on success, 1 on failure
#######################################
postgres::gui::get_instance_networks() {
    local instance_name="${1:-main}"
    local instance_config_file="${POSTGRES_INSTANCES_DIR}/${instance_name}/config/instance.conf"
    
    if [[ ! -f "$instance_config_file" ]]; then
        log::error "Instance configuration not found: $instance_name"
        return 1
    fi
    
    # Read networks from instance config
    local networks
    # shellcheck disable=SC1090
    source "$instance_config_file"
    
    if [[ -n "${networks:-}" ]]; then
        echo "$networks"
    else
        echo "${POSTGRES_NETWORK}"
    fi
}

#######################################
# Start GUI for PostgreSQL instance
# Arguments:
#   $1 - instance name (default: main)
#   $2 - GUI port (optional, auto-assigned if not specified)
# Returns: 0 on success, 1 on failure
#######################################
postgres::gui::start() {
    local instance_name="${1:-main}"
    local gui_port="${2:-}"
    
    # Check if PostgreSQL instance exists and is running
    if ! postgres::common::container_exists "$instance_name"; then
        log::error "PostgreSQL instance '$instance_name' does not exist"
        return 1
    fi
    
    if ! postgres::common::is_running "$instance_name"; then
        log::error "PostgreSQL instance '$instance_name' is not running"
        return 1
    fi
    
    # Check if GUI already running
    if postgres::gui::is_running "$instance_name"; then
        log::info "GUI for instance '$instance_name' is already running"
        postgres::gui::show_status "$instance_name"
        return 0
    fi
    
    # Allocate port if not specified
    if [[ -z "$gui_port" ]]; then
        log::info "Finding available port for GUI..."
        gui_port=$(postgres::gui::find_available_port)
        if [[ -z "$gui_port" ]]; then
            log::error "No available ports in range ($POSTGRES_GUI_PORT_RANGE_START-$POSTGRES_GUI_PORT_RANGE_END)"
            return 1
        fi
        log::info "Found available port: $gui_port"
    else
        # Validate specified port
        if ! postgres::gui::validate_port "$gui_port"; then
            return 1
        fi
    fi
    
    # Get connection details
    local connection_string
    connection_string=$(postgres::gui::get_connection_string "$instance_name")
    if [[ $? -ne 0 ]]; then
        return 1
    fi
    
    # Get networks to join
    local networks
    networks=$(postgres::gui::get_instance_networks "$instance_name")
    if [[ $? -ne 0 ]]; then
        return 1
    fi
    
    local container_name="${POSTGRES_GUI_CONTAINER_PREFIX}-${instance_name}"
    
    log::info "Starting pgweb GUI for instance '$instance_name' on port $gui_port"
    
    # Start pgweb container
    local docker_cmd=(
        docker run -d
        --name "$container_name"
        --restart unless-stopped
        -p "${gui_port}:8081"
        -e "DATABASE_URL=${connection_string}"
        -e "PGWEB_DATABASE_URL=${connection_string}"
    )
    
    # Add networks
    IFS=',' read -ra network_array <<< "$networks"
    for network in "${network_array[@]}"; do
        network=$(echo "$network" | xargs)  # trim whitespace
        if [[ -n "$network" ]]; then
            docker_cmd+=(--network "$network")
        fi
    done
    
    docker_cmd+=("$POSTGRES_GUI_IMAGE")
    
    if container_id=$("${docker_cmd[@]}" 2>&1); then
        log::debug "pgweb container started: $container_id"
        
        # Connect to additional networks (skip the first one as it's already connected)
        if [[ ${#network_array[@]} -gt 1 ]]; then
            for network in "${network_array[@]:1}"; do
                network=$(echo "$network" | xargs)  # trim whitespace
                if [[ -n "$network" ]]; then
                    log::info "Connecting GUI container to network: $network"
                    docker network connect "$network" "$container_name" 2>/dev/null || true
                fi
            done
        fi
        
        # Wait for GUI to start
        log::info "Waiting for pgweb to start..."
        local max_attempts=30
        local attempt=0
        
        # Show progress dots
        while [[ $attempt -lt $max_attempts ]]; do
            if curl -s "http://localhost:${gui_port}" >/dev/null 2>&1; then
                echo  # New line after progress dots
                log::success "✅ pgweb GUI started successfully"
                log::info ""
                log::info "GUI Access Information:"
                log::info "========================"
                log::info "URL: http://localhost:${gui_port}"
                log::info "Instance: $instance_name"
                
                # Get and display connection info
                local password=$(postgres::common::get_instance_config "$instance_name" "password" 2>/dev/null || echo "Check instance config")
                log::info "Database: ${POSTGRES_DEFAULT_DB}"
                log::info "Username: ${POSTGRES_DEFAULT_USER}"
                log::info "Password: $password"
                log::info ""
                log::info "Note: The GUI is already connected. Just open the URL in your browser!"
                return 0
            fi
            
            # Show progress
            printf "."
            sleep 1
            ((attempt++))
        done
        
        echo  # New line after progress dots
        
        log::error "GUI failed to start within expected time"
        postgres::gui::stop "$instance_name"
        return 1
    else
        log::error "Failed to start pgweb container: $container_id"
        return 1
    fi
}

#######################################
# Stop GUI for PostgreSQL instance
# Arguments:
#   $1 - instance name (default: main)
# Returns: 0 on success, 1 on failure
#######################################
postgres::gui::stop() {
    local instance_name="${1:-main}"
    local container_name="${POSTGRES_GUI_CONTAINER_PREFIX}-${instance_name}"
    
    if ! postgres::gui::container_exists "$instance_name"; then
        log::warning "GUI for instance '$instance_name' does not exist"
        return 0
    fi
    
    log::info "Stopping GUI for instance '$instance_name'"
    
    if docker stop "$container_name" >/dev/null 2>&1; then
        docker rm "$container_name" >/dev/null 2>&1
        log::success "GUI stopped successfully"
        return 0
    else
        log::error "Failed to stop GUI container"
        return 1
    fi
}

#######################################
# Show GUI status for instance
# Arguments:
#   $1 - instance name (default: main)
# Returns: 0 on success, 1 on failure
#######################################
postgres::gui::show_status() {
    local instance_name="${1:-main}"
    local container_name="${POSTGRES_GUI_CONTAINER_PREFIX}-${instance_name}"
    
    log::info "GUI Status for instance: $instance_name"
    log::info "=================================="
    
    if ! postgres::gui::container_exists "$instance_name"; then
        log::info "Status: Not created"
        return 0
    fi
    
    if postgres::gui::is_running "$instance_name"; then
        local port
        port=$(docker port "$container_name" 8081/tcp 2>/dev/null | cut -d: -f2)
        
        log::success "Status: Running"
        log::info "Port: ${port:-unknown}"
        log::info "URL: http://localhost:${port:-unknown}"
        log::info "Container: $container_name"
        
        # Test connectivity
        if [[ -n "$port" ]] && curl -s "http://localhost:${port}" >/dev/null 2>&1; then
            log::success "Health: Healthy"
            
            # Show connection details if healthy
            log::info ""
            log::info "Connection Details:"
            log::info "-------------------"
            local password=$(postgres::common::get_instance_config "$instance_name" "password" 2>/dev/null || echo "Check instance config")
            log::info "Database: ${POSTGRES_DEFAULT_DB}"
            log::info "Username: ${POSTGRES_DEFAULT_USER}"
            log::info "Password: $password"
            log::info ""
            log::info "Access the GUI at: http://localhost:${port}"
        else
            log::warning "Health: Unhealthy (not responding)"
        fi
    else
        log::warning "Status: Stopped"
        log::info "Start with: ./manage.sh --action gui --instance $instance_name"
    fi
    
    return 0
}

#######################################
# List all GUI instances
# Returns: 0 on success
#######################################
postgres::gui::list() {
    log::info "PostgreSQL GUI Instances"
    log::info "========================"
    
    printf "%-20s %-10s %-10s %-30s\n" "INSTANCE" "STATUS" "PORT" "URL"
    printf "%-20s %-10s %-10s %-30s\n" "--------" "------" "----" "---"
    
    # Find all GUI containers
    local containers
    containers=$(docker ps -a --filter "name=${POSTGRES_GUI_CONTAINER_PREFIX}-" --format "{{.Names}}" 2>/dev/null)
    
    if [[ -z "$containers" ]]; then
        log::info "No GUI instances found"
        return 0
    fi
    
    while IFS= read -r container_name; do
        local instance_name="${container_name#${POSTGRES_GUI_CONTAINER_PREFIX}-}"
        local status="stopped"
        local port="N/A"
        local url="N/A"
        
        if docker ps --format "{{.Names}}" | grep -q "^${container_name}$"; then
            status="running"
            port=$(docker port "$container_name" 8081/tcp 2>/dev/null | cut -d: -f2)
            if [[ -n "$port" ]]; then
                url="http://localhost:${port}"
            fi
        fi
        
        printf "%-20s %-10s %-10s %-30s\n" "$instance_name" "$status" "$port" "$url"
    done <<< "$containers"
    
    return 0
}