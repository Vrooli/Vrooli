#!/usr/bin/env bash
# Node-RED Docker Management Functions
# All Docker-related operations for Node-RED containers

#######################################
# Build custom Node-RED Docker image
#######################################
node_red::build_custom_image() {
    node_red::show_building_image
    
    # Check if Dockerfile exists in docker directory
    local docker_dir="$SCRIPT_DIR/docker"
    if [[ ! -f "$docker_dir/Dockerfile" ]]; then
        node_red::show_docker_build_error
        return 1
    fi
    
    # Build the image
    if docker build -t "$IMAGE_NAME" "$docker_dir"; then
        log::success "Custom image built successfully"
        return 0
    else
        node_red::show_build_failed_error
        return 1
    fi
}

#######################################
# Create Docker network if it doesn't exist
#######################################
node_red::create_network() {
    if ! docker network inspect "$NETWORK_NAME" >/dev/null 2>&1; then
        node_red::show_creating_network
        docker network create "$NETWORK_NAME"
    fi
}

#######################################
# Create Docker volume for Node-RED data
#######################################
node_red::create_volume() {
    node_red::show_creating_volume
    docker volume create "$VOLUME_NAME"
}

#######################################
# Build Node-RED Docker run command
#######################################
node_red::build_docker_command() {
    local image_to_use="$1"
    
    local docker_cmd="docker run -d"
    docker_cmd+=" --name $CONTAINER_NAME"
    docker_cmd+=" --restart unless-stopped"
    docker_cmd+=" --network $NETWORK_NAME"
    docker_cmd+=" -p $RESOURCE_PORT:1880"
    docker_cmd+=" -v $VOLUME_NAME:/data"
    docker_cmd+=" -v $SCRIPT_DIR/flows:/data/flows"
    docker_cmd+=" -v $SCRIPT_DIR/settings.js:/data/settings.js:ro"
    
    # Add host access volumes if enabled
    if [[ "$NODE_RED_ENABLE_HOST_ACCESS" == "yes" ]]; then
        # Host system binaries (read-only)
        if [[ -d /usr/bin ]]; then
            docker_cmd+=" -v /usr/bin:/host/usr/bin:ro"
        fi
        if [[ -d /bin ]]; then
            docker_cmd+=" -v /bin:/host/bin:ro"
        fi
        
        # Workspace access (read-write)
        docker_cmd+=" -v $HOME/Vrooli:/workspace:rw"
    fi
    
    # Add Docker socket access if enabled
    if [[ "$NODE_RED_ENABLE_DOCKER_SOCKET" == "yes" ]] && [[ -S /var/run/docker.sock ]]; then
        docker_cmd+=" -v /var/run/docker.sock:/var/run/docker.sock:ro"
    fi
    
    # Environment variables
    docker_cmd+=" -e NODE_RED_FLOW_FILE=$DEFAULT_FLOW_FILE"
    docker_cmd+=" -e NODE_RED_CREDENTIAL_SECRET=$DEFAULT_SECRET"
    docker_cmd+=" -e TZ=${TZ:-UTC}"
    
    # Health check
    docker_cmd+=" --health-cmd=\"curl -f http://localhost:1880 || exit 1\""
    docker_cmd+=" --health-interval=$DOCKER_HEALTH_INTERVAL"
    docker_cmd+=" --health-timeout=$DOCKER_HEALTH_TIMEOUT"
    docker_cmd+=" --health-retries=$DOCKER_HEALTH_RETRIES"
    
    # Add the image name
    docker_cmd+=" $image_to_use"
    
    echo "$docker_cmd"
}

#######################################
# Start Node-RED container
#######################################
node_red::start_container() {
    local build_image="${1:-$NODE_RED_ENABLE_CUSTOM_IMAGE}"
    
    # Determine which image to use
    local image_to_use
    if [[ "$build_image" == "yes" ]]; then
        node_red::build_custom_image || return 1
        image_to_use="$IMAGE_NAME"
    else
        image_to_use="$OFFICIAL_IMAGE"
    fi
    
    # Create network if it doesn't exist
    node_red::create_network
    
    # Create volume
    node_red::create_volume
    
    # Create settings.js if it doesn't exist
    node_red::create_default_settings
    
    # Build and execute Docker command
    node_red::show_starting_container
    local docker_cmd
    docker_cmd=$(node_red::build_docker_command "$image_to_use")
    
    # Execute the command
    eval "$docker_cmd"
}

#######################################
# Stop Node-RED container
#######################################
node_red::stop_container() {
    if ! node_red::is_installed; then
        node_red::show_not_installed_error
        return 1
    fi
    
    if ! node_red::is_running; then
        node_red::show_already_running_warning
        return 0
    fi
    
    node_red::show_stopping
    docker stop "$CONTAINER_NAME"
    log::success "Node-RED stopped"
}

