#!/usr/bin/env bash

# Unstructured.io Installation Functions
# This file contains functions for installing and uninstalling the Unstructured.io service

# Get script directory for relative path resolution
LIB_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${LIB_DIR}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LIB_NETWORK_DIR}/ports.sh"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/system_commands.sh" 2>/dev/null || true

#######################################
# Install Unstructured.io service
#######################################
unstructured_io::install() {
    local force="$1"
    
    echo "$MSG_UNSTRUCTURED_IO_INSTALLING"
    
    # Check Docker availability
    if ! unstructured_io::check_docker; then
        return 1
    fi
    
    # Check if container already exists
    if unstructured_io::container_exists; then
        if [ "$force" != "yes" ]; then
            echo "$MSG_UNSTRUCTURED_IO_ALREADY_INSTALLED"
            return 0
        else
            # Force reinstall - remove existing container
            unstructured_io::uninstall
        fi
    fi
    
    # Check if port is available
    if ports::is_port_in_use "$UNSTRUCTURED_IO_PORT"; then
        echo "$MSG_ERROR_PORT_IN_USE"
        return 1
    fi
    
    # Pull Docker image
    echo "$MSG_PULLING_IMAGE"
    if ! docker pull "$UNSTRUCTURED_IO_IMAGE"; then
        echo "$MSG_IMAGE_PULL_FAILED"
        return 1
    fi
    echo "$MSG_IMAGE_PULL_SUCCESS"
    
    # Create and start container
    if ! unstructured_io::create_container; then
        echo "$MSG_CONTAINER_CREATE_FAILED"
        return 1
    fi
    echo "$MSG_CONTAINER_CREATE_SUCCESS"
    
    # Start the container
    if ! docker start "$UNSTRUCTURED_IO_CONTAINER_NAME"; then
        echo "$MSG_CONTAINER_START_FAILED"
        return 1
    fi
    echo "$MSG_CONTAINER_START_SUCCESS"
    
    # Wait for API to be ready
    if unstructured_io::wait_for_api; then
        echo "$MSG_UNSTRUCTURED_IO_INSTALL_SUCCESS"
        unstructured_io::status
        return 0
    else
        echo "$MSG_UNSTRUCTURED_IO_UNHEALTHY"
        return 1
    fi
}

#######################################
# Create Docker container
#######################################
unstructured_io::create_container() {
    docker create \
        --name "$UNSTRUCTURED_IO_CONTAINER_NAME" \
        --restart unless-stopped \
        -p "${UNSTRUCTURED_IO_PORT}:${UNSTRUCTURED_IO_API_PORT}" \
        --memory="$UNSTRUCTURED_IO_MEMORY_LIMIT" \
        --cpus="$UNSTRUCTURED_IO_CPU_LIMIT" \
        -e UNSTRUCTURED_API_LOG_LEVEL=INFO \
        -e UNSTRUCTURED_API_MAX_REQUEST_SIZE=52428800 \
        -e UNSTRUCTURED_API_WORKERS=1 \
        -e UNSTRUCTURED_API_PORT="$UNSTRUCTURED_IO_API_PORT" \
        "$UNSTRUCTURED_IO_IMAGE"
}

#######################################
# Uninstall Unstructured.io service
#######################################
unstructured_io::uninstall() {
    echo "$MSG_UNINSTALLING"
    
    # Stop container if running
    if unstructured_io::container_running; then
        if docker stop "$UNSTRUCTURED_IO_CONTAINER_NAME"; then
            echo "$MSG_CONTAINER_STOP_SUCCESS"
        else
            echo "⚠️  Failed to stop container"
        fi
    fi
    
    # Remove container if exists
    if unstructured_io::container_exists; then
        if docker rm "$UNSTRUCTURED_IO_CONTAINER_NAME"; then
            echo "$MSG_CONTAINER_REMOVE_SUCCESS"
        else
            echo "⚠️  Failed to remove container"
            return 1
        fi
    fi
    
    echo "$MSG_UNINSTALL_SUCCESS"
    return 0
}

#######################################
# Start Unstructured.io service
#######################################
unstructured_io::start() {
    if ! unstructured_io::container_exists; then
        echo "$MSG_UNSTRUCTURED_IO_NOT_FOUND"
        echo "Run with --action install to set up the service"
        return 1
    fi
    
    if unstructured_io::container_running; then
        echo "Unstructured.io is already running"
        return 0
    fi
    
    if docker start "$UNSTRUCTURED_IO_CONTAINER_NAME"; then
        if unstructured_io::wait_for_api; then
            echo "$MSG_CONTAINER_START_SUCCESS"
            return 0
        else
            echo "$MSG_UNSTRUCTURED_IO_UNHEALTHY"
            return 1
        fi
    else
        echo "$MSG_CONTAINER_START_FAILED"
        return 1
    fi
}

#######################################
# Stop Unstructured.io service
#######################################
unstructured_io::stop() {
    if ! unstructured_io::container_exists; then
        echo "$MSG_UNSTRUCTURED_IO_NOT_FOUND"
        return 1
    fi
    
    if ! unstructured_io::container_running; then
        echo "Unstructured.io is not running"
        return 0
    fi
    
    if docker stop "$UNSTRUCTURED_IO_CONTAINER_NAME"; then
        echo "$MSG_CONTAINER_STOP_SUCCESS"
        return 0
    else
        echo "❌ Failed to stop container"
        return 1
    fi
}

#######################################
# Restart Unstructured.io service
#######################################
unstructured_io::restart() {
    echo "Restarting Unstructured.io..."
    
    if unstructured_io::stop; then
        sleep 2
        if unstructured_io::start; then
            echo "✅ Unstructured.io restarted successfully"
            return 0
        fi
    fi
    
    return 1
}

# Export functions for subshell availability
export -f unstructured_io::install
export -f unstructured_io::create_container
export -f unstructured_io::uninstall
export -f unstructured_io::start
export -f unstructured_io::stop
export -f unstructured_io::restart