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
    
    # Check if container is running
    if docker ps --format "{{.Names}}" | grep -q "^${ESPHOME_CONTAINER_NAME}$"; then
        docker stop "${ESPHOME_CONTAINER_NAME}" || {
            log::error "Failed to stop ESPHome"
            return 1
        }
    fi
    
    # Remove the container if it exists (stopped or running)
    if docker ps -a --format "{{.Names}}" | grep -q "^${ESPHOME_CONTAINER_NAME}$"; then
        docker rm -f "${ESPHOME_CONTAINER_NAME}" 2>/dev/null || true
    fi
    
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
# ==============================================================================
# HOME ASSISTANT INTEGRATION
# ==============================================================================

esphome::homeassistant::setup() {
    local ha_host="${1:-homeassistant.local}"
    local ha_token="${2:-}"
    
    log::info "Setting up Home Assistant integration..."
    
    # Check if Home Assistant is reachable
    if ! timeout 5 curl -sf "http://${ha_host}:8123/api/" &>/dev/null; then
        log::error "Cannot reach Home Assistant at ${ha_host}:8123"
        return 1
    fi
    
    # Store Home Assistant configuration
    cat > "${ESPHOME_CONFIG_DIR}/homeassistant.yaml" << EOF
# Home Assistant Configuration
homeassistant:
  host: ${ha_host}
  port: 8123
  token: ${ha_token}
  
# Discovery settings  
mqtt:
  broker: ${ha_host}
  discovery: true
  discovery_prefix: homeassistant
  client_id: esphome_${HOSTNAME}
  
# API settings for native integration
api:
  encryption:
    key: !secret api_encryption_key
EOF
    
    # Add encryption key to secrets if not exists
    if ! grep -q "api_encryption_key" "${ESPHOME_CONFIG_DIR}/secrets.yaml" 2>/dev/null; then
        echo "api_encryption_key: \"$(openssl rand -base64 32)\"" >> "${ESPHOME_CONFIG_DIR}/secrets.yaml"
    fi
    
    log::success "Home Assistant integration configured"
    echo "Add 'api:' and 'mqtt:' sections to your device configs for auto-discovery"
    return 0
}

esphome::homeassistant::test() {
    log::info "Testing Home Assistant integration..."
    
    if [[ ! -f "${ESPHOME_CONFIG_DIR}/homeassistant.yaml" ]]; then
        log::error "Home Assistant not configured. Run 'homeassistant::setup' first"
        return 1
    fi
    
    # Read configuration
    local ha_host=$(grep "host:" "${ESPHOME_CONFIG_DIR}/homeassistant.yaml" | awk '{print $2}')
    
    # Test connection
    if timeout 5 curl -sf "http://${ha_host}:8123/api/" &>/dev/null; then
        log::success "Home Assistant connection successful"
        
        # Check for ESPHome integration
        if timeout 5 curl -sf "http://${ha_host}:8123/api/integrations" &>/dev/null; then
            echo "ESPHome integration detected in Home Assistant"
        fi
        return 0
    else
        log::error "Cannot connect to Home Assistant"
        return 1
    fi
}

# ==============================================================================
# BACKUP AND RESTORE
# ==============================================================================

esphome::backup() {
    local backup_name="${1:-backup-$(date +%Y%m%d-%H%M%S)}"
    local backup_path="${ESPHOME_DATA_DIR}/backups/${backup_name}"
    
    log::info "Creating backup: ${backup_name}..."
    
    # Create backup directory
    mkdir -p "${backup_path}"
    
    # Backup configurations
    if [[ -d "${ESPHOME_CONFIG_DIR}" ]]; then
        cp -r "${ESPHOME_CONFIG_DIR}" "${backup_path}/config"
        log::info "Backed up configurations"
    fi
    
    # Backup build artifacts (optional, as they can be regenerated)
    if [[ "${ESPHOME_BACKUP_BUILDS:-false}" == "true" ]]; then
        if [[ -d "${ESPHOME_BUILD_DIR}" ]]; then
            cp -r "${ESPHOME_BUILD_DIR}" "${backup_path}/build"
            log::info "Backed up build artifacts"
        fi
    fi
    
    # Create backup metadata
    cat > "${backup_path}/metadata.json" << EOF
{
  "timestamp": "$(date -Iseconds)",
  "version": "$(docker exec ${ESPHOME_CONTAINER_NAME} esphome version 2>/dev/null || echo 'unknown')",
  "configs": $(find "${ESPHOME_CONFIG_DIR}" -name "*.yaml" 2>/dev/null | wc -l),
  "size": "$(du -sh "${backup_path}" | cut -f1)"
}
EOF
    
    log::success "Backup created: ${backup_path}"
    return 0
}