#######################################
# Start existing Node-RED container
#######################################
node_red::start_existing_container() {
    if ! node_red::is_installed; then
        node_red::show_not_installed_error
        return 1
    fi
    
    if node_red::is_running; then
        node_red::show_already_running_warning
        return 0
    fi
    
    node_red::show_starting
    docker start "$CONTAINER_NAME"
    
    if node_red::wait_for_ready; then
        log::success "Node-RED started successfully"
        return 0
    else
        return 1
    fi
}

#######################################
# Restart Node-RED container
#######################################
node_red::restart_container() {
    node_red::show_restarting
    node_red::stop_container
    node_red::start_existing_container
}

#######################################
# Remove Node-RED container
#######################################
node_red::remove_container() {
    if ! node_red::is_installed; then
        return 0  # Already removed
    fi
    
    # Stop container if running
    if node_red::is_running; then
        docker stop "$CONTAINER_NAME"
    fi
    
    # Remove container
    docker rm "$CONTAINER_NAME"
}

#######################################
# Remove Node-RED volume
#######################################
node_red::remove_volume() {
    if docker volume inspect "$VOLUME_NAME" >/dev/null 2>&1; then
        docker volume rm "$VOLUME_NAME"
    fi
}

#######################################
# Remove custom Node-RED image
#######################################
node_red::remove_custom_image() {
    if docker image inspect "$IMAGE_NAME" >/dev/null 2>&1; then
        docker rmi "$IMAGE_NAME"
    fi
}

#######################################
# Get container information
#######################################
node_red::get_container_info() {
    if ! node_red::is_installed; then
        return 1
    fi
    
    docker container inspect "$CONTAINER_NAME" 2>/dev/null
}

#######################################
# Check if custom image exists
#######################################
node_red::custom_image_exists() {
    docker image inspect "$IMAGE_NAME" >/dev/null 2>&1
}

#######################################
# Pull official Node-RED image
#######################################
node_red::pull_official_image() {
    log::info "Pulling official Node-RED image..."
    docker pull "$OFFICIAL_IMAGE"
}

#######################################
# Validate Docker setup for Node-RED
#######################################
node_red::validate_docker_setup() {
    # Check Docker is available
    if ! node_red::check_docker; then
        return 1
    fi
    
    # Check port availability
    if ! node_red::check_port "$RESOURCE_PORT"; then
        log::error "Port $RESOURCE_PORT is already in use"
        log::info "Stop the service using port $RESOURCE_PORT or use a different port"
        log::info "Example: NODE_RED_CUSTOM_PORT=1881 $0 --action install"
        return 1
    fi
    
    return 0
}

#######################################
# Clean up all Node-RED Docker resources
#######################################
node_red::cleanup_docker_resources() {
    node_red::remove_container
    node_red::remove_volume
    node_red::remove_custom_image
    
    # Clean up orphaned networks if no other containers are using them
    if docker network inspect "$NETWORK_NAME" >/dev/null 2>&1; then
        local containers_using_network
        containers_using_network=$(docker network inspect "$NETWORK_NAME" --format '{{len .Containers}}' 2>/dev/null || echo "0")
        if [[ "$containers_using_network" == "0" ]]; then
            docker network rm "$NETWORK_NAME" 2>/dev/null || true
        fi
    fi
}

#######################################
# Execute command inside Node-RED container
#######################################
node_red::exec_in_container() {
    local cmd="$1"
    
    if ! node_red::is_running; then
        node_red::show_not_running_error
        return 1
    fi
    
    docker exec "$CONTAINER_NAME" $cmd
}

#######################################
# Copy files to/from Node-RED container
#######################################
node_red::copy_to_container() {
    local src="$1"
    local dest="$2"
    
    if ! node_red::is_running; then
        node_red::show_not_running_error
        return 1
    fi
    
    docker cp "$src" "$CONTAINER_NAME:$dest"
}

node_red::copy_from_container() {
    local src="$1"
    local dest="$2"
    
    if ! node_red::is_running; then
        node_red::show_not_running_error
        return 1
    fi
    
    docker cp "$CONTAINER_NAME:$src" "$dest"
}

#######################################
# Get Docker resource usage stats
#######################################
node_red::get_docker_stats() {
    if ! node_red::is_running; then
        return 1
    fi
    
    docker stats --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}\t{{.BlockIO}}" "$CONTAINER_NAME"
}

#######################################
# Inspect Node-RED container network
#######################################
node_red::inspect_network() {
    docker network inspect "$NETWORK_NAME" 2>/dev/null
}