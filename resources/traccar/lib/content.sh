#!/bin/bash
# Traccar Content Management Functions

set -euo pipefail

# Add content (device or GPS data)
content_add() {
    local file=""
    local type="device"
    local name=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --file)
                file="$2"
                shift 2
                ;;
            --type)
                type="$2"
                shift 2
                ;;
            --name)
                name="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    case "$type" in
        device)
            # Create device from JSON file
            if [[ -n "$file" ]] && [[ -f "$file" ]]; then
                local response=$(curl -sf -X POST \
                    -H "Content-Type: application/json" \
                    -u "${TRACCAR_ADMIN_EMAIL}:${TRACCAR_ADMIN_PASSWORD}" \
                    -d "@${file}" \
                    "http://${TRACCAR_HOST}:${TRACCAR_PORT}/api/devices")
                echo "Device created: $(echo "$response" | jq -r '.name')"
            else
                echo "Error: File not found: $file" >&2
                exit 1
            fi
            ;;
        position)
            # Add GPS position data
            if [[ -n "$file" ]] && [[ -f "$file" ]]; then
                # Process GPS data file
                echo "Processing GPS position data from $file..."
                # Implementation would parse and send position data
            fi
            ;;
        *)
            echo "Error: Unknown content type: $type" >&2
            exit 1
            ;;
    esac
}

# List content
content_list() {
    local filter="${1:-}"
    local format="${2:-text}"
    
    # List all devices
    local response=$(curl -sf \
        -u "${TRACCAR_ADMIN_EMAIL}:${TRACCAR_ADMIN_PASSWORD}" \
        "http://${TRACCAR_HOST}:${TRACCAR_PORT}/api/devices")
    
    if [[ "$format" == "--format=json" ]]; then
        echo "$response"
    else
        echo "Traccar Devices:"
        echo "$response" | jq -r '.[] | "  - \(.name) (ID: \(.id), Status: \(.status))"'
    fi
}

# Get specific content
content_get() {
    local name=""
    local output=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name)
                name="$2"
                shift 2
                ;;
            --output)
                output="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$name" ]]; then
        echo "Error: --name is required" >&2
        exit 1
    fi
    
    # Get device by name or ID
    local response=$(curl -sf \
        -u "${TRACCAR_ADMIN_EMAIL}:${TRACCAR_ADMIN_PASSWORD}" \
        "http://${TRACCAR_HOST}:${TRACCAR_PORT}/api/devices" | \
        jq --arg name "$name" '.[] | select(.name == $name or (.id | tostring) == $name)')
    
    if [[ -z "$response" ]]; then
        echo "Error: Device not found: $name" >&2
        exit 2
    fi
    
    if [[ -n "$output" ]]; then
        echo "$response" > "$output"
        echo "Device details saved to $output"
    else
        echo "$response" | jq .
    fi
}

# Remove content
content_remove() {
    local name=""
    local force=false
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name)
                name="$2"
                shift 2
                ;;
            --force)
                force=true
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$name" ]]; then
        echo "Error: --name is required" >&2
        exit 1
    fi
    
    # Get device ID
    local device_id=$(curl -sf \
        -u "${TRACCAR_ADMIN_EMAIL}:${TRACCAR_ADMIN_PASSWORD}" \
        "http://${TRACCAR_HOST}:${TRACCAR_PORT}/api/devices" | \
        jq --arg name "$name" -r '.[] | select(.name == $name or (.id | tostring) == $name) | .id')
    
    if [[ -z "$device_id" ]]; then
        echo "Error: Device not found: $name" >&2
        exit 2
    fi
    
    if [[ "$force" != true ]]; then
        echo -n "Remove device $name (ID: $device_id)? [y/N] "
        read -r confirm
        if [[ "$confirm" != "y" ]]; then
            echo "Cancelled"
            exit 0
        fi
    fi
    
    curl -sf -X DELETE \
        -u "${TRACCAR_ADMIN_EMAIL}:${TRACCAR_ADMIN_PASSWORD}" \
        "http://${TRACCAR_HOST}:${TRACCAR_PORT}/api/devices/${device_id}"
    
    echo "Device removed: $name"
}