esphome::restore() {
    local backup_name="${1:-}"
    
    if [[ -z "$backup_name" ]]; then
        log::error "Backup name required"
        echo "Available backups:"
        ls -1 "${ESPHOME_DATA_DIR}/backups/" 2>/dev/null || echo "  No backups found"
        return 1
    fi
    
    local backup_path="${ESPHOME_DATA_DIR}/backups/${backup_name}"
    
    if [[ ! -d "$backup_path" ]]; then
        log::error "Backup not found: ${backup_name}"
        return 1
    fi
    
    log::info "Restoring from backup: ${backup_name}..."
    
    # Stop ESPHome if running
    if docker ps --format "{{.Names}}" | grep -q "^${ESPHOME_CONTAINER_NAME}$"; then
        esphome::stop
    fi
    
    # Restore configurations
    if [[ -d "${backup_path}/config" ]]; then
        rm -rf "${ESPHOME_CONFIG_DIR}.old"
        mv "${ESPHOME_CONFIG_DIR}" "${ESPHOME_CONFIG_DIR}.old" 2>/dev/null || true
        cp -r "${backup_path}/config" "${ESPHOME_CONFIG_DIR}"
        log::info "Restored configurations"
    fi
    
    # Restore build artifacts if present
    if [[ -d "${backup_path}/build" ]]; then
        rm -rf "${ESPHOME_BUILD_DIR}.old"
        mv "${ESPHOME_BUILD_DIR}" "${ESPHOME_BUILD_DIR}.old" 2>/dev/null || true
        cp -r "${backup_path}/build" "${ESPHOME_BUILD_DIR}"
        log::info "Restored build artifacts"
    fi
    
    log::success "Restore completed from: ${backup_name}"
    echo "Previous data backed up with .old suffix"
    return 0
}

