#!/usr/bin/env bash
################################################################################
# ESPHome Core Library
################################################################################

set -euo pipefail

# ==============================================================================
# LIFECYCLE MANAGEMENT
# ==============================================================================

esphome::install() {
    log::info "Installing ESPHome resource..."
    
    # Create required directories
    local dirs=(
        "${ESPHOME_DATA_DIR}"
        "${ESPHOME_CONFIG_DIR}"
        "${ESPHOME_BUILD_DIR}"
        "${ESPHOME_CACHE_DIR}"
    )
    
    for dir in "${dirs[@]}"; do
        if [[ ! -d "$dir" ]]; then
            log::info "Creating directory: $dir"
            mkdir -p "$dir"
        fi
    done
    
    # Pull Docker image
    log::info "Pulling ESPHome Docker image: ${ESPHOME_IMAGE}"
    docker pull "${ESPHOME_IMAGE}" || {
        log::error "Failed to pull ESPHome image"
        return 1
    }
    
    # Create example configuration
    if [[ ! -f "${ESPHOME_CONFIG_DIR}/example.yaml" ]]; then
        cat > "${ESPHOME_CONFIG_DIR}/example.yaml" << 'EOF'
# Example ESPHome configuration
esphome:
  name: example-device

esp32:
  board: esp32dev
  framework:
    type: arduino

wifi:
  ssid: "YourWiFiSSID"
  password: "YourWiFiPassword"

# Enable logging
logger:

# Enable OTA updates
ota:
  - platform: esphome
    password: "vrooli_ota"

# Enable Home Assistant API
api:

# Web server for status
web_server:
  port: 80
EOF
        log::info "Created example configuration"
    fi
    
    log::success "ESPHome installed successfully"
    return 0
}

esphome::uninstall() {
    log::info "Uninstalling ESPHome resource..."
    
    # Stop container if running
    esphome::stop
    
    # Remove container
    if docker ps -a --format "{{.Names}}" | grep -q "^${ESPHOME_CONTAINER_NAME}$"; then
        log::info "Removing container: ${ESPHOME_CONTAINER_NAME}"
        docker rm -f "${ESPHOME_CONTAINER_NAME}" 2>/dev/null || true
    fi
    
    # Optionally remove data (prompt user)
    if [[ -d "${ESPHOME_DATA_DIR}" ]]; then
        log::warning "Data directory exists: ${ESPHOME_DATA_DIR}"
        echo "Remove data directory? (y/N): "
        read -r response
        if [[ "$response" =~ ^[Yy]$ ]]; then
            rm -rf "${ESPHOME_DATA_DIR}"
            log::info "Data directory removed"
        fi
    fi
    
    log::success "ESPHome uninstalled"
    return 0
}

esphome::start() {
    log::info "Starting ESPHome..."
    
    # Check if already running
    if docker ps --format "{{.Names}}" | grep -q "^${ESPHOME_CONTAINER_NAME}$"; then
        log::warning "ESPHome is already running"
        return 0
    fi
    
    # Prepare Docker run command
    local docker_cmd=(
        "docker" "run" "-d"
        "--name" "${ESPHOME_CONTAINER_NAME}"
        "--restart" "${ESPHOME_DOCKER_RESTART}"
        "-p" "${ESPHOME_PORT}:6052"
        "-v" "${ESPHOME_CONFIG_DIR}:/config"
        "-v" "${ESPHOME_BUILD_DIR}:/config/.esphome"
        "-v" "${ESPHOME_CACHE_DIR}:/config/.platformio"
        "--network" "${ESPHOME_DOCKER_NETWORK}"
    )
    
    # Add environment variables
    [[ -n "${ESPHOME_DASHBOARD_PASSWORD}" ]] && docker_cmd+=("-e" "ESPHOME_DASHBOARD_AUTH=${ESPHOME_DASHBOARD_USERNAME}:${ESPHOME_DASHBOARD_PASSWORD}")
    docker_cmd+=("-e" "ESPHOME_DASHBOARD=true")
    
    # Add privileged mode for USB access (optional)
    if [[ "${ESPHOME_USB_ACCESS:-false}" == "true" ]]; then
        docker_cmd+=("--privileged")
        docker_cmd+=("-v" "/dev:/dev")
    fi
    
    # Add the image and command
    docker_cmd+=("${ESPHOME_IMAGE}" "dashboard" "/config")
    
    # Run the container
    log::info "Starting container with command: ${docker_cmd[*]}"
    "${docker_cmd[@]}" || {
        log::error "Failed to start ESPHome container"
        return 1
    }
    
    # Wait for startup if requested
    if [[ "${1:-}" == "--wait" ]]; then
        log::info "Waiting for ESPHome to be ready..."
        local max_attempts=30
        local attempt=0
        
        while [ $attempt -lt $max_attempts ]; do
            if esphome::health_check; then
                log::success "ESPHome is ready"
                return 0
            fi
            sleep 2
            ((attempt++))
        done
        
        log::error "ESPHome failed to become ready"
        return 1
    fi
    
    log::success "ESPHome started"
    return 0
}

