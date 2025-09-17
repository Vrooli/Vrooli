#!/usr/bin/env bash
################################################################################
# Zigbee2MQTT Core Library
# 
# Core functionality for managing Zigbee2MQTT bridge
################################################################################

set -euo pipefail

# Source logging utilities
source "${VROOLI_ROOT:-${HOME}/Vrooli}/scripts/lib/utils/log.sh"

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

# Check for MQTT broker availability with helpful guidance
zigbee2mqtt::check_mqtt_broker() {
    log::info "Checking for MQTT broker at ${MQTT_HOST}:${MQTT_PORT}..."
    
    if timeout 5 nc -zv "${MQTT_HOST}" "${MQTT_PORT}" &>/dev/null; then
        log::success "MQTT broker is available"
        return 0
    fi
    
    log::error "MQTT broker not available at ${MQTT_HOST}:${MQTT_PORT}"
    echo ""
    echo "Zigbee2MQTT requires an MQTT broker to function. You have several options:"
    echo ""
    echo "Option 1: Install Mosquitto locally (recommended):"
    echo "  sudo apt-get update && sudo apt-get install -y mosquitto mosquitto-clients"
    echo "  sudo systemctl start mosquitto"
    echo ""
    echo "Option 2: Run Mosquitto in Docker:"
    echo "  docker run -d --name mosquitto -p 1883:1883 eclipse-mosquitto"
    echo ""
    echo "Option 3: Use an existing MQTT broker:"
    echo "  Set MQTT_HOST and MQTT_PORT environment variables:"
    echo "  export MQTT_HOST=your-broker-host"
    echo "  export MQTT_PORT=1883"
    echo ""
    echo "After setting up MQTT, retry: vrooli resource zigbee2mqtt manage start"
    
    return 1
}

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
            if timeout 5 curl -sf "http://localhost:${ZIGBEE2MQTT_PORT}/" &>/dev/null; then
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
    
    # Check for MQTT broker with helpful guidance
    if ! zigbee2mqtt::check_mqtt_broker; then
        return 1
    fi
    
    # Check if adapter exists before adding device flag
    local device_flag=""
    if [[ -e "${ZIGBEE_ADAPTER}" ]]; then
        device_flag="--device ${ZIGBEE_ADAPTER}:/dev/ttyACM0"
        log::info "Using Zigbee adapter: ${ZIGBEE_ADAPTER}"
    else
        log::warning "Zigbee adapter not found at ${ZIGBEE_ADAPTER}, starting in mock mode"
        log::info "You can test configuration but device control won't work without hardware"
    fi
    
    # Start container (with or without device)
    # Port 8090 is for the API, 8080 is for the Web UI
    docker run -d \
        --name "${ZIGBEE2MQTT_CONTAINER}" \
        --restart unless-stopped \
        -p "${ZIGBEE2MQTT_PORT}:8080" \
        -v "${ZIGBEE2MQTT_DATA_DIR}:/app/data" \
        -v /run/udev:/run/udev:ro \
        ${device_flag} \
        -e TZ="${TZ:-UTC}" \
        "koenkk/zigbee2mqtt:${ZIGBEE2MQTT_VERSION}"
    
    if [[ "$wait" == "--wait" ]]; then
        log::info "Waiting for Zigbee2MQTT to be ready..."
        local attempts=0
        while [[ $attempts -lt 30 ]]; do
            if timeout 5 curl -sf "http://localhost:${ZIGBEE2MQTT_PORT}/" &>/dev/null; then
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

# Home Assistant integration with MQTT discovery
homeassistant: true

# Enable MQTT discovery for auto-configuration  
device_options:
  retain: true
  qos: 1

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

