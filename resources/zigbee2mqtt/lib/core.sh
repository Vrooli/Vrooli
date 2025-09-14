#!/usr/bin/env bash
################################################################################
# Zigbee2MQTT Core Library
# 
# Core functionality for managing Zigbee2MQTT bridge
################################################################################

set -euo pipefail

# Configuration
ZIGBEE2MQTT_VERSION="${ZIGBEE2MQTT_VERSION:-latest}"
ZIGBEE2MQTT_PORT="${ZIGBEE2MQTT_PORT:-8090}"
ZIGBEE2MQTT_CONTAINER="${ZIGBEE2MQTT_CONTAINER:-zigbee2mqtt}"
ZIGBEE2MQTT_DATA_DIR="${ZIGBEE2MQTT_DATA_DIR:-${VROOLI_ROOT:-${HOME}/Vrooli}/data/zigbee2mqtt}"

# MQTT Configuration
MQTT_HOST="${MQTT_HOST:-localhost}"
MQTT_PORT="${MQTT_PORT:-1883}"
MQTT_USER="${MQTT_USER:-}"
MQTT_PASS="${MQTT_PASS:-}"

# Zigbee adapter configuration
ZIGBEE_ADAPTER="${ZIGBEE_ADAPTER:-/dev/ttyACM0}"
ZIGBEE_ADAPTER_TYPE="${ZIGBEE_ADAPTER_TYPE:-auto}"

# Source port registry for consistent port allocation
source "${VROOLI_ROOT:-${HOME}/Vrooli}/scripts/resources/port_registry.sh"

################################################################################
# Core Functions
################################################################################

# Display resource information
zigbee2mqtt::info() {
    cat << EOF
Resource: Zigbee2MQTT
Version: ${ZIGBEE2MQTT_VERSION}
Description: Zigbee to MQTT bridge supporting 3000+ devices

Configuration:
  API Port: ${ZIGBEE2MQTT_PORT}
  MQTT Host: ${MQTT_HOST}:${MQTT_PORT}
  Data Directory: ${ZIGBEE2MQTT_DATA_DIR}
  Zigbee Adapter: ${ZIGBEE_ADAPTER}
  Container: ${ZIGBEE2MQTT_CONTAINER}

Dependencies:
  - MQTT Broker (mosquitto or similar)
  - Zigbee USB Adapter (CC2531/CC2652/ConBee)
  - Docker

Integration Points:
  - Home Assistant (via MQTT discovery)
  - Node-RED (via MQTT)
  - Grafana (metrics visualization)
EOF
}

# Check Zigbee2MQTT status
zigbee2mqtt::status() {
    local verbose="${1:-}"
    
    if docker ps --format "{{.Names}}" | grep -q "^${ZIGBEE2MQTT_CONTAINER}$"; then
        echo "Status: Running"
        
        if [[ "$verbose" == "--verbose" ]] || [[ "$verbose" == "-v" ]]; then
            echo ""
            echo "Container Details:"
            docker ps --filter "name=${ZIGBEE2MQTT_CONTAINER}" --format "table {{.Status}}\t{{.Ports}}"
            
            echo ""
            echo "Health Check:"
            if timeout 5 curl -sf "http://localhost:${ZIGBEE2MQTT_PORT}/api/health" &>/dev/null; then
                echo "  API: Healthy"
            else
                echo "  API: Not responding"
            fi
            
            echo ""
            echo "Device Count:"
            local device_count=$(docker exec "${ZIGBEE2MQTT_CONTAINER}" ls /app/data/devices.yaml 2>/dev/null | wc -l || echo "0")
            echo "  Paired Devices: ${device_count}"
        fi
    else
        echo "Status: Stopped"
        return 1
    fi
}

# View logs
zigbee2mqtt::logs() {
    local lines="${1:-50}"
    local follow="${2:-}"
    
    if [[ "$follow" == "--follow" ]] || [[ "$follow" == "-f" ]]; then
        docker logs -f "${ZIGBEE2MQTT_CONTAINER}"
    else
        docker logs --tail "$lines" "${ZIGBEE2MQTT_CONTAINER}"
    fi
}

