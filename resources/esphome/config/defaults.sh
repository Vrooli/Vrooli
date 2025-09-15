#!/usr/bin/env bash
################################################################################
# ESPHome Default Configuration
################################################################################

# Export configuration for ESPHome
esphome::export_config() {
    # Container settings
    export ESPHOME_CONTAINER_NAME="${ESPHOME_CONTAINER_NAME:-esphome}"
    export ESPHOME_IMAGE="${ESPHOME_IMAGE:-esphome/esphome:latest}"
    
    # Port configuration - load from registry
    PORT_REGISTRY="${APP_ROOT:-${VROOLI_ROOT:-${HOME}/Vrooli}}/scripts/resources/port_registry.sh"
    if [[ -f "$PORT_REGISTRY" ]]; then
        source "$PORT_REGISTRY"
        export ESPHOME_PORT="${ESPHOME_PORT:-${RESOURCE_PORTS[esphome]}}"
    fi
    
    # Fail if port not set
    if [[ -z "${ESPHOME_PORT}" ]]; then
        echo "ERROR: ESPHOME_PORT not provided and could not be obtained from registry" >&2
        return 1
    fi
    export ESPHOME_BASE_URL="${ESPHOME_BASE_URL:-http://localhost:${ESPHOME_PORT}}"
    
    # Directory paths
    export ESPHOME_DATA_DIR="${ESPHOME_DATA_DIR:-${HOME}/.vrooli/esphome}"
    export ESPHOME_CONFIG_DIR="${ESPHOME_CONFIG_DIR:-${ESPHOME_DATA_DIR}/config}"
    export ESPHOME_BUILD_DIR="${ESPHOME_BUILD_DIR:-${ESPHOME_DATA_DIR}/build}"
    export ESPHOME_CACHE_DIR="${ESPHOME_CACHE_DIR:-${ESPHOME_DATA_DIR}/.cache}"
    
    # Dashboard settings
    export ESPHOME_DASHBOARD_ENABLED="${ESPHOME_DASHBOARD_ENABLED:-true}"
    export ESPHOME_DASHBOARD_PASSWORD="${ESPHOME_DASHBOARD_PASSWORD:-}"  # Optional password
    export ESPHOME_DASHBOARD_USERNAME="${ESPHOME_DASHBOARD_USERNAME:-admin}"
    
    # OTA settings
    export ESPHOME_OTA_PASSWORD="${ESPHOME_OTA_PASSWORD:-vrooli_ota}"
    export ESPHOME_API_PASSWORD="${ESPHOME_API_PASSWORD:-}"  # Optional API password
    
    # Network settings
    export ESPHOME_MDNS_ENABLED="${ESPHOME_MDNS_ENABLED:-true}"
    export ESPHOME_DISCOVERY_TIMEOUT="${ESPHOME_DISCOVERY_TIMEOUT:-10}"
    
    # Build settings
    export ESPHOME_COMPILE_TIMEOUT="${ESPHOME_COMPILE_TIMEOUT:-300}"  # 5 minutes
    export ESPHOME_UPLOAD_TIMEOUT="${ESPHOME_UPLOAD_TIMEOUT:-120}"    # 2 minutes
    export ESPHOME_PARALLEL_BUILDS="${ESPHOME_PARALLEL_BUILDS:-2}"
    
    # Logging
    export ESPHOME_LOG_LEVEL="${ESPHOME_LOG_LEVEL:-INFO}"
    export ESPHOME_LOG_FILE="${ESPHOME_LOG_FILE:-${ESPHOME_DATA_DIR}/esphome.log}"
    
    # Health check settings
    export ESPHOME_HEALTH_CHECK_INTERVAL="${ESPHOME_HEALTH_CHECK_INTERVAL:-30}"
    export ESPHOME_STARTUP_TIMEOUT="${ESPHOME_STARTUP_TIMEOUT:-60}"
    
    # Docker settings
    export ESPHOME_DOCKER_NETWORK="${ESPHOME_DOCKER_NETWORK:-bridge}"
    export ESPHOME_DOCKER_RESTART="${ESPHOME_DOCKER_RESTART:-unless-stopped}"
    
    # Platform settings
    export ESPHOME_SUPPORTED_PLATFORMS="${ESPHOME_SUPPORTED_PLATFORMS:-ESP32,ESP8266,ESP32-S2,ESP32-S3,ESP32-C3}"
    export ESPHOME_DEFAULT_PLATFORM="${ESPHOME_DEFAULT_PLATFORM:-ESP32}"
    export ESPHOME_DEFAULT_BOARD="${ESPHOME_DEFAULT_BOARD:-esp32dev}"
}

# Validation function
esphome::validate_config() {
    local errors=0
    
    # Check port availability
    if ! [[ "$ESPHOME_PORT" =~ ^[0-9]+$ ]] || [ "$ESPHOME_PORT" -lt 1024 ] || [ "$ESPHOME_PORT" -gt 65535 ]; then
        echo "ERROR: Invalid port number: $ESPHOME_PORT"
        ((errors++))
    fi
    
    # Check required directories exist or can be created
    for dir in "$ESPHOME_DATA_DIR" "$ESPHOME_CONFIG_DIR" "$ESPHOME_BUILD_DIR"; do
        if [[ ! -d "$dir" ]]; then
            mkdir -p "$dir" 2>/dev/null || {
                echo "ERROR: Cannot create directory: $dir"
                ((errors++))
            }
        fi
    done
    
    # Validate timeout values
    for timeout_var in ESPHOME_COMPILE_TIMEOUT ESPHOME_UPLOAD_TIMEOUT ESPHOME_STARTUP_TIMEOUT; do
        timeout_val="${!timeout_var}"
        if ! [[ "$timeout_val" =~ ^[0-9]+$ ]] || [ "$timeout_val" -lt 1 ]; then
            echo "ERROR: Invalid timeout value for $timeout_var: $timeout_val"
            ((errors++))
        fi
    done
    
    return $errors
}

# Get configuration value
esphome::get_config() {
    local key="${1:-}"
    if [[ -z "$key" ]]; then
        echo "ERROR: Configuration key required" >&2
        return 1
    fi
    
    # Convert key to uppercase and prefix with ESPHOME_
    local var_name="ESPHOME_${key^^}"
    var_name="${var_name//-/_}"  # Replace hyphens with underscores
    
    echo "${!var_name:-}"
}

# Display configuration
esphome::show_config() {
    echo "ESPHome Configuration:"
    echo "====================="
    echo "Container Name: ${ESPHOME_CONTAINER_NAME}"
    echo "Image: ${ESPHOME_IMAGE}"
    echo "Port: ${ESPHOME_PORT}"
    echo "Base URL: ${ESPHOME_BASE_URL}"
    echo "Data Directory: ${ESPHOME_DATA_DIR}"
    echo "Config Directory: ${ESPHOME_CONFIG_DIR}"
    echo "Build Directory: ${ESPHOME_BUILD_DIR}"
    echo "Dashboard Enabled: ${ESPHOME_DASHBOARD_ENABLED}"
    echo "mDNS Discovery: ${ESPHOME_MDNS_ENABLED}"
    echo "Parallel Builds: ${ESPHOME_PARALLEL_BUILDS}"
    echo "Log Level: ${ESPHOME_LOG_LEVEL}"
}