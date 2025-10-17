#!/bin/bash
# Traccar GPS Tracking Functions

set -euo pipefail

# Push GPS position
track_push() {
    local device=""
    local lat=""
    local lon=""
    local speed="0"
    local bearing="0"
    local altitude="0"
    local accuracy="10"
    local timestamp=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --device)
                device="$2"
                shift 2
                ;;
            --lat)
                lat="$2"
                shift 2
                ;;
            --lon)
                lon="$2"
                shift 2
                ;;
            --speed)
                speed="$2"
                shift 2
                ;;
            --bearing)
                bearing="$2"
                shift 2
                ;;
            --altitude)
                altitude="$2"
                shift 2
                ;;
            --accuracy)
                accuracy="$2"
                shift 2
                ;;
            --timestamp)
                timestamp="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$device" ]] || [[ -z "$lat" ]] || [[ -z "$lon" ]]; then
        echo "Error: --device, --lat, and --lon are required" >&2
        exit 1
    fi
    
    # Get device unique ID
    local device_data=$(curl -sf \
        -u "${TRACCAR_ADMIN_EMAIL}:${TRACCAR_ADMIN_PASSWORD}" \
        "http://${TRACCAR_HOST}:${TRACCAR_PORT}/api/devices" | \
        jq --arg name "$device" '.[] | select(.name == $name or (.id | tostring) == $name)')
    
    if [[ -z "$device_data" ]]; then
        echo "Error: Device not found: $device" >&2
        exit 2
    fi
    
    local unique_id=$(echo "$device_data" | jq -r '.uniqueId')
    local device_id=$(echo "$device_data" | jq -r '.id')
    
    # Use current timestamp if not provided
    if [[ -z "$timestamp" ]]; then
        timestamp=$(date -u +"%Y-%m-%dT%H:%M:%S.000Z")
    fi
    
    # Send position via OsmAnd protocol (simple HTTP GET)
    local url="http://${TRACCAR_HOST}:${TRACCAR_PORT}/?id=${unique_id}&lat=${lat}&lon=${lon}"
    url="${url}&speed=${speed}&bearing=${bearing}&altitude=${altitude}&accuracy=${accuracy}"
    url="${url}&timestamp=$(date +%s)"
    
    if curl -sf "$url" &>/dev/null; then
        echo "Position sent successfully:"
        echo "  Device:   $device (ID: $device_id)"
        echo "  Location: $lat, $lon"
        echo "  Speed:    ${speed} km/h"
        echo "  Time:     $timestamp"
    else
        # Try alternative: Create position via API
        local position_json=$(cat << EOF
{
  "deviceId": ${device_id},
  "protocol": "api",
  "deviceTime": "${timestamp}",
  "fixTime": "${timestamp}",
  "serverTime": "${timestamp}",
  "valid": true,
  "latitude": ${lat},
  "longitude": ${lon},
  "altitude": ${altitude},
  "speed": ${speed},
  "course": ${bearing},
  "accuracy": ${accuracy}
}
EOF
)
        
        curl -sf -X POST \
            -H "Content-Type: application/json" \
            -u "${TRACCAR_ADMIN_EMAIL}:${TRACCAR_ADMIN_PASSWORD}" \
            -d "${position_json}" \
            "http://${TRACCAR_HOST}:${TRACCAR_PORT}/api/positions"
        
        echo "Position sent via API"
    fi
}

# Get position history
track_history() {
    local device=""
    local days="7"
    local format="text"
    local from=""
    local to=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --device)
                device="$2"
                shift 2
                ;;
            --days)
                days="$2"
                shift 2
                ;;
            --from)
                from="$2"
                shift 2
                ;;
            --to)
                to="$2"
                shift 2
                ;;
            --format)
                format="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$device" ]]; then
        echo "Error: --device is required" >&2
        exit 1
    fi
    
    # Get device ID
    local device_id=$(curl -sf \
        -u "${TRACCAR_ADMIN_EMAIL}:${TRACCAR_ADMIN_PASSWORD}" \
        "http://${TRACCAR_HOST}:${TRACCAR_PORT}/api/devices" | \
        jq --arg name "$device" -r '.[] | select(.name == $name or (.id | tostring) == $name) | .id')
    
    if [[ -z "$device_id" ]]; then
        echo "Error: Device not found: $device" >&2
        exit 2
    fi
    
    # Calculate date range
    if [[ -z "$from" ]]; then
        from=$(date -u -d "${days} days ago" +"%Y-%m-%dT00:00:00.000Z")
    fi
    if [[ -z "$to" ]]; then
        to=$(date -u +"%Y-%m-%dT%H:%M:%S.000Z")
    fi
    
    # Get positions
    local positions=$(curl -sf -G \
        -u "${TRACCAR_ADMIN_EMAIL}:${TRACCAR_ADMIN_PASSWORD}" \
        --data-urlencode "deviceId=${device_id}" \
        --data-urlencode "from=${from}" \
        --data-urlencode "to=${to}" \
        "http://${TRACCAR_HOST}:${TRACCAR_PORT}/api/positions")
    
    if [[ "$format" == "json" ]]; then
        echo "$positions"
    else
        echo "Position History for $device:"
        echo "  From: $from"
        echo "  To:   $to"
        echo ""
        echo "$positions" | jq -r '.[] | "  [\(.deviceTime)] Lat: \(.latitude), Lon: \(.longitude), Speed: \(.speed) km/h"'
    fi
}

# Live tracking (WebSocket)
track_live() {
    local device=""
    local duration="60"
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --device)
                device="$2"
                shift 2
                ;;
            --duration)
                duration="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$device" ]]; then
        echo "Error: --device is required" >&2
        exit 1
    fi
    
    echo "Starting live tracking for $device..."
    echo "Duration: ${duration} seconds"
    echo ""
    echo "Note: Live tracking requires WebSocket connection."
    echo "For demo purposes, polling latest position every 5 seconds..."
    echo ""
    
    # Get device ID
    local device_id=$(curl -sf \
        -u "${TRACCAR_ADMIN_EMAIL}:${TRACCAR_ADMIN_PASSWORD}" \
        "http://${TRACCAR_HOST}:${TRACCAR_PORT}/api/devices" | \
        jq --arg name "$device" -r '.[] | select(.name == $name or (.id | tostring) == $name) | .id')
    
    if [[ -z "$device_id" ]]; then
        echo "Error: Device not found: $device" >&2
        exit 2
    fi
    
    local elapsed=0
    while [[ $elapsed -lt $duration ]]; do
        # Get latest position
        local position=$(curl -sf \
            -u "${TRACCAR_ADMIN_EMAIL}:${TRACCAR_ADMIN_PASSWORD}" \
            "http://${TRACCAR_HOST}:${TRACCAR_PORT}/api/positions?deviceId=${device_id}" | \
            jq -r '.[0] | if . then "[\(.deviceTime)] Lat: \(.latitude), Lon: \(.longitude), Speed: \(.speed) km/h" else "No position data" end')
        
        echo "[$(date +%H:%M:%S)] $position"
        
        sleep 5
        elapsed=$((elapsed + 5))
    done
    
    echo ""
    echo "Live tracking ended"
}