# Control device state (on/off)
zigbee2mqtt::device::control() {
    local device="${1:-}"
    local state="${2:-}"
    
    if [[ -z "$device" ]]; then
        log::error "Usage: device control <device> <on|off|toggle>"
        return 1
    fi
    
    if [[ -z "$state" ]]; then
        log::error "State required: on, off, or toggle"
        return 1
    fi
    
    # Validate state
    case "$state" in
        on|off|toggle)
            ;;
        *)
            log::error "Invalid state: $state (use on, off, or toggle)"
            return 1
            ;;
    esac
    
    log::info "Setting $device to $state"
    
    # Send command via API
    local payload
    if [[ "$state" == "toggle" ]]; then
        payload='{"state": "TOGGLE"}'
    else
        payload="{\"state\": \"${state^^}\"}"
    fi
    
    if curl -X POST "http://localhost:${ZIGBEE2MQTT_PORT}/api/device/${device}/set" \
        -H "Content-Type: application/json" \
        -d "$payload" 2>/dev/null | jq -e '.' &>/dev/null; then
        log::success "Device $device set to $state"
    else
        log::error "Failed to control device"
        return 1
    fi
}

# Set device brightness
zigbee2mqtt::device::brightness() {
    local device="${1:-}"
    local brightness="${2:-}"
    
    if [[ -z "$device" ]] || [[ -z "$brightness" ]]; then
        log::error "Usage: device brightness <device> <0-255>"
        return 1
    fi
    
    # Validate brightness value
    if ! [[ "$brightness" =~ ^[0-9]+$ ]] || [[ $brightness -lt 0 ]] || [[ $brightness -gt 255 ]]; then
        log::error "Brightness must be between 0 and 255"
        return 1
    fi
    
    log::info "Setting $device brightness to $brightness"
    
    # Send command via API
    if curl -X POST "http://localhost:${ZIGBEE2MQTT_PORT}/api/device/${device}/set" \
        -H "Content-Type: application/json" \
        -d "{\"brightness\": $brightness}" 2>/dev/null | jq -e '.' &>/dev/null; then
        log::success "Brightness set to $brightness"
    else
        log::error "Failed to set brightness"
        return 1
    fi
}