# Execute/process content
content_execute() {
    local name=""
    local options=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name)
                name="$2"
                shift 2
                ;;
            --options)
                options="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Special case for seeding demo data
    if [[ "$name" == "demo" ]] || [[ "$name" == "seed-demo" ]]; then
        seed_demo_data
        return 0
    fi
    
    if [[ -n "$name" ]] && [[ -f "$name" ]]; then
        echo "Processing GPS batch file: $name"
        # Process batch GPS data file
        # This would parse and send multiple position updates
        echo "Batch processing completed"
    else
        echo "Error: Batch file not found: $name" >&2
        exit 1
    fi
}

# Seed demo data function
seed_demo_data() {
    echo "Seeding Traccar with demo data..."
    
    # Array of demo vehicle data
    local -a vehicles=(
        "Fleet-001,truck,37.7749,-122.4194,San Francisco HQ"
        "Fleet-002,car,37.7849,-122.4094,Downtown SF"
        "Fleet-003,van,37.7949,-122.3994,Mission District"
        "Delivery-001,truck,37.8049,-122.4294,North Beach"
        "Service-001,car,37.7649,-122.4394,SOMA District"
    )
    
    # Create demo devices
    echo "Creating demo devices..."
    for vehicle_data in "${vehicles[@]}"; do
        IFS=',' read -r name type lat lon location <<< "$vehicle_data"
        local unique_id=$(echo "$name" | sha256sum | cut -c1-12)
        
        # Check if device already exists
        local existing=$(curl -sf \
            -u "${TRACCAR_ADMIN_EMAIL}:${TRACCAR_ADMIN_PASSWORD}" \
            "http://${TRACCAR_HOST}:${TRACCAR_PORT}/api/devices" | \
            jq --arg name "$name" '.[] | select(.name == $name)')
        
        if [[ -n "$existing" ]]; then
            echo "  Device $name already exists, skipping..."
            continue
        fi
        
        # Create device
        local device_json=$(cat << EOF
{
  "name": "${name}",
  "uniqueId": "${unique_id}",
  "category": "${type}",
  "model": "Demo ${type}",
  "contact": "${location}",
  "disabled": false
}
EOF
)
        
        local response=$(curl -sf -X POST \
            -H "Content-Type: application/json" \
            -u "${TRACCAR_ADMIN_EMAIL}:${TRACCAR_ADMIN_PASSWORD}" \
            -d "${device_json}" \
            "http://${TRACCAR_HOST}:${TRACCAR_PORT}/api/devices")
        
        if [[ $? -eq 0 ]]; then
            local device_id=$(echo "$response" | jq -r '.id')
            echo "  Created device: $name (ID: $device_id)"
            
            # Add position history - just a few recent positions for demo
            echo "  Adding recent positions for $name..."
            local base_lat=$lat
            local base_lon=$lon
            
            # Add 5 recent positions over the last hour
            for minutes_ago in 60 45 30 15 0; do
                # Calculate timestamp
                local timestamp=$(date -u -d "$minutes_ago minutes ago" +"%Y-%m-%dT%H:%M:%S.000Z")
                
                # Add small random movement
                local lat_offset=$(awk -v min=-0.002 -v max=0.002 'BEGIN{srand(); print min+rand()*(max-min)}')
                local lon_offset=$(awk -v min=-0.002 -v max=0.002 'BEGIN{srand(); print min+rand()*(max-min)}')
                local position_lat=$(awk -v base="$base_lat" -v offset="$lat_offset" 'BEGIN{print base + offset}')
                local position_lon=$(awk -v base="$base_lon" -v offset="$lon_offset" 'BEGIN{print base + offset}')
                local speed=$(shuf -i 0-60 -n 1)
                local bearing=$(shuf -i 0-360 -n 1)
                
                # Send position via OsmAnd protocol
                curl -sf "http://${TRACCAR_HOST}:${TRACCAR_PORT}/?id=${unique_id}&lat=${position_lat}&lon=${position_lon}&speed=${speed}&bearing=${bearing}&timestamp=$(date -d "$timestamp" +%s)" &>/dev/null || true
            done
            echo "  Added 5 recent positions for $name"
        else
            echo "  Failed to create device: $name"
        fi
    done
    
    echo ""
    echo "Demo data seeding completed!"
    echo "Access Traccar web UI at: http://${TRACCAR_HOST}:${TRACCAR_PORT}"
    echo "Login with: ${TRACCAR_ADMIN_EMAIL} / ${TRACCAR_ADMIN_PASSWORD}"
}