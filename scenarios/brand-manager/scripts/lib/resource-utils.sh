#!/bin/bash
# Brand Manager Resource Utilities
# Shared functions for port and hostname resolution

# Get resource port helper function
get_resource_port() {
    local resource_name="$1"
    local port
    port=$(bash -c "source ${VROOLI_ROOT:-${HOME}/Vrooli}/scripts/resources/port_registry.sh && ports::get_resource_port $resource_name" 2>/dev/null || echo "")
    if [[ -n "$port" ]]; then
        echo "$port"
    else
        # Fallback to defaults from resource-urls.json
        case "$resource_name" in
            postgres) echo "5432" ;;
            minio) echo "9000" ;;
            n8n) echo "5678" ;;
            ollama) echo "11434" ;;
            comfyui) echo "8188" ;;
            vault) echo "8200" ;;
            brand-manager) echo "8090" ;;
            *) echo "8080" ;;
        esac
    fi
}

# Get resource hostname helper function
get_resource_hostname() {
    local resource_name="$1"
    # Check if we're running in a containerized environment
    if [[ "${CONTAINER_ENV:-false}" == "true" ]] || [[ -n "${DOCKER_HOST:-}" ]]; then
        # Use Docker service names
        echo "$resource_name"
    else
        # Use localhost for local development
        echo "localhost"
    fi
}

# Get full service URL
get_service_url() {
    local service_name="$1"
    local endpoint="${2:-}"
    local hostname=$(get_resource_hostname "$service_name")
    local port=$(get_resource_port "$service_name")
    
    if [[ -n "$endpoint" ]]; then
        echo "http://$hostname:$port$endpoint"
    else
        echo "http://$hostname:$port"
    fi
}

# Export functions for use in other scripts
export -f get_resource_port
export -f get_resource_hostname  
export -f get_service_url