esphome::stop() {
    log::info "Stopping ESPHome..."
    
    if ! docker ps --format "{{.Names}}" | grep -q "^${ESPHOME_CONTAINER_NAME}$"; then
        log::warning "ESPHome is not running"
        return 0
    fi
    
    docker stop "${ESPHOME_CONTAINER_NAME}" || {
        log::error "Failed to stop ESPHome"
        return 1
    }
    
    log::success "ESPHome stopped"
    return 0
}

esphome::restart() {
    log::info "Restarting ESPHome..."
    esphome::stop
    sleep 2
    esphome::start "$@"
}

# ==============================================================================
# STATUS AND HEALTH
# ==============================================================================

esphome::status() {
    echo "ESPHome Status"
    echo "=============="
    echo ""
    
    # Check container status
    if docker ps --format "{{.Names}}" | grep -q "^${ESPHOME_CONTAINER_NAME}$"; then
        echo "Container: Running ✓"
        
        # Get container details
        local container_info
        container_info=$(docker inspect "${ESPHOME_CONTAINER_NAME}" 2>/dev/null | jq -r '.[0]')
        
        if [[ -n "$container_info" ]]; then
            echo "  Started: $(echo "$container_info" | jq -r '.State.StartedAt')"
            echo "  Uptime: $(docker ps --format "{{.Status}}" -f "name=${ESPHOME_CONTAINER_NAME}")"
        fi
    else
        echo "Container: Stopped ✗"
    fi
    
    echo ""
    
    # Check health endpoint
    echo -n "Health Check: "
    if esphome::health_check; then
        echo "Healthy ✓"
    else
        echo "Unhealthy ✗"
    fi
    
    echo ""
    
    # Check dashboard accessibility
    echo -n "Dashboard: "
    if timeout 5 curl -sf "${ESPHOME_BASE_URL}" > /dev/null 2>&1; then
        echo "Accessible ✓"
        echo "  URL: ${ESPHOME_BASE_URL}"
    else
        echo "Not accessible ✗"
    fi
    
    echo ""
    
    # Show configuration counts
    if [[ -d "${ESPHOME_CONFIG_DIR}" ]]; then
        local config_count
        config_count=$(find "${ESPHOME_CONFIG_DIR}" -name "*.yaml" -o -name "*.yml" | wc -l)
        echo "Configurations: $config_count YAML files"
    fi
    
    # Show build artifacts
    if [[ -d "${ESPHOME_BUILD_DIR}" ]]; then
        local build_count
        build_count=$(find "${ESPHOME_BUILD_DIR}" -type d -maxdepth 1 | wc -l)
        echo "Build Artifacts: $((build_count - 1)) devices"
    fi
    
    return 0
}