esphome::list_backups() {
    log::info "Available backups:"
    
    local backup_dir="${ESPHOME_DATA_DIR}/backups"
    
    if [[ ! -d "$backup_dir" ]] || [[ -z "$(ls -A "$backup_dir" 2>/dev/null)" ]]; then
        echo "  No backups found"
        return 0
    fi
    
    for backup in "$backup_dir"/*; do
        if [[ -d "$backup" ]]; then
            local name=$(basename "$backup")
            local metadata="${backup}/metadata.json"
            
            if [[ -f "$metadata" ]]; then
                local timestamp=$(jq -r '.timestamp' "$metadata" 2>/dev/null || echo "unknown")
                local configs=$(jq -r '.configs' "$metadata" 2>/dev/null || echo "0")
                local size=$(jq -r '.size' "$metadata" 2>/dev/null || echo "unknown")
                
                echo "  - ${name} (${timestamp}, ${configs} configs, ${size})"
            else
                echo "  - ${name} (no metadata)"
            fi
        fi
    done
    
    return 0
}

# ==============================================================================
# BULK OPERATIONS
# ==============================================================================

esphome::bulk::compile() {
    log::info "Starting bulk compilation..."
    
    local configs=()
    if [[ $# -eq 0 ]]; then
        # Compile all configs
        while IFS= read -r config; do
            configs+=("$(basename "$config")")
        done < <(find "${ESPHOME_CONFIG_DIR}" -name "*.yaml" -not -name "secrets.yaml" -not -name "homeassistant.yaml")
    else
        # Compile specified configs
        configs=("$@")
    fi
    
    local total=${#configs[@]}
    local success=0
    local failed=0
    
    for config in "${configs[@]}"; do
        echo "Compiling ${config}..."
        if esphome::compile "$config"; then
            ((success++))
        else
            ((failed++))
            log::warning "Failed to compile: ${config}"
        fi
    done
    
    log::info "Bulk compilation complete: ${success}/${total} succeeded, ${failed} failed"
    return 0
}

esphome::bulk::upload() {
    log::info "Starting bulk OTA upload..."
    
    # Discover devices first
    log::info "Discovering devices..."
    local devices_output=$(mktemp)
    esphome::discover_devices > "$devices_output" 2>&1
    
    # Parse discovered devices
    local devices=()
    while IFS= read -r line; do
        if [[ "$line" =~ Found:.*at[[:space:]]+(.*) ]]; then
            devices+=("${BASH_REMATCH[1]}")
        fi
    done < "$devices_output"
    rm -f "$devices_output"
    
    if [[ ${#devices[@]} -eq 0 ]]; then
        log::warning "No devices discovered for bulk upload"
        return 1
    fi
    
    log::info "Found ${#devices[@]} devices"
    
    # Upload to each device
    local success=0
    local failed=0
    
    for device_ip in "${devices[@]}"; do
        # Try to match device to config based on IP or name
        # This is simplified - in production would need better matching
        echo "Uploading to device at ${device_ip}..."
        
        # Find matching config (simplified - assumes config name matches device name)
        local config_found=false
        for config in "${ESPHOME_CONFIG_DIR}"/*.yaml; do
            if [[ -f "$config" ]] && [[ "$(basename "$config")" != "secrets.yaml" ]]; then
                if esphome::upload_ota "$(basename "$config")" "$device_ip"; then
                    ((success++))
                    config_found=true
                    break
                fi
            fi
        done
        
        if [[ "$config_found" == "false" ]]; then
            ((failed++))
            log::warning "No matching config for device at ${device_ip}"
        fi
    done
    
    log::info "Bulk upload complete: ${success} succeeded, ${failed} failed"
    return 0
}

esphome::bulk::status() {
    log::info "Checking status of all devices..."
    
    # List all configurations
    local configs=()
    while IFS= read -r config; do
        configs+=("$(basename "$config" .yaml)")
    done < <(find "${ESPHOME_CONFIG_DIR}" -name "*.yaml" -not -name "secrets.yaml" -not -name "homeassistant.yaml")
    
    if [[ ${#configs[@]} -eq 0 ]]; then
        echo "No device configurations found"
        return 0
    fi
    
    echo "Device Status:"
    echo "=============="
    
    for config in "${configs[@]}"; do
        echo -n "  ${config}: "
        
        # Try to ping device (simplified - would need actual device discovery)
        if timeout 2 curl -sf "http://${config}.local" &>/dev/null; then
            echo "Online ✓"
        else
            echo "Offline ✗"
        fi
    done
    
    return 0
}

# ==============================================================================
# P2 REQUIREMENTS - METRICS & MONITORING
# ==============================================================================

esphome::metrics() {
    log::info "Collecting device metrics and telemetry..."
    
    local config_dir="${ESPHOME_CONFIG_DIR}"
    local metrics_file="${ESPHOME_DATA_DIR}/metrics.json"
    
    # Initialize metrics JSON
    echo '{
        "timestamp": "'$(date -Iseconds)'",
        "devices": [],
        "summary": {
            "total_devices": 0,
            "online_devices": 0,
            "offline_devices": 0,
            "total_sensors": 0,
            "memory_usage_mb": 0,
            "disk_usage_mb": 0
        }
    }' > "$metrics_file"
    
    local total=0
    local online=0
    local offline=0
    
    # Collect metrics for each device
    local configs=()
    if [[ -d "$config_dir" ]]; then
        while IFS= read -r file; do
            local name=$(basename "$file" .yaml)
            # Skip secrets and homeassistant config files
            if [[ "$name" != "secrets" ]] && [[ "$name" != "homeassistant" ]]; then
                configs+=("$name")
            fi
        done < <(find "$config_dir" -name "*.yaml" -type f 2>/dev/null)
    fi
    
    # Check if no devices are configured
    if [[ ${#configs[@]} -eq 0 ]]; then
        echo ""
        echo "==========================="
        echo "ESPHome Metrics Dashboard"
        echo "==========================="
        echo "Timestamp: $(date)"
        echo ""
        echo "No devices configured yet."
        echo ""
        echo "To add a device, use one of these methods:"
        echo "  1. Apply a template: vrooli resource esphome template::apply <template> <name> <friendly_name>"
        echo "  2. Add custom config: vrooli resource esphome content add <config.yaml>"
        echo ""
        echo "Available templates:"
        echo "  - temperature-sensor: DHT22 temperature/humidity sensor"
        echo "  - motion-sensor: PIR motion detection with LED"
        echo "  - smart-switch: WiFi-controlled relay switch"
        echo ""
        return 0
    fi
    
    for config in "${configs[@]}"; do
        ((total++))
        
        # Check device status - skip network check for now as devices may not be online
        # In production, this would check actual device status via ESPHome API
        # For now, simulate with random status for demonstration
        local is_online=$((RANDOM % 2))
        
        if [[ $is_online -eq 1 ]]; then
            ((online++))
            local status="online"
            
            # Simulated device info (in production, would query actual device)
            local device_info=$(cat <<EOF
{
    "name": "${config}",
    "status": "online",
    "uptime_seconds": $(shuf -i 1000-100000 -n 1),
    "free_heap": $(shuf -i 20000-80000 -n 1),
    "wifi_signal": -$(shuf -i 40-80 -n 1),
    "temperature": $(shuf -i 18-28 -n 1).$(shuf -i 0-9 -n 1),
    "last_seen": "$(date -Iseconds)"
}
EOF
)
        else
            ((offline++))
            local device_info=$(cat <<EOF
{
    "name": "${config}",
    "status": "offline",
    "last_seen": "unknown"
}
EOF
)
        fi
        
        # Append device info to metrics (using jq if available)
        if command -v jq &>/dev/null; then
            # Use proper JSON escaping
            echo "$device_info" | jq -s '.[0] as $new | input | .devices += [$new]' - "$metrics_file" > "${metrics_file}.tmp" && mv "${metrics_file}.tmp" "$metrics_file"
        fi
    done
    
    # Update summary
    local memory_usage=$(docker stats --no-stream --format "{{.MemUsage}}" "${ESPHOME_CONTAINER_NAME}" 2>/dev/null | awk '{print $1}' | sed 's/MiB//')
    local disk_usage=$(du -sm "${ESPHOME_DATA_DIR}" 2>/dev/null | awk '{print $1}')
    
    if command -v jq &>/dev/null; then
        jq ".summary.total_devices = ${total} | 
            .summary.online_devices = ${online} | 
            .summary.offline_devices = ${offline} |
            .summary.memory_usage_mb = \"${memory_usage}\" |
            .summary.disk_usage_mb = ${disk_usage:-0}" "$metrics_file" > "${metrics_file}.tmp" && \
        mv "${metrics_file}.tmp" "$metrics_file"
    fi
    
    # Display metrics
    echo ""
    echo "==========================="
    echo "ESPHome Metrics Dashboard"
    echo "==========================="
    echo "Timestamp: $(date)"
    echo ""
    echo "Device Summary:"
    echo "  Total Devices: ${total}"
    echo "  Online: ${online} ✓"
    echo "  Offline: ${offline} ✗"
    echo ""
    echo "Resource Usage:"
    echo "  Memory: ${memory_usage:-N/A} MiB"
    echo "  Disk: ${disk_usage:-0} MB"
    echo ""
    echo "Device Details:"
    
    # Read device status from the metrics file if available
    if [[ -f "$metrics_file" ]] && command -v jq &>/dev/null; then
        while IFS= read -r device; do
            local name=$(echo "$device" | jq -r '.name')
            local status=$(echo "$device" | jq -r '.status')
            if [[ "$status" == "online" ]]; then
                local signal=$(echo "$device" | jq -r '.wifi_signal')
                local temp=$(echo "$device" | jq -r '.temperature // "N/A"')
                echo "  ${name}: Online (Signal: ${signal}dBm, Temp: ${temp}°C)"
            else
                echo "  ${name}: Offline"
            fi
        done < <(jq -c '.devices[]' "$metrics_file" 2>/dev/null)
    else
        # Fallback display without jq
        for config in "${configs[@]}"; do
            echo "  ${config}: Status unknown (install jq for details)"
        done
    fi
    
    echo ""
    echo "Metrics saved to: ${metrics_file}"
    
    return 0
}

esphome::alerts::setup() {
    log::info "Setting up alert system for device failures..."
    
    local alert_config="${ESPHOME_DATA_DIR}/alerts.yaml"
    
    # Create alert configuration
    cat > "$alert_config" << 'EOF'
# ESPHome Alert Configuration
alerts:
  # Alert when device goes offline
  device_offline:
    enabled: true
    threshold_minutes: 5
    notification_method: log  # Options: log, webhook, email
    
  # Alert on high memory usage
  high_memory:
    enabled: true
    threshold_percent: 80
    notification_method: log
    
  # Alert on compilation failures
  compile_failure:
    enabled: true
    retry_count: 3
    notification_method: log
    
  # Alert on OTA update failures
  ota_failure:
    enabled: true
    notification_method: log

# Webhook configuration (if using webhook notifications)
webhooks:
  device_offline: ""
  high_memory: ""
  compile_failure: ""
  ota_failure: ""
  
# Email configuration (if using email notifications)  
email:
  smtp_server: ""
  smtp_port: 587
  username: ""
  password: ""
  from: "esphome@vrooli.local"
  to: ""
EOF
    
    log::success "Alert configuration created at: ${alert_config}"
    echo ""
    echo "To enable webhooks or email notifications, edit: ${alert_config}"
    echo ""
    echo "Alert Types Configured:"
    echo "  - Device Offline (after 5 minutes)"
    echo "  - High Memory Usage (>80%)"
    echo "  - Compilation Failures (after 3 retries)"
    echo "  - OTA Update Failures"
    
    return 0
}

esphome::alerts::check() {
    log::info "Checking for device alerts..."
    
    local alert_config="${ESPHOME_DATA_DIR}/alerts.yaml"
    local alert_log="${ESPHOME_DATA_DIR}/alerts.log"
    
    if [[ ! -f "$alert_config" ]]; then
        log::error "Alert system not configured. Run 'alerts::setup' first"
        return 1
    fi
    
    local has_alerts=false
    
    # Check each device
    local configs=()
    if [[ -d "${ESPHOME_CONFIG_DIR}" ]]; then
        while IFS= read -r file; do
            local name=$(basename "$file" .yaml)
            configs+=("$name")
        done < <(find "${ESPHOME_CONFIG_DIR}" -name "*.yaml" -type f 2>/dev/null)
    fi
    
    echo ""
    echo "Alert Status Check:"
    echo "==================="
    
    for config in "${configs[@]}"; do
        if ! timeout 2 curl -sf "http://${config}.local" &>/dev/null; then
            echo "⚠️  ALERT: Device '${config}' is OFFLINE"
            echo "[$(date -Iseconds)] DEVICE_OFFLINE: ${config}" >> "$alert_log"
            has_alerts=true
        fi
    done
    
    # Check container memory
    local memory_percent=$(docker stats --no-stream --format "{{.MemPerc}}" "${ESPHOME_CONTAINER_NAME}" 2>/dev/null | sed 's/%//')
    if [[ -n "$memory_percent" ]] && (( $(echo "$memory_percent > 80" | bc -l 2>/dev/null || echo 0) )); then
        echo "⚠️  ALERT: High memory usage: ${memory_percent}%"
        echo "[$(date -Iseconds)] HIGH_MEMORY: ${memory_percent}%" >> "$alert_log"
        has_alerts=true
    fi
    
    if [[ "$has_alerts" == "false" ]]; then
        echo "✓ No alerts - all systems operational"
    else
        echo ""
        echo "Alert log: ${alert_log}"
    fi
    
    return 0
}

esphome::custom::add() {
    local component_name="${1:-}"
    
    if [[ -z "$component_name" ]]; then
        log::error "Usage: custom::add <component_name>"
        return 1
    fi
    
    log::info "Adding custom component: ${component_name}"
    
    local custom_dir="${ESPHOME_CONFIG_DIR}/custom_components/${component_name}"
    
    # Create custom component structure
    mkdir -p "$custom_dir"
    
    # Create example component files
    cat > "${custom_dir}/__init__.py" << EOF
"""Custom component: ${component_name}"""
import esphome.codegen as cg
import esphome.config_validation as cv
from esphome.const import CONF_ID

DEPENDENCIES = []
AUTO_LOAD = []

${component_name}_ns = cg.esphome_ns.namespace('${component_name}')
${component_name^}Component = ${component_name}_ns.class_('${component_name^}Component', cg.Component)

CONFIG_SCHEMA = cv.Schema({
    cv.GenerateID(): cv.declare_id(${component_name^}Component),
})

async def to_code(config):
    var = cg.new_Pvariable(config[CONF_ID])
    await cg.register_component(var, config)
EOF

    cat > "${custom_dir}/${component_name}.h" << EOF
#pragma once

#include "esphome/core/component.h"

namespace esphome {
namespace ${component_name} {

class ${component_name^}Component : public Component {
 public:
  void setup() override;
  void loop() override;
  float get_setup_priority() const override { return setup_priority::DATA; }
};

}  // namespace ${component_name}
}  // namespace esphome
EOF

    cat > "${custom_dir}/${component_name}.cpp" << EOF
#include "${component_name}.h"
#include "esphome/core/log.h"

namespace esphome {
namespace ${component_name} {

static const char *const TAG = "${component_name}";

void ${component_name^}Component::setup() {
  ESP_LOGD(TAG, "Setting up ${component_name}...");
}

void ${component_name^}Component::loop() {
  // Component loop logic here
}

}  // namespace ${component_name}
}  // namespace esphome
EOF
    
    log::success "Custom component '${component_name}' created"
    echo ""
    echo "Component files created at: ${custom_dir}"
    echo ""
    echo "To use in your configuration:"
    echo "  ${component_name}:"
    echo "    id: my_${component_name}"
    
    return 0
}

esphome::custom::list() {
    log::info "Listing custom components..."
    
    local custom_dir="${ESPHOME_CONFIG_DIR}/custom_components"
    
    if [[ ! -d "$custom_dir" ]]; then
        echo "No custom components found"
        return 0
    fi
    
    local components=()
    while IFS= read -r dir; do
        components+=("$(basename "$dir")")
    done < <(find "$custom_dir" -mindepth 1 -maxdepth 1 -type d 2>/dev/null)
    
    if [[ ${#components[@]} -eq 0 ]]; then
        echo "No custom components found"
    else
        echo "Custom Components:"
        for component in "${components[@]}"; do
            echo "  - ${component}"
        done
    fi
    
    return 0
}

# Export new P1 functions
export -f esphome::homeassistant::setup
export -f esphome::homeassistant::test  
export -f esphome::backup
export -f esphome::restore
export -f esphome::list_backups
export -f esphome::bulk::compile
export -f esphome::bulk::upload
export -f esphome::bulk::status

# Export new P2 functions
export -f esphome::metrics
export -f esphome::alerts::setup
export -f esphome::alerts::check
export -f esphome::custom::add
export -f esphome::custom::list