# Set device color
zigbee2mqtt::device::color() {
    local device="${1:-}"
    local color="${2:-}"
    
    if [[ -z "$device" ]] || [[ -z "$color" ]]; then
        log::error "Usage: device color <device> <hex-color|r,g,b>"
        echo "Examples:"
        echo "  device color living_room_bulb '#FF0000'  # Red"
        echo "  device color bedroom_light '255,0,0'     # Red (RGB)"
        return 1
    fi
    
    local payload
    
    # Check if hex color (starts with #)
    if [[ "$color" =~ ^#[0-9A-Fa-f]{6}$ ]]; then
        payload="{\"color\": {\"hex\": \"$color\"}}"
    # Check if RGB format (r,g,b)
    elif [[ "$color" =~ ^[0-9]+,[0-9]+,[0-9]+$ ]]; then
        IFS=',' read -r r g b <<< "$color"
        if [[ $r -le 255 ]] && [[ $g -le 255 ]] && [[ $b -le 255 ]]; then
            payload="{\"color\": {\"rgb\": \"$r,$g,$b\"}}"
        else
            log::error "RGB values must be between 0 and 255"
            return 1
        fi
    else
        log::error "Invalid color format. Use hex (#RRGGBB) or RGB (r,g,b)"
        return 1
    fi
    
    log::info "Setting $device color to $color"
    
    # Send command via API
    if curl -X POST "http://localhost:${ZIGBEE2MQTT_PORT}/api/device/${device}/set" \
        -H "Content-Type: application/json" \
        -d "$payload" 2>/dev/null | jq -e '.' &>/dev/null; then
        log::success "Color set to $color"
    else
        log::error "Failed to set color"
        return 1
    fi
}

# Set device temperature (color temperature)
zigbee2mqtt::device::temperature() {
    local device="${1:-}"
    local temp="${2:-}"
    
    if [[ -z "$device" ]] || [[ -z "$temp" ]]; then
        log::error "Usage: device temperature <device> <kelvin>"
        echo "Common values: 2700K (warm), 4000K (neutral), 6500K (cool)"
        return 1
    fi
    
    # Remove K suffix if present
    temp="${temp%K}"
    temp="${temp%k}"
    
    # Validate temperature value (typical range 2000-6500)
    if ! [[ "$temp" =~ ^[0-9]+$ ]] || [[ $temp -lt 2000 ]] || [[ $temp -gt 7000 ]]; then
        log::error "Temperature must be between 2000K and 7000K"
        return 1
    fi
    
    log::info "Setting $device color temperature to ${temp}K"
    
    # Convert Kelvin to mired (Zigbee uses mired)
    local mired=$((1000000 / temp))
    
    # Send command via API
    if curl -X POST "http://localhost:${ZIGBEE2MQTT_PORT}/api/device/${device}/set" \
        -H "Content-Type: application/json" \
        -d "{\"color_temp\": $mired}" 2>/dev/null | jq -e '.' &>/dev/null; then
        log::success "Color temperature set to ${temp}K"
    else
        log::error "Failed to set color temperature"
        return 1
    fi
}

################################################################################
# Groups & Scenes Management
################################################################################

# Create a device group
zigbee2mqtt::group::create() {
    local group_name="${1:-}"
    shift || true
    local devices=("$@")
    
    if [[ -z "$group_name" ]] || [[ ${#devices[@]} -eq 0 ]]; then
        log::error "Usage: group create <group_name> <device1> [device2 ...]"
        return 1
    fi
    
    log::info "Creating group: $group_name"
    
    # Build devices array for JSON
    local devices_json=$(printf '"%s",' "${devices[@]}")
    devices_json="[${devices_json%,}]"
    
    curl -X POST "http://localhost:${ZIGBEE2MQTT_PORT}/api/bridge/request/group/add" \
        -H "Content-Type: application/json" \
        -d "{\"friendly_name\": \"${group_name}\", \"devices\": ${devices_json}}" || {
        log::error "Failed to create group"
        return 1
    }
    
    log::success "Group '$group_name' created with ${#devices[@]} devices"
}

# List all groups
zigbee2mqtt::group::list() {
    log::info "Listing groups..."
    
    curl -sf "http://localhost:${ZIGBEE2MQTT_PORT}/api/groups" | jq -r '.[] | "\(.friendly_name) - \(.members | length) devices"' || {
        log::warning "No groups found or failed to retrieve"
        return 1
    }
}

# Control a group
zigbee2mqtt::group::control() {
    local group="${1:-}"
    local state="${2:-}"
    
    if [[ -z "$group" ]] || [[ -z "$state" ]]; then
        log::error "Usage: group control <group_name> <on|off|toggle>"
        return 1
    fi
    
    log::info "Setting group '$group' to $state"
    
    local payload
    if [[ "$state" == "toggle" ]]; then
        payload='{"state": "TOGGLE"}'
    else
        payload="{\"state\": \"${state^^}\"}"
    fi
    
    curl -X POST "http://localhost:${ZIGBEE2MQTT_PORT}/api/group/${group}/set" \
        -H "Content-Type: application/json" \
        -d "$payload" || {
        log::error "Failed to control group"
        return 1
    }
    
    log::success "Group '$group' set to $state"
}

# Remove a group
zigbee2mqtt::group::remove() {
    local group="${1:-}"
    
    if [[ -z "$group" ]]; then
        log::error "Group name required"
        return 1
    fi
    
    log::info "Removing group: $group"
    
    curl -X POST "http://localhost:${ZIGBEE2MQTT_PORT}/api/bridge/request/group/remove" \
        -H "Content-Type: application/json" \
        -d "{\"id\": \"${group}\"}" || {
        log::error "Failed to remove group"
        return 1
    }
    
    log::success "Group '$group' removed"
}

# Create a scene
zigbee2mqtt::scene::create() {
    local scene_name="${1:-}"
    local group="${2:-}"
    
    if [[ -z "$scene_name" ]] || [[ -z "$group" ]]; then
        log::error "Usage: scene create <scene_name> <group_name>"
        echo "Note: Scene will capture current state of all devices in the group"
        return 1
    fi
    
    log::info "Creating scene '$scene_name' for group '$group'..."
    echo "Capturing current device states..."
    
    # Store current state as scene
    curl -X POST "http://localhost:${ZIGBEE2MQTT_PORT}/api/bridge/request/scene/store" \
        -H "Content-Type: application/json" \
        -d "{\"id\": \"${scene_name}\", \"group\": \"${group}\"}" || {
        log::error "Failed to create scene"
        return 1
    }
    
    log::success "Scene '$scene_name' created"
}

# Recall/activate a scene
zigbee2mqtt::scene::recall() {
    local scene_name="${1:-}"
    
    if [[ -z "$scene_name" ]]; then
        log::error "Scene name required"
        return 1
    fi
    
    log::info "Activating scene: $scene_name"
    
    curl -X POST "http://localhost:${ZIGBEE2MQTT_PORT}/api/bridge/request/scene/recall" \
        -H "Content-Type: application/json" \
        -d "{\"id\": \"${scene_name}\"}" || {
        log::error "Failed to activate scene"
        return 1
    }
    
    log::success "Scene '$scene_name' activated"
}

################################################################################
# OTA Firmware Updates
################################################################################

# Check for firmware updates
zigbee2mqtt::ota::check() {
    local device="${1:-all}"
    
    log::info "Checking for firmware updates..."
    
    if [[ "$device" == "all" ]]; then
        # Check all devices
        curl -X POST "http://localhost:${ZIGBEE2MQTT_PORT}/api/bridge/request/device/ota_update/check" \
            -H "Content-Type: application/json" \
            -d '{"id": "all"}' 2>/dev/null | jq '.' || {
            log::error "Failed to check for updates"
            return 1
        }
    else
        # Check specific device
        curl -X POST "http://localhost:${ZIGBEE2MQTT_PORT}/api/bridge/request/device/ota_update/check" \
            -H "Content-Type: application/json" \
            -d "{\"id\": \"${device}\"}" 2>/dev/null | jq '.' || {
            log::error "Failed to check for updates"
            return 1
        }
    fi
}

# Update device firmware
zigbee2mqtt::ota::update() {
    local device="${1:-}"
    
    if [[ -z "$device" ]]; then
        log::error "Device name required"
        echo "Usage: ota update <device_name>"
        return 1
    fi
    
    log::warning "Firmware update will take 5-30 minutes and device may be unresponsive"
    read -p "Continue with firmware update for $device? (y/N): " -n 1 -r
    echo
    
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log::info "Update cancelled"
        return 0
    fi
    
    log::info "Starting firmware update for $device..."
    echo "DO NOT power off the device during update!"
    
    curl -X POST "http://localhost:${ZIGBEE2MQTT_PORT}/api/bridge/request/device/ota_update/update" \
        -H "Content-Type: application/json" \
        -d "{\"id\": \"${device}\"}" || {
        log::error "Failed to start firmware update"
        return 1
    }
    
    log::info "Firmware update initiated. Check logs for progress:"
    echo "  vrooli resource zigbee2mqtt logs --follow"
}

################################################################################
# Touchlink Support
################################################################################

# Scan for Touchlink devices
zigbee2mqtt::touchlink::scan() {
    log::info "Scanning for Touchlink devices..."
    echo "Bring your Touchlink device within 10cm of the coordinator"
    echo "Scanning for 20 seconds..."
    
    # Start Touchlink scan via API
    local response=$(curl -X POST "http://localhost:${ZIGBEE2MQTT_PORT}/api/bridge/request/touchlink/scan" \
        -H "Content-Type: application/json" \
        -d '{}' 2>/dev/null)
    
    if [[ -n "$response" ]]; then
        echo "$response" | jq -r '.data.found[] | "Found: \(.ieee_address) - Channel: \(.channel)"' 2>/dev/null || {
            log::warning "No Touchlink devices found"
            return 1
        }
        log::success "Scan complete. Use 'touchlink identify' to identify specific devices"
    else
        log::error "Touchlink scan failed - ensure coordinator supports Touchlink"
        return 1
    fi
}

# Identify Touchlink device (blink/flash)
zigbee2mqtt::touchlink::identify() {
    local device="${1:-}"
    local duration="${2:-10}"
    
    if [[ -z "$device" ]]; then
        log::error "Device IEEE address required"
        echo "Usage: touchlink identify <ieee_address> [duration]"
        echo "Get IEEE address from 'touchlink scan' command"
        return 1
    fi
    
    log::info "Identifying device $device for ${duration} seconds..."
    
    curl -X POST "http://localhost:${ZIGBEE2MQTT_PORT}/api/bridge/request/touchlink/identify" \
        -H "Content-Type: application/json" \
        -d "{\"ieee_address\": \"${device}\", \"duration\": ${duration}}" || {
        log::error "Failed to identify device"
        return 1
    }
    
    log::success "Device should be blinking/flashing now"
}

# Reset Touchlink device to factory defaults
zigbee2mqtt::touchlink::reset() {
    local device="${1:-}"
    
    if [[ -z "$device" ]]; then
        log::error "Device IEEE address required"
        echo "Usage: touchlink reset <ieee_address>"
        return 1
    fi
    
    log::warning "This will factory reset device $device"
    read -p "Continue? (y/N): " -n 1 -r
    echo
    
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log::info "Reset cancelled"
        return 0
    fi
    
    log::info "Resetting device to factory defaults..."
    
    curl -X POST "http://localhost:${ZIGBEE2MQTT_PORT}/api/bridge/request/touchlink/factory_reset" \
        -H "Content-Type: application/json" \
        -d "{\"ieee_address\": \"${device}\"}" || {
        log::error "Failed to reset device"
        return 1
    }
    
    log::success "Device reset to factory defaults"
    echo "Device can now be paired to a new network"
}

################################################################################
# External Converters Support
################################################################################

# Add external converter for unsupported device
zigbee2mqtt::converter::add() {
    local converter_file="${1:-}"
    
    if [[ -z "$converter_file" ]] || [[ ! -f "$converter_file" ]]; then
        log::error "Valid converter file required"
        echo "Usage: converter add <converter_file.js>"
        echo ""
        echo "Example converter structure:"
        echo "  const definition = {"
        echo "    zigbeeModel: ['Device Model'],"
        echo "    model: 'Custom Device',"
        echo "    vendor: 'Custom Vendor',"
        echo "    description: 'Custom device description',"
        echo "    fromZigbee: [],"
        echo "    toZigbee: [],"
        echo "    exposes: []"
        echo "  };"
        echo "  module.exports = definition;"
        return 1
    fi
    
    # Create external converters directory
    local converters_dir="${ZIGBEE2MQTT_DATA_DIR}/external_converters"
    mkdir -p "$converters_dir"
    
    # Copy converter file
    local converter_name="$(basename "$converter_file")"
    cp "$converter_file" "$converters_dir/$converter_name"
    
    # Update configuration to include external converters
    local config_file="${ZIGBEE2MQTT_DATA_DIR}/configuration.yaml"
    
    # Check if external_converters is already in config
    if ! grep -q "external_converters:" "$config_file"; then
        echo "" >> "$config_file"
        echo "# External device converters" >> "$config_file"
        echo "external_converters:" >> "$config_file"
        echo "  - external_converters/${converter_name}" >> "$config_file"
    else
        # Add to existing external_converters list
        sed -i "/external_converters:/a\  - external_converters/${converter_name}" "$config_file"
    fi
    
    log::success "External converter added: $converter_name"
    echo "Restart Zigbee2MQTT to load the converter:"
    echo "  vrooli resource zigbee2mqtt manage restart"
}

# List external converters
zigbee2mqtt::converter::list() {
    local converters_dir="${ZIGBEE2MQTT_DATA_DIR}/external_converters"
    
    if [[ ! -d "$converters_dir" ]] || [[ -z "$(ls -A "$converters_dir" 2>/dev/null)" ]]; then
        log::info "No external converters installed"
        return 0
    fi
    
    log::info "External converters:"
    ls -la "$converters_dir" | grep -E "\.js$" | awk '{print "  - " $NF}'
}

# Remove external converter
zigbee2mqtt::converter::remove() {
    local converter_name="${1:-}"
    
    if [[ -z "$converter_name" ]]; then
        log::error "Converter name required"
        echo "Usage: converter remove <converter_name.js>"
        return 1
    fi
    
    local converters_dir="${ZIGBEE2MQTT_DATA_DIR}/external_converters"
    local converter_file="$converters_dir/$converter_name"
    
    if [[ ! -f "$converter_file" ]]; then
        log::error "Converter not found: $converter_name"
        return 1
    fi
    
    # Remove file
    rm "$converter_file"
    
    # Remove from configuration
    local config_file="${ZIGBEE2MQTT_DATA_DIR}/configuration.yaml"
    sed -i "/external_converters\/${converter_name}/d" "$config_file"
    
    log::success "External converter removed: $converter_name"
    echo "Restart Zigbee2MQTT to apply changes"
}

# Generate converter template for new device
zigbee2mqtt::converter::generate() {
    local model="${1:-}"
    local vendor="${2:-}"
    local output="${3:-custom_device.js}"
    
    if [[ -z "$model" ]] || [[ -z "$vendor" ]]; then
        log::error "Model and vendor required"
        echo "Usage: converter generate <model> <vendor> [output_file]"
        return 1
    fi
    
    log::info "Generating converter template for $vendor $model..."
    
    cat > "$output" << EOF
// External converter for ${vendor} ${model}
// Generated by Zigbee2MQTT resource

const fz = require('zigbee-herdsman-converters/converters/fromZigbee');
const tz = require('zigbee-herdsman-converters/converters/toZigbee');
const exposes = require('zigbee-herdsman-converters/lib/exposes');
const reporting = require('zigbee-herdsman-converters/lib/reporting');
const extend = require('zigbee-herdsman-converters/lib/extend');
const e = exposes.presets;
const ea = exposes.access;

const definition = {
    zigbeeModel: ['${model}'],
    model: '${model}',
    vendor: '${vendor}',
    description: 'Custom ${vendor} ${model} device',
    // Extend from generic device types if applicable
    // extend: extend.light_onoff_brightness_colortemp(),
    
    // Define what the device sends to Zigbee2MQTT
    fromZigbee: [
        fz.on_off,           // For on/off devices
        // fz.brightness,    // For dimmable lights
        // fz.color_colortemp, // For color lights
        // fz.temperature,   // For temperature sensors
        // fz.humidity,      // For humidity sensors
        // fz.battery,       // For battery powered devices
    ],
    
    // Define what Zigbee2MQTT can send to device
    toZigbee: [
        tz.on_off,           // For controllable on/off
        // tz.light_brightness, // For brightness control
        // tz.light_colortemp, // For color temperature
        // tz.light_color,   // For RGB color
    ],
    
    // Define what capabilities to expose in UI/MQTT
    exposes: [
        e.switch(),          // On/off switch
        // e.light_brightness(), // Dimmable light
        // e.light_brightness_colortemp(), // Light with brightness and color temp
        // e.temperature(),  // Temperature sensor
        // e.humidity(),     // Humidity sensor
        // e.battery(),      // Battery level
    ],
    
    // Optional: Configure device-specific settings
    configure: async (device, coordinatorEndpoint, logger) => {
        const endpoint = device.getEndpoint(1);
        // Configure reporting intervals
        await reporting.bind(endpoint, coordinatorEndpoint, ['genOnOff']);
        await reporting.onOff(endpoint);
    },
};

module.exports = definition;
EOF
    
    log::success "Converter template generated: $output"
    echo ""
    echo "Next steps:"
    echo "1. Edit $output to match your device capabilities"
    echo "2. Test with: vrooli resource zigbee2mqtt converter add $output"
    echo "3. Restart Zigbee2MQTT and pair your device"
}

################################################################################
# Home Assistant Integration
################################################################################

# Enable/disable Home Assistant discovery
zigbee2mqtt::homeassistant::discovery() {
    local action="${1:-status}"
    
    case "$action" in
        enable)
            log::info "Enabling Home Assistant discovery..."
            # Update configuration to enable discovery
            sed -i 's/homeassistant: false/homeassistant: true/' "${ZIGBEE2MQTT_DATA_DIR}/configuration.yaml"
            zigbee2mqtt::restart
            log::success "Home Assistant discovery enabled"
            echo "Devices will now be auto-discovered by Home Assistant via MQTT"
            ;;
        disable)
            log::info "Disabling Home Assistant discovery..."
            # Update configuration to disable discovery
            sed -i 's/homeassistant: true/homeassistant: false/' "${ZIGBEE2MQTT_DATA_DIR}/configuration.yaml"
            zigbee2mqtt::restart
            log::success "Home Assistant discovery disabled"
            ;;
        status)
            if grep -q "homeassistant: true" "${ZIGBEE2MQTT_DATA_DIR}/configuration.yaml"; then
                echo "Home Assistant discovery: ENABLED"
                echo "Discovery topics: homeassistant/+/+/config"
            else
                echo "Home Assistant discovery: DISABLED"
            fi
            ;;
        *)
            log::error "Usage: homeassistant discovery <enable|disable|status>"
            return 1
            ;;
    esac
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
    local backup_dir="${ZIGBEE2MQTT_DATA_DIR}/backups"
    mkdir -p "$backup_dir"
    
    local backup_file="${1:-${backup_dir}/coordinator_backup_$(date +%Y%m%d_%H%M%S).json}"
    
    log::info "Creating coordinator backup..."
    
    # Request backup via MQTT bridge
    local response=$(curl -X POST "http://localhost:${ZIGBEE2MQTT_PORT}/api/bridge/request/backup" \
        -H "Content-Type: application/json" \
        -d '{}' 2>/dev/null)
    
    if [[ -z "$response" ]]; then
        log::error "Failed to create backup - no response from Zigbee2MQTT"
        return 1
    fi
    
    # Save backup to file
    echo "$response" > "$backup_file"
    
    # Also backup the full configuration
    cp "${ZIGBEE2MQTT_DATA_DIR}/configuration.yaml" "${backup_file%.json}_config.yaml"
    
    log::success "Backup saved to: $backup_file"
    echo "Configuration backed up to: ${backup_file%.json}_config.yaml"
    
    # List recent backups
    echo ""
    echo "Recent backups:"
    ls -lt "$backup_dir" | head -6
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
    
    log::info "Restoring coordinator from backup..."
    
    # Stop Zigbee2MQTT first
    zigbee2mqtt::stop
    
    # Backup current state before restore
    local current_backup="${ZIGBEE2MQTT_DATA_DIR}/backups/pre_restore_$(date +%Y%m%d_%H%M%S).json"
    cp "${ZIGBEE2MQTT_DATA_DIR}/coordinator_backup.json" "$current_backup" 2>/dev/null || true
    
    # Copy backup file to expected location
    cp "$backup_file" "${ZIGBEE2MQTT_DATA_DIR}/coordinator_backup.json"
    
    # Start Zigbee2MQTT with restore
    zigbee2mqtt::start --wait
    
    log::success "Coordinator restored from backup"
    echo "Previous state backed up to: $current_backup"
    echo "Network should be restored with all device pairings intact"
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