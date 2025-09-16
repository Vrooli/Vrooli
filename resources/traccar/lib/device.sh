#!/bin/bash
# Traccar Device Management Functions

set -euo pipefail

# Create a new device
device_create() {
    local name=""
    local type="vehicle"
    local unique_id=""
    local phone=""
    local model=""
    local contact=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name)
                name="$2"
                shift 2
                ;;
            --type)
                type="$2"
                shift 2
                ;;
            --unique-id)
                unique_id="$2"
                shift 2
                ;;
            --phone)
                phone="$2"
                shift 2
                ;;
            --model)
                model="$2"
                shift 2
                ;;
            --contact)
                contact="$2"
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
    
    # Generate unique ID if not provided
    if [[ -z "$unique_id" ]]; then
        unique_id=$(date +%s%N | sha256sum | cut -c1-12)
    fi
    
    # Create device JSON
    local device_json=$(cat << EOF
{
  "name": "${name}",
  "uniqueId": "${unique_id}",
  "category": "${type}",
  "model": "${model}",
  "phone": "${phone}",
  "contact": "${contact}",
  "disabled": false
}
EOF
)
    
    # Create device via API
    local response=$(curl -sf -X POST \
        -H "Content-Type: application/json" \
        -u "${TRACCAR_ADMIN_EMAIL}:${TRACCAR_ADMIN_PASSWORD}" \
        -d "${device_json}" \
        "http://${TRACCAR_HOST}:${TRACCAR_PORT}/api/devices")
    
    if [[ $? -eq 0 ]]; then
        local device_id=$(echo "$response" | jq -r '.id')
        echo "Device created successfully:"
        echo "  Name:      $name"
        echo "  ID:        $device_id"
        echo "  Unique ID: $unique_id"
        echo "  Type:      $type"
    else
        echo "Error: Failed to create device" >&2
        exit 1
    fi
}

# List all devices
device_list() {
    local format="text"
    local filter=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --format)
                format="$2"
                shift 2
                ;;
            --filter)
                filter="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Get all devices
    local response=$(curl -sf \
        -u "${TRACCAR_ADMIN_EMAIL}:${TRACCAR_ADMIN_PASSWORD}" \
        "http://${TRACCAR_HOST}:${TRACCAR_PORT}/api/devices")
    
    if [[ "$format" == "json" ]]; then
        if [[ -n "$filter" ]]; then
            echo "$response" | jq --arg filter "$filter" '[.[] | select(.name | contains($filter))]'
        else
            echo "$response"
        fi
    else
        echo "Traccar Devices:"
        if [[ -n "$filter" ]]; then
            echo "$response" | jq -r --arg filter "$filter" '.[] | select(.name | contains($filter)) | "  [\(.id)] \(.name) - \(.category // "unknown") (\(.status // "offline"))"'
        else
            echo "$response" | jq -r '.[] | "  [\(.id)] \(.name) - \(.category // "unknown") (\(.status // "offline"))"'
        fi
    fi
}

# Update device information
device_update() {
    local device_id=""
    local name=""
    local phone=""
    local model=""
    local contact=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --id)
                device_id="$2"
                shift 2
                ;;
            --name)
                name="$2"
                shift 2
                ;;
            --phone)
                phone="$2"
                shift 2
                ;;
            --model)
                model="$2"
                shift 2
                ;;
            --contact)
                contact="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$device_id" ]]; then
        echo "Error: --id is required" >&2
        exit 1
    fi
    
    # Get current device data
    local current=$(curl -sf \
        -u "${TRACCAR_ADMIN_EMAIL}:${TRACCAR_ADMIN_PASSWORD}" \
        "http://${TRACCAR_HOST}:${TRACCAR_PORT}/api/devices/${device_id}")
    
    if [[ -z "$current" ]]; then
        echo "Error: Device not found: $device_id" >&2
        exit 2
    fi
    
    # Update fields
    local updated=$(echo "$current" | jq \
        --arg name "${name:-$(echo "$current" | jq -r '.name')}" \
        --arg phone "${phone:-$(echo "$current" | jq -r '.phone // empty')}" \
        --arg model "${model:-$(echo "$current" | jq -r '.model // empty')}" \
        --arg contact "${contact:-$(echo "$current" | jq -r '.contact // empty')}" \
        '.name = $name | .phone = $phone | .model = $model | .contact = $contact')
    
    # Update device via API
    curl -sf -X PUT \
        -H "Content-Type: application/json" \
        -u "${TRACCAR_ADMIN_EMAIL}:${TRACCAR_ADMIN_PASSWORD}" \
        -d "${updated}" \
        "http://${TRACCAR_HOST}:${TRACCAR_PORT}/api/devices/${device_id}"
    
    echo "Device updated successfully"
}

# Delete a device
device_delete() {
    local device_id=""
    local force=false
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --id)
                device_id="$2"
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
    
    if [[ -z "$device_id" ]]; then
        echo "Error: --id is required" >&2
        exit 1
    fi
    
    # Get device info
    local device=$(curl -sf \
        -u "${TRACCAR_ADMIN_EMAIL}:${TRACCAR_ADMIN_PASSWORD}" \
        "http://${TRACCAR_HOST}:${TRACCAR_PORT}/api/devices/${device_id}")
    
    if [[ -z "$device" ]]; then
        echo "Error: Device not found: $device_id" >&2
        exit 2
    fi
    
    local device_name=$(echo "$device" | jq -r '.name')
    
    if [[ "$force" != true ]]; then
        echo -n "Delete device '$device_name' (ID: $device_id)? [y/N] "
        read -r confirm
        if [[ "$confirm" != "y" ]]; then
            echo "Cancelled"
            exit 0
        fi
    fi
    
    curl -sf -X DELETE \
        -u "${TRACCAR_ADMIN_EMAIL}:${TRACCAR_ADMIN_PASSWORD}" \
        "http://${TRACCAR_HOST}:${TRACCAR_PORT}/api/devices/${device_id}"
    
    echo "Device deleted: $device_name"
}