esphome::health_check() {
    # Simple health check - verify container is running and responsive
    if ! docker ps --format "{{.Names}}" | grep -q "^${ESPHOME_CONTAINER_NAME}$"; then
        return 1
    fi
    
    # ESPHome doesn't have /health endpoint, check dashboard availability
    # and return a proper health response
    if timeout 5 curl -sf "${ESPHOME_BASE_URL}" > /dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

# Add a health endpoint wrapper that returns JSON
esphome::health() {
    if esphome::health_check; then
        echo '{"status":"healthy","service":"esphome","dashboard":"accessible"}'
        return 0
    else
        echo '{"status":"unhealthy","service":"esphome","error":"Dashboard not accessible"}'
        return 1
    fi
}

esphome::view_logs() {
    if ! docker ps --format "{{.Names}}" | grep -q "^${ESPHOME_CONTAINER_NAME}$"; then
        log::error "ESPHome is not running"
        return 1
    fi
    
    docker logs "${ESPHOME_CONTAINER_NAME}" --tail 100 -f
}

# ==============================================================================
# CONTENT MANAGEMENT
# ==============================================================================

esphome::add_config() {
    local config_file="${1:-}"
    
    if [[ -z "$config_file" ]]; then
        log::error "Configuration file required"
        echo "Usage: resource-esphome content add <config.yaml>"
        return 1
    fi
    
    if [[ ! -f "$config_file" ]]; then
        log::error "Configuration file not found: $config_file"
        return 1
    fi
    
    local basename
    basename=$(basename "$config_file")
    
    cp "$config_file" "${ESPHOME_CONFIG_DIR}/${basename}"
    log::success "Added configuration: $basename"
    return 0
}

esphome::list_configs() {
    echo "ESPHome Configurations:"
    echo "======================"
    
    if [[ -d "${ESPHOME_CONFIG_DIR}" ]]; then
        find "${ESPHOME_CONFIG_DIR}" \( -name "*.yaml" -o -name "*.yml" \) -type f | while read -r file; do
            local basename
            basename=$(basename "$file")
            echo "  - $basename"
        done
    else
        echo "No configurations found"
    fi
}

esphome::get_config() {
    local config_name="${1:-}"
    
    if [[ -z "$config_name" ]]; then
        log::error "Configuration name required"
        return 1
    fi
    
    local config_file="${ESPHOME_CONFIG_DIR}/${config_name}"
    
    if [[ ! -f "$config_file" ]]; then
        # Try with .yaml extension
        config_file="${ESPHOME_CONFIG_DIR}/${config_name}.yaml"
    fi
    
    if [[ ! -f "$config_file" ]]; then
        log::error "Configuration not found: $config_name"
        return 1
    fi
    
    cat "$config_file"
}

esphome::remove_config() {
    local config_name="${1:-}"
    
    if [[ -z "$config_name" ]]; then
        log::error "Configuration name required"
        return 1
    fi
    
    local config_file="${ESPHOME_CONFIG_DIR}/${config_name}"
    
    if [[ ! -f "$config_file" ]]; then
        config_file="${ESPHOME_CONFIG_DIR}/${config_name}.yaml"
    fi
    
    if [[ ! -f "$config_file" ]]; then
        log::error "Configuration not found: $config_name"
        return 1
    fi
    
    rm "$config_file"
    log::success "Removed configuration: $config_name"
    
    # Also remove build artifacts
    local device_name
    device_name=$(basename "$config_file" .yaml)
    if [[ -d "${ESPHOME_BUILD_DIR}/${device_name}" ]]; then
        rm -rf "${ESPHOME_BUILD_DIR}/${device_name}"
        log::info "Removed build artifacts for: $device_name"
    fi
    
    return 0
}

esphome::compile() {
    local config_name="${1:-}"
    
    if [[ -z "$config_name" ]]; then
        log::error "Configuration name required"
        echo "Usage: resource-esphome content execute <config.yaml>"
        return 1
    fi
    
    if ! docker ps --format "{{.Names}}" | grep -q "^${ESPHOME_CONTAINER_NAME}$"; then
        log::error "ESPHome is not running"
        return 1
    fi
    
    log::info "Compiling firmware for: $config_name"
    
    docker exec "${ESPHOME_CONTAINER_NAME}" \
        esphome compile "/config/${config_name}" || {
        log::error "Compilation failed"
        return 1
    }
    
    log::success "Firmware compiled successfully"
    return 0
}

# ==============================================================================
# TEMPLATE MANAGEMENT
# ==============================================================================

esphome::list_templates() {
    log::info "Available ESPHome templates:"
    
    local script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    local template_dir="${script_dir}/../templates"
    if [[ ! -d "$template_dir" ]]; then
        log::warning "No templates directory found"
        return 1
    fi
    
    local templates=()
    while IFS= read -r -d '' file; do
        templates+=("$(basename "$file" .yaml)")
    done < <(find "$template_dir" -name "*.yaml" -type f -print0)
    
    if [[ ${#templates[@]} -eq 0 ]]; then
        echo "No templates available"
        return 0
    fi
    
    echo "Templates:"
    echo "=========="
    for template in "${templates[@]}"; do
        echo "  - $template"
    done
    
    return 0
}

esphome::apply_template() {
    local template_name="${1:-}"
    local device_name="${2:-}"
    local friendly_name="${3:-${device_name}}"
    
    if [[ -z "$template_name" ]] || [[ -z "$device_name" ]]; then
        log::error "Template name and device name required"
        echo "Usage: resource-esphome template apply <template> <device_name> [friendly_name]"
        return 1
    fi
    
    local script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    local template_file="${script_dir}/../templates/${template_name}.yaml"
    if [[ ! -f "$template_file" ]]; then
        log::error "Template not found: $template_name"
        esphome::list_templates
        return 1
    fi
    
    local output_file="${ESPHOME_CONFIG_DIR}/${device_name}.yaml"
    
    log::info "Applying template: $template_name -> $device_name"
    
    # Substitute variables in template
    sed -e "s/\${device_name}/$device_name/g" \
        -e "s/\${friendly_name}/$friendly_name/g" \
        "$template_file" > "$output_file"
    
    # Create secrets file if it doesn't exist
    local secrets_file="${ESPHOME_CONFIG_DIR}/secrets.yaml"
    if [[ ! -f "$secrets_file" ]]; then
        cat > "$secrets_file" << 'EOF'
# ESPHome Secrets File
wifi_ssid: "YourWiFiSSID"
wifi_password: "YourWiFiPassword"
ap_password: "vrooli_fallback"
ota_password: "vrooli_ota"
api_encryption_key: "VGhpc0lzQVNlY3JldEtleUZvckVTUEhvbWVBUEk="
EOF
        log::info "Created secrets file with default values"
        log::warning "Please update secrets.yaml with your actual values"
    fi
    
    log::success "Template applied: $output_file"
    echo "Next steps:"
    echo "1. Edit secrets.yaml with your WiFi credentials"
    echo "2. Compile: resource-esphome content execute ${device_name}.yaml"
    echo "3. Flash to device via USB or OTA"
    
    return 0
}

# ==============================================================================
# DEVICE MANAGEMENT
# ==============================================================================

esphome::discover_devices() {
    log::info "Discovering ESP devices on network..."
    
    if ! docker ps --format "{{.Names}}" | grep -q "^${ESPHOME_CONTAINER_NAME}$"; then
        log::error "ESPHome is not running"
        return 1
    fi
    
    # Use mDNS discovery
    docker exec "${ESPHOME_CONTAINER_NAME}" \
        python -c "
import zeroconf
import time

class DeviceListener:
    def __init__(self):
        self.devices = []
    
    def add_service(self, zeroconf, service_type, name):
        info = zeroconf.get_service_info(service_type, name)
        if info:
            print(f'Found: {name} at {info.parsed_addresses()[0] if info.parsed_addresses() else \"unknown\"}')

zc = zeroconf.Zeroconf()
listener = DeviceListener()
browser = zeroconf.ServiceBrowser(zc, '_esphomelib._tcp.local.', listener)
time.sleep(${ESPHOME_DISCOVERY_TIMEOUT})
zc.close()
" 2>/dev/null || {
        log::warning "mDNS discovery failed, trying alternative method..."
        echo "Please check devices manually via the dashboard"
    }
}

esphome::upload_ota() {
    local config_name="${1:-}"
    local device_ip="${2:-}"
    
    if [[ -z "$config_name" ]] || [[ -z "$device_ip" ]]; then
        log::error "Configuration and device IP required"
        echo "Usage: resource-esphome upload <config.yaml> <device_ip>"
        return 1
    fi
    
    if ! docker ps --format "{{.Names}}" | grep -q "^${ESPHOME_CONTAINER_NAME}$"; then
        log::error "ESPHome is not running"
        return 1
    fi
    
    log::info "Uploading firmware to device at $device_ip..."
    
    docker exec "${ESPHOME_CONTAINER_NAME}" \
        esphome upload "/config/${config_name}" --device "$device_ip" || {
        log::error "Upload failed"
        return 1
    }
    
    log::success "Firmware uploaded successfully"
    return 0
}

esphome::monitor_serial() {
    local device="${1:-/dev/ttyUSB0}"
    
    if ! docker ps --format "{{.Names}}" | grep -q "^${ESPHOME_CONTAINER_NAME}$"; then
        log::error "ESPHome is not running"
        return 1
    fi
    
    log::info "Monitoring serial output from: $device"
    
    docker exec -it "${ESPHOME_CONTAINER_NAME}" \
        esphome logs --device "$device" || {
        log::error "Failed to monitor device"
        return 1
    }
}

esphome::validate_config() {
    local config_name="${1:-}"
    
    if [[ -z "$config_name" ]]; then
        log::error "Configuration name required"
        return 1
    fi
    
    if ! docker ps --format "{{.Names}}" | grep -q "^${ESPHOME_CONTAINER_NAME}$"; then
        log::error "ESPHome is not running"
        return 1
    fi
    
    log::info "Validating configuration: $config_name"
    
    docker exec "${ESPHOME_CONTAINER_NAME}" \
        esphome config "/config/${config_name}" || {
        log::error "Validation failed"
        return 1
    }
    
    log::success "Configuration is valid"
    return 0
}

# Add info command for v2.0 compliance
esphome::info() {
    esphome::export_config
    
    cat <<EOF
{
  "resource": "esphome",
  "version": "latest",
  "status": "$(docker ps --format "{{.Names}}" | grep -q "^${ESPHOME_CONTAINER_NAME}$" && echo "running" || echo "stopped")",
  "port": ${ESPHOME_PORT},
  "dashboard_url": "${ESPHOME_BASE_URL}",
  "config": {
    "container_name": "${ESPHOME_CONTAINER_NAME}",
    "image": "${ESPHOME_IMAGE}",
    "data_dir": "${ESPHOME_DATA_DIR}",
    "config_dir": "${ESPHOME_CONFIG_DIR}",
    "build_dir": "${ESPHOME_BUILD_DIR}",
    "parallel_builds": ${ESPHOME_PARALLEL_BUILDS},
    "compile_timeout": ${ESPHOME_COMPILE_TIMEOUT},
    "upload_timeout": ${ESPHOME_UPLOAD_TIMEOUT}
  },
  "capabilities": [
    "firmware-generation",
    "ota-updates",
    "device-discovery",
    "yaml-configuration",
    "web-dashboard"
  ]
}
EOF
}

esphome::clean_build() {
    local config_name="${1:-}"
    
    if [[ -z "$config_name" ]]; then
        log::info "Cleaning all build artifacts..."
        rm -rf "${ESPHOME_BUILD_DIR}"/*
    else
        local device_name
        device_name=$(basename "$config_name" .yaml)
        log::info "Cleaning build artifacts for: $device_name"
        rm -rf "${ESPHOME_BUILD_DIR}/${device_name}"
    fi
    
    log::success "Build artifacts cleaned"
    return 0
}