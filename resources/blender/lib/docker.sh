#!/bin/bash
# Blender Docker operations

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
BLENDER_DOCKER_DIR="${APP_ROOT}/resources/blender/lib"

# Only source core.sh if functions aren't already defined
if ! type blender::init &>/dev/null; then
    source "${BLENDER_DOCKER_DIR}/core.sh"
fi

# Restart Blender container
blender::docker::restart() {
    echo "[INFO] Restarting Blender..."
    
    if blender::is_running; then
        blender::stop || {
            echo "[ERROR] Failed to stop Blender during restart"
            return 1
        }
        sleep 2
    fi
    
    blender::start || {
        echo "[ERROR] Failed to start Blender during restart"
        return 1
    }
    
    echo "[SUCCESS] Blender restarted successfully"
    return 0
}

# Show Blender container logs
blender::docker::logs() {
    local tail_lines="${1:-100}"
    
    if ! blender::is_running; then
        echo "[ERROR] Blender container is not running"
        return 1
    fi
    
    echo "[INFO] Showing last $tail_lines lines of Blender logs..."
    docker logs --tail "$tail_lines" "$BLENDER_CONTAINER_NAME" 2>&1 || {
        echo "[ERROR] Failed to get Blender logs"
        return 1
    }
    
    return 0
}

# Check if Blender container exists
blender::docker::exists() {
    docker inspect "$BLENDER_CONTAINER_NAME" &>/dev/null
}

# Remove Blender container and image
blender::docker::uninstall() {
    echo "[INFO] Removing Blender Docker resources..."
    
    # Stop container if running
    if blender::is_running; then
        blender::stop
    fi
    
    # Remove container if exists
    if blender::docker::exists; then
        echo "[INFO] Removing Blender container..."
        docker rm -f "$BLENDER_CONTAINER_NAME" || {
            echo "[WARN] Failed to remove container, continuing..."
        }
    fi
    
    # Remove image if exists
    if docker inspect "vrooli/blender:latest" &>/dev/null; then
        echo "[INFO] Removing Blender image..."
        docker rmi "vrooli/blender:latest" || {
            echo "[WARN] Failed to remove image, continuing..."
        }
    fi
    
    echo "[SUCCESS] Blender Docker resources removed"
    return 0
}

# Export functions
export -f blender::docker::restart
export -f blender::docker::logs
export -f blender::docker::exists
export -f blender::docker::uninstall