################################################################################
# Lifecycle Management
################################################################################

# Install Zigbee2MQTT
zigbee2mqtt::install() {
    log::info "Installing Zigbee2MQTT..."
    
    # Create data directory
    mkdir -p "${ZIGBEE2MQTT_DATA_DIR}"
    
    # Check for USB adapter
    if [[ ! -e "${ZIGBEE_ADAPTER}" ]]; then
        log::warning "Zigbee adapter not found at ${ZIGBEE_ADAPTER}"
        log::info "Please connect your Zigbee USB adapter and update ZIGBEE_ADAPTER environment variable"
    fi
    
    # Create initial configuration
    zigbee2mqtt::create_config
    
    # Pull Docker image
    log::info "Pulling Zigbee2MQTT Docker image..."
    docker pull "koenkk/zigbee2mqtt:${ZIGBEE2MQTT_VERSION}"
    
    log::success "Zigbee2MQTT installed successfully"
}

# Start Zigbee2MQTT
zigbee2mqtt::start() {
    local wait="${1:-}"
    
    log::info "Starting Zigbee2MQTT..."
    
    # Check if already running
    if docker ps --format "{{.Names}}" | grep -q "^${ZIGBEE2MQTT_CONTAINER}$"; then
        log::warning "Zigbee2MQTT is already running"
        return 0
    fi
    
    # Ensure MQTT broker is available
    if ! timeout 5 nc -zv "${MQTT_HOST}" "${MQTT_PORT}" &>/dev/null; then
        log::error "MQTT broker not available at ${MQTT_HOST}:${MQTT_PORT}"
        log::info "Please start an MQTT broker first (e.g., mosquitto)"
        return 1
    fi
    
    # Start container
    docker run -d \
        --name "${ZIGBEE2MQTT_CONTAINER}" \
        --restart unless-stopped \
        -p "${ZIGBEE2MQTT_PORT}:8080" \
        -v "${ZIGBEE2MQTT_DATA_DIR}:/app/data" \
        -v /run/udev:/run/udev:ro \
        --device "${ZIGBEE_ADAPTER}:/dev/ttyACM0" \
        -e TZ="${TZ:-UTC}" \
        "koenkk/zigbee2mqtt:${ZIGBEE2MQTT_VERSION}"
    
    if [[ "$wait" == "--wait" ]]; then
        log::info "Waiting for Zigbee2MQTT to be ready..."
        local attempts=0
        while [[ $attempts -lt 30 ]]; do
            if timeout 5 curl -sf "http://localhost:${ZIGBEE2MQTT_PORT}/api/health" &>/dev/null; then
                log::success "Zigbee2MQTT is ready"
                return 0
            fi
            sleep 2
            ((attempts++))
        done
        log::error "Zigbee2MQTT failed to become ready"
        return 1
    fi
    
    log::success "Zigbee2MQTT started"
}

# Stop Zigbee2MQTT
zigbee2mqtt::stop() {
    log::info "Stopping Zigbee2MQTT..."
    
    if docker ps --format "{{.Names}}" | grep -q "^${ZIGBEE2MQTT_CONTAINER}$"; then
        docker stop "${ZIGBEE2MQTT_CONTAINER}"
        docker rm "${ZIGBEE2MQTT_CONTAINER}"
        log::success "Zigbee2MQTT stopped"
    else
        log::warning "Zigbee2MQTT is not running"
    fi
}

# Restart Zigbee2MQTT
zigbee2mqtt::restart() {
    zigbee2mqtt::stop
    sleep 2
    zigbee2mqtt::start "$@"
}

# Uninstall Zigbee2MQTT
zigbee2mqtt::uninstall() {
    log::info "Uninstalling Zigbee2MQTT..."
    
    # Stop container if running
    zigbee2mqtt::stop
    
    # Remove Docker image
    docker rmi "koenkk/zigbee2mqtt:${ZIGBEE2MQTT_VERSION}" 2>/dev/null || true
    
    # Optionally remove data
    read -p "Remove Zigbee2MQTT data directory? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        rm -rf "${ZIGBEE2MQTT_DATA_DIR}"
        log::info "Data directory removed"
    fi
    
    log::success "Zigbee2MQTT uninstalled"
}

