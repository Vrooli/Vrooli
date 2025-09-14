#!/usr/bin/env bash
# Unstructured.io Core Functions

# Get the directory of this script
UNSTRUCTURED_IO_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source all library files
for lib in common api cache-simple process validate status install content; do
    lib_file="${UNSTRUCTURED_IO_LIB_DIR}/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        source "$lib_file"
    fi
done

# Initialize Unstructured.io environment
unstructured_io::core::init() {
    local data_dir="${UNSTRUCTURED_DATA_DIR:-$HOME/.unstructured}"
    
    # Create necessary directories
    mkdir -p "$data_dir"
    mkdir -p "$data_dir/cache"
    mkdir -p "$data_dir/config"
    mkdir -p "$data_dir/results"
    mkdir -p "$data_dir/temp"
}

# Get Unstructured.io credentials for integration (N8n-compatible JSON format)
unstructured_io::core::credentials() {
    # Source credentials utilities
    source "${var_SCRIPTS_RESOURCES_LIB_DIR}/credentials-utils.sh"
    
    credentials::parse_args "$@" || return $?
    
    local status
    status=$(credentials::get_resource_status "$UNSTRUCTURED_IO_CONTAINER_NAME")
    
    local connections_array="[]"
    if [[ "$status" == "running" ]]; then
        local connection_obj
        connection_obj=$(jq -n \
            --arg host "localhost" \
            --argjson port "$UNSTRUCTURED_IO_PORT" \
            --arg path "/general/v0/general" \
            '{host: $host, port: $port, path: $path, ssl: false}')
        
        local metadata_obj
        metadata_obj=$(jq -n \
            --arg description "Document processing service" \
            --arg strategy "$UNSTRUCTURED_IO_DEFAULT_STRATEGY" \
            '{description: $description, default_strategy: $strategy}')
        
        local connection
        connection=$(credentials::build_connection \
            "main" "Unstructured.io API" "httpRequest" \
            "$connection_obj" "{}" "$metadata_obj")
        
        connections_array="[$connection]"
    fi
    
    credentials::format_output "$(credentials::build_response "unstructured-io" "$status" "$connections_array")"
}

# Check if Unstructured.io is installed (Docker container)
unstructured_io::core::is_installed() {
    unstructured_io::container_exists
}

# Check if Unstructured.io is running
unstructured_io::core::is_running() {
    unstructured_io::container_running
}

# Get service version
unstructured_io::core::version() {
    if unstructured_io::container_running; then
        # Try to get version from API
        local health_url="$UNSTRUCTURED_IO_BASE_URL/healthcheck"
        if command -v curl >/dev/null 2>&1; then
            curl -s "$health_url" 2>/dev/null | grep -o '"version":"[^"]*"' | cut -d'"' -f4 2>/dev/null || echo "unknown"
        else
            echo "unknown"
        fi
    else
        # Get version from Docker image
        docker inspect "$UNSTRUCTURED_IO_CONTAINER_NAME" 2>/dev/null | \
            grep -o '"Image":"[^"]*"' | cut -d':' -f2 | sed 's/"$//' || echo "unknown"
    fi
}