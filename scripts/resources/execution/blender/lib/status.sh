#!/bin/bash
# Blender status functionality

# Get script directory
BLENDER_STATUS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Only source core.sh if functions aren't already defined
if ! type blender::is_running &>/dev/null; then
    source "${BLENDER_STATUS_DIR}/core.sh"
fi

# Check Blender health
blender::health_check() {
    if ! blender::is_running; then
        return 1
    fi
    
    # Try to run a simple Blender command as health check
    if docker exec "$BLENDER_CONTAINER_NAME" blender --version &>/dev/null; then
        return 0
    fi
    
    return 1
}

# Get Blender version
blender::get_version() {
    if blender::is_running; then
        docker exec "$BLENDER_CONTAINER_NAME" blender --version 2>/dev/null | grep -oP 'Blender \K[0-9.]+' | head -1
    else
        echo "${BLENDER_VERSION}"
    fi
}

# Main status function
blender::status() {
    local verbose="${1:-false}"
    local format="${2:-text}"
    
    # Source format utilities
    source "${BLENDER_STATUS_DIR}/../../../../lib/utils/format.sh"
    
    # Initialize
    blender::init >/dev/null 2>&1
    
    local status="unknown"
    local message=""
    local installed="false"
    local running="false"
    local healthy="false"
    local version=""
    
    # Check installation
    if blender::is_installed; then
        installed="true"
        
        # Check if running
        if blender::is_running; then
            running="true"
            status="running"
            
            # Check health
            if blender::health_check; then
                healthy="true"
                status="healthy"
                message="Blender is running and healthy"
            else
                message="Blender is running but not responding"
            fi
            
            # Get version
            version=$(blender::get_version)
        else
            status="stopped"
            message="Blender is installed but not running"
            version="${BLENDER_VERSION}"
        fi
    else
        status="not_installed"
        message="Blender is not installed"
        version="${BLENDER_VERSION}"
    fi
    
    # Count scripts and outputs
    local script_count=0
    local output_count=0
    
    if [[ -d "$BLENDER_SCRIPTS_DIR" ]]; then
        script_count=$(find "$BLENDER_SCRIPTS_DIR" -name "*.py" -type f 2>/dev/null | wc -l)
    fi
    
    if [[ -d "$BLENDER_OUTPUT_DIR" ]]; then
        output_count=$(find "$BLENDER_OUTPUT_DIR" -type f 2>/dev/null | wc -l)
    fi
    
    # Build output data for format utility
    local -a output_data=(
        "name" "blender"
        "status" "$status"
        "installed" "$installed"
        "running" "$running"
        "health" "$healthy"
        "healthy" "$healthy"
        "version" "$version"
        "port" "$BLENDER_PORT"
        "scripts" "$script_count"
        "outputs" "$output_count"
        "scripts_dir" "$BLENDER_SCRIPTS_DIR"
        "output_dir" "$BLENDER_OUTPUT_DIR"
        "message" "$message"
        "description" "Professional 3D creation suite with Python API"
        "category" "execution"
    )
    
    # Add container info if verbose and running
    if [[ "$verbose" == "true" && "$running" == "true" ]]; then
        local container_info
        container_info=$(docker inspect "$BLENDER_CONTAINER_NAME" 2>/dev/null | jq -r '.[0] | {
            id: .Id[0:12],
            created: .Created,
            image: .Config.Image,
            state: .State.Status
        }' 2>/dev/null)
        
        if [[ -n "$container_info" ]]; then
            local container_id=$(echo "$container_info" | jq -r '.id')
            local container_image=$(echo "$container_info" | jq -r '.image')
            local container_state=$(echo "$container_info" | jq -r '.state')
            
            output_data+=(
                "container_id" "$container_id"
                "container_image" "$container_image"
                "container_state" "$container_state"
            )
        fi
    fi
    
    # Format output using standard formatter
    format::key_value "$format" "${output_data[@]}"
}

# Export functions
export -f blender::health_check
export -f blender::get_version
export -f blender::status