################################################################################
# Configuration Management
################################################################################

# Create initial configuration
zigbee2mqtt::create_config() {
    local config_file="${ZIGBEE2MQTT_DATA_DIR}/configuration.yaml"
    
    if [[ -f "$config_file" ]]; then
        log::info "Configuration already exists, skipping creation"
        return 0
    fi
    
    cat > "$config_file" << EOF
# Zigbee2MQTT configuration
# See https://www.zigbee2mqtt.io/guide/configuration/

# MQTT settings
mqtt:
  base_topic: zigbee2mqtt
  server: mqtt://${MQTT_HOST}:${MQTT_PORT}
  ${MQTT_USER:+user: ${MQTT_USER}}
  ${MQTT_PASS:+password: ${MQTT_PASS}}

# Serial settings
serial:
  port: /dev/ttyACM0
  adapter: ${ZIGBEE_ADAPTER_TYPE}

# Advanced settings
advanced:
  network_key: GENERATE
  pan_id: GENERATE
  ext_pan_id: [0xDD, 0xDD, 0xDD, 0xDD, 0xDD, 0xDD, 0xDD, 0xDD]
  channel: 11
  
# Frontend
frontend:
  port: 8080
  host: 0.0.0.0

# Permit join (disabled by default for security)
permit_join: false

# Home Assistant integration
homeassistant: true

# Device availability
availability:
  active:
    timeout: 10
  passive:
    timeout: 1500
EOF
    
    log::success "Configuration created at $config_file"
}

################################################################################
# Content Management
################################################################################

# List paired devices
zigbee2mqtt::content::list() {
    if ! docker ps --format "{{.Names}}" | grep -q "^${ZIGBEE2MQTT_CONTAINER}$"; then
        log::error "Zigbee2MQTT is not running"
        return 1
    fi
    
    curl -sf "http://localhost:${ZIGBEE2MQTT_PORT}/api/devices" | jq -r '.[] | "\(.friendly_name) (\(.ieee_address)) - \(.definition.model // "Unknown")"' 2>/dev/null || {
        log::warning "Could not retrieve device list"
        return 1
    }
}

# Add/pair new device
zigbee2mqtt::content::add() {
    zigbee2mqtt::device::pair
}

# Get device configuration
zigbee2mqtt::content::get() {
    local device="${1:-}"
    
    if [[ -z "$device" ]]; then
        log::error "Device name required"
        return 1
    fi
    
    curl -sf "http://localhost:${ZIGBEE2MQTT_PORT}/api/device/${device}" | jq '.' 2>/dev/null || {
        log::error "Could not retrieve device configuration"
        return 1
    }
}

# Remove device
zigbee2mqtt::content::remove() {
    local device="${1:-}"
    
    if [[ -z "$device" ]]; then
        log::error "Device name required"
        return 1
    fi
    
    zigbee2mqtt::device::unpair "$device"
}

# Execute command
zigbee2mqtt::content::execute() {
    local command="${1:-}"
    shift || true
    
    if [[ -z "$command" ]]; then
        log::error "Command required"
        return 1
    fi
    
    # Send command via MQTT
    docker exec "${ZIGBEE2MQTT_CONTAINER}" mosquitto_pub \
        -h "${MQTT_HOST}" \
        -p "${MQTT_PORT}" \
        -t "zigbee2mqtt/bridge/request/${command}" \
        -m "$*"
}

################################################################################
# Device Management
################################################################################

# Enable pairing mode
zigbee2mqtt::device::pair() {
    log::info "Enabling pairing mode for 120 seconds..."
    
    curl -X POST "http://localhost:${ZIGBEE2MQTT_PORT}/api/bridge/request/permit_join" \
        -H "Content-Type: application/json" \
        -d '{"value": true, "time": 120}' || {
        log::error "Failed to enable pairing mode"
        return 1
    }
    
    log::success "Pairing mode enabled. Press the pairing button on your device."
    echo "Pairing will automatically disable after 120 seconds."
}

# Remove device from network
zigbee2mqtt::device::unpair() {
    local device="${1:-}"
    
    if [[ -z "$device" ]]; then
        log::error "Device name required"
        return 1
    fi
    
    log::info "Removing device: $device"
    
    curl -X POST "http://localhost:${ZIGBEE2MQTT_PORT}/api/bridge/request/device/remove" \
        -H "Content-Type: application/json" \
        -d "{\"id\": \"${device}\"}" || {
        log::error "Failed to remove device"
        return 1
    }
    
    log::success "Device removed"
}

# Rename device
zigbee2mqtt::device::rename() {
    local old_name="${1:-}"
    local new_name="${2:-}"
    
    if [[ -z "$old_name" ]] || [[ -z "$new_name" ]]; then
        log::error "Usage: device rename <old_name> <new_name>"
        return 1
    fi
    
    curl -X POST "http://localhost:${ZIGBEE2MQTT_PORT}/api/bridge/request/device/rename" \
        -H "Content-Type: application/json" \
        -d "{\"from\": \"${old_name}\", \"to\": \"${new_name}\"}" || {
        log::error "Failed to rename device"
        return 1
    }
    
    log::success "Device renamed from $old_name to $new_name"
}

# Configure device
zigbee2mqtt::device::configure() {
    local device="${1:-}"
    shift || true
    
    if [[ -z "$device" ]]; then
        log::error "Device name required"
        return 1
    fi
    
    log::info "Configuring device: $device"
    
    # TODO: Implement device-specific configuration
    log::warning "Device configuration not yet implemented"
}

################################################################################
# Network Management
################################################################################

# Show network map
zigbee2mqtt::network::map() {
    log::info "Generating network map..."
    
    curl -sf "http://localhost:${ZIGBEE2MQTT_PORT}/api/networkmap" | jq '.' || {
        log::error "Failed to generate network map"
        return 1
    }
}

# Backup coordinator
zigbee2mqtt::network::backup() {
    local backup_file="${1:-${ZIGBEE2MQTT_DATA_DIR}/coordinator_backup_$(date +%Y%m%d_%H%M%S).json}"
    
    log::info "Creating coordinator backup..."
    
    curl -sf "http://localhost:${ZIGBEE2MQTT_PORT}/api/bridge/request/backup" \
        -o "$backup_file" || {
        log::error "Failed to create backup"
        return 1
    }
    
    log::success "Backup saved to: $backup_file"
}

# Restore coordinator backup
zigbee2mqtt::network::restore() {
    local backup_file="${1:-}"
    
    if [[ -z "$backup_file" ]] || [[ ! -f "$backup_file" ]]; then
        log::error "Valid backup file required"
        return 1
    fi
    
    log::warning "Restoring coordinator will restart Zigbee2MQTT"
    read -p "Continue? (y/N): " -n 1 -r
    echo
    
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log::info "Restore cancelled"
        return 0
    fi
    
    # TODO: Implement restore functionality
    log::warning "Restore functionality not yet implemented"
}

# Change Zigbee channel
zigbee2mqtt::network::channel() {
    local channel="${1:-}"
    
    if [[ -z "$channel" ]] || [[ ! "$channel" =~ ^[0-9]+$ ]] || [[ $channel -lt 11 ]] || [[ $channel -gt 26 ]]; then
        log::error "Valid channel required (11-26)"
        return 1
    fi
    
    log::warning "Changing channel will restart Zigbee2MQTT and may require re-pairing devices"
    read -p "Continue? (y/N): " -n 1 -r
    echo
    
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log::info "Channel change cancelled"
        return 0
    fi
    
    # Update configuration
    local config_file="${ZIGBEE2MQTT_DATA_DIR}/configuration.yaml"
    sed -i "s/channel: .*/channel: ${channel}/" "$config_file"
    
    # Restart to apply changes
    zigbee2mqtt::restart
    
    log::success "Channel changed to $channel"
}