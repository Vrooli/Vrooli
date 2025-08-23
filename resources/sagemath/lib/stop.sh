#!/bin/bash

# Stop functions for SageMath

sagemath_stop() {
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