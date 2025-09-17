#!/usr/bin/env bash
# Traccar Adapter for Open MCT

set -euo pipefail

# Traccar Configuration
TRACCAR_URL="${TRACCAR_URL:-http://localhost:8082}"
TRACCAR_API_KEY="${TRACCAR_API_KEY:-}"
TRACCAR_POLL_INTERVAL="${TRACCAR_POLL_INTERVAL:-5}"  # seconds

# Connect to Traccar and forward positions to Open MCT
connect_traccar() {
    local url="${1:-$TRACCAR_URL}"
    local api_key="${2:-$TRACCAR_API_KEY}"
    
    if [[ -z "$url" ]]; then
        echo "Error: Traccar URL not specified"
        exit 1
    fi
    
    echo "Connecting to Traccar: $url"
    echo "Polling interval: ${TRACCAR_POLL_INTERVAL}s"
    
    while true; do
        # Get positions from Traccar
        if [[ -n "$api_key" ]]; then
            positions=$(curl -s -H "Authorization: Bearer $api_key" "${url}/api/positions")
        else
            positions=$(curl -s "${url}/api/positions")
        fi
        
        # Process each position
        echo "$positions" | jq -c '.[]' | while read -r position; do
            device_id=$(echo "$position" | jq -r '.deviceId')
            latitude=$(echo "$position" | jq -r '.latitude')
            longitude=$(echo "$position" | jq -r '.longitude')
            speed=$(echo "$position" | jq -r '.speed // 0')
            altitude=$(echo "$position" | jq -r '.altitude // 0')
            timestamp=$(echo "$position" | jq -r '.fixTime' | xargs -I{} date -d {} +%s000 2>/dev/null || date +%s000)
            
            # Create telemetry data
            data=$(cat <<EOF
{
    "timestamp": $timestamp,
    "value": $speed,
    "data": {
        "latitude": $latitude,
        "longitude": $longitude,
        "altitude": $altitude,
        "speed": $speed,
        "device_id": $device_id
    }
}
EOF
            )
            
            # Send to Open MCT
            curl -X POST "http://localhost:${OPENMCT_PORT}/api/telemetry/traccar_device_${device_id}/data" \
                -H "Content-Type: application/json" \
                -d "$data" \
                2>/dev/null || echo "Failed to forward position for device $device_id"
        done
        
        sleep "$TRACCAR_POLL_INTERVAL"
    done
}

# Configure Traccar source
configure_traccar() {
    local url="${1:-}"
    local api_key="${2:-}"
    local interval="${3:-5}"
    
    if [[ -z "$url" ]]; then
        echo "Usage: configure_traccar <url> [api_key] [poll_interval]"
        exit 1
    fi
    
    # Save configuration
    cat > "${OPENMCT_CONFIG_DIR}/traccar_source.json" << EOF
{
    "type": "traccar",
    "url": "$url",
    "api_key": "$api_key",
    "poll_interval": $interval,
    "enabled": true
}
EOF
    
    echo "Traccar source configured:"
    echo "  URL: $url"
    echo "  Poll interval: ${interval}s"
    echo "  Config saved to: ${OPENMCT_CONFIG_DIR}/traccar_source.json"
}

# Test Traccar connection
test_traccar() {
    local url="${1:-$TRACCAR_URL}"
    local api_key="${2:-$TRACCAR_API_KEY}"
    
    if [[ -z "$url" ]]; then
        echo "Error: Traccar URL not specified"
        exit 1
    fi
    
    echo "Testing Traccar connection to: $url"
    
    # Try to get server info
    if [[ -n "$api_key" ]]; then
        response=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $api_key" "${url}/api/server")
    else
        response=$(curl -s -o /dev/null -w "%{http_code}" "${url}/api/server")
    fi
    
    if [[ "$response" == "200" ]]; then
        echo "✓ Traccar connection successful"
        
        # Get device count
        if [[ -n "$api_key" ]]; then
            devices=$(curl -s -H "Authorization: Bearer $api_key" "${url}/api/devices" | jq '. | length')
        else
            devices=$(curl -s "${url}/api/devices" | jq '. | length')
        fi
        
        echo "  Found $devices devices"
        return 0
    else
        echo "✗ Traccar connection failed (HTTP $response)"
        return 1
    fi
}

# List Traccar devices
list_devices() {
    local url="${1:-$TRACCAR_URL}"
    local api_key="${2:-$TRACCAR_API_KEY}"
    
    if [[ -z "$url" ]]; then
        echo "Error: Traccar URL not specified"
        exit 1
    fi
    
    echo "Traccar Devices:"
    echo "================"
    
    if [[ -n "$api_key" ]]; then
        devices=$(curl -s -H "Authorization: Bearer $api_key" "${url}/api/devices")
    else
        devices=$(curl -s "${url}/api/devices")
    fi
    
    echo "$devices" | jq -r '.[] | "\(.id): \(.name) (\(.status // "unknown"))"'
}

# Main command handler
case "${1:-}" in
    connect)
        shift
        connect_traccar "$@"
        ;;
    configure)
        shift
        configure_traccar "$@"
        ;;
    test)
        shift
        test_traccar "$@"
        ;;
    devices)
        shift
        list_devices "$@"
        ;;
    *)
        echo "Traccar Adapter for Open MCT"
        echo "============================="
        echo ""
        echo "Commands:"
        echo "  connect [url] [api_key]     - Connect to Traccar and forward positions"
        echo "  configure <url> [api_key]   - Configure Traccar source"
        echo "  test [url] [api_key]        - Test Traccar connection"
        echo "  devices [url] [api_key]     - List Traccar devices"
        echo ""
        echo "Environment Variables:"
        echo "  TRACCAR_URL          - Traccar server URL"
        echo "  TRACCAR_API_KEY      - Traccar API key (optional)"
        echo "  TRACCAR_POLL_INTERVAL - Polling interval in seconds (default: 5)"
        ;;
esac