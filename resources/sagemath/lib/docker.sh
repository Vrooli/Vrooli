#!/usr/bin/env bash
################################################################################
# SageMath Docker Operations
# 
# Docker lifecycle management for SageMath resource
################################################################################

# Universal Contract v2.0 handler for manage::start
sagemath::docker::start() {
    local verbose="${1:-}"
    
    echo "Starting SageMath..."
    
    # Check if container exists
    if ! sagemath_container_exists; then
        echo "Container doesn't exist. Installing first..."
        sagemath::install::execute
    fi
    
    # Check if already running
    if sagemath_container_running; then
        echo "SageMath is already running"
        return 0
    fi
    
    # Start container
    if docker start "$SAGEMATH_CONTAINER_NAME"; then
        echo "Waiting for SageMath to be ready..."
        
        # Wait for Jupyter to be available
        local max_attempts=30
        local attempt=0
        
        while [ $attempt -lt $max_attempts ]; do
            if curl -s "http://localhost:${SAGEMATH_PORT_JUPYTER}" >/dev/null 2>&1; then
                echo "SageMath is ready!"
                
                # Get Jupyter token
                local token=$(docker logs "$SAGEMATH_CONTAINER_NAME" 2>&1 | grep -o 'token=[a-z0-9]*' | tail -1 | cut -d= -f2)
                if [ -n "$token" ]; then
                    echo "Jupyter notebook URL: http://localhost:${SAGEMATH_PORT_JUPYTER}/?token=${token}"
                fi
                
                return 0
            fi
            sleep 2
            ((attempt++))
        done
        
        echo "Warning: SageMath started but Jupyter interface not responding" >&2
        return 0
    else
        echo "Error: Failed to start SageMath container" >&2
        return 1
    fi
}

# Universal Contract v2.0 handler for manage::stop
sagemath::docker::stop() {
    local verbose="${1:-}"
    
    echo "Stopping SageMath..."
    
    if ! sagemath_container_running; then
        echo "SageMath is not running"
        return 0
    fi
    
    if docker stop "$SAGEMATH_CONTAINER_NAME"; then
        echo "SageMath stopped successfully"
        return 0
    else
        echo "Error: Failed to stop SageMath" >&2
        return 1
    fi
}

# Universal Contract v2.0 handler for manage::restart
sagemath::docker::restart() {
    echo "Restarting SageMath..."
    sagemath::docker::stop "$@"
    sagemath::docker::start "$@"
}

# Universal Contract v2.0 handler for logs command
sagemath::docker::logs() {
    if ! sagemath_container_exists; then
        echo "SageMath container does not exist" >&2
        return 1
    fi
    
    echo "Showing SageMath logs..."
    docker logs "$SAGEMATH_CONTAINER_NAME" "$@"
}