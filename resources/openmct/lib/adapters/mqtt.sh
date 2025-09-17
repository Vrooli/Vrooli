#!/usr/bin/env bash
# MQTT Adapter for Open MCT

set -euo pipefail

# MQTT Configuration
MQTT_BROKER="${MQTT_BROKER:-}"
MQTT_PORT="${MQTT_PORT:-1883}"
MQTT_TOPIC="${MQTT_TOPIC:-telemetry/#}"
MQTT_CLIENT_ID="${MQTT_CLIENT_ID:-openmct-mqtt-$(date +%s)}"

# Connect to MQTT broker and forward to Open MCT
connect_mqtt() {
    local broker="${1:-$MQTT_BROKER}"
    local topic="${2:-$MQTT_TOPIC}"
    
    if [[ -z "$broker" ]]; then
        echo "Error: MQTT broker not specified"
        exit 1
    fi
    
    echo "Connecting to MQTT broker: $broker:$MQTT_PORT"
    echo "Subscribing to topic: $topic"
    
    # Use mosquitto_sub if available
    if command -v mosquitto_sub &> /dev/null; then
        mosquitto_sub -h "$broker" -p "$MQTT_PORT" -t "$topic" -i "$MQTT_CLIENT_ID" | while read -r message; do
            # Parse JSON message and forward to Open MCT
            stream=$(echo "$message" | jq -r '.stream // "mqtt"')
            timestamp=$(echo "$message" | jq -r '.timestamp // now * 1000')
            value=$(echo "$message" | jq -r '.value // 0')
            data=$(echo "$message" | jq -r '.data // {}')
            
            # Send to Open MCT API
            curl -X POST "http://localhost:${OPENMCT_PORT}/api/telemetry/${stream}/data" \
                -H "Content-Type: application/json" \
                -d "{\"timestamp\": $timestamp, \"value\": $value, \"data\": $data}" \
                2>/dev/null || echo "Failed to forward message"
        done
    else
        echo "Error: mosquitto_sub not installed"
        echo "Install with: apt-get install mosquitto-clients"
        exit 1
    fi
}

# Configure MQTT source
configure_mqtt() {
    local broker="${1:-}"
    local topic="${2:-telemetry/#}"
    local port="${3:-1883}"
    
    if [[ -z "$broker" ]]; then
        echo "Usage: configure_mqtt <broker> [topic] [port]"
        exit 1
    fi
    
    # Save configuration
    cat > "${OPENMCT_CONFIG_DIR}/mqtt_source.json" << EOF
{
    "type": "mqtt",
    "broker": "$broker",
    "port": $port,
    "topic": "$topic",
    "client_id": "$MQTT_CLIENT_ID",
    "enabled": true
}
EOF
    
    echo "MQTT source configured:"
    echo "  Broker: $broker:$port"
    echo "  Topic: $topic"
    echo "  Config saved to: ${OPENMCT_CONFIG_DIR}/mqtt_source.json"
}

# Test MQTT connection
test_mqtt() {
    local broker="${1:-$MQTT_BROKER}"
    
    if [[ -z "$broker" ]]; then
        echo "Error: MQTT broker not specified"
        exit 1
    fi
    
    echo "Testing MQTT connection to: $broker:$MQTT_PORT"
    
    # Try to connect and receive one message
    if timeout 5 mosquitto_sub -h "$broker" -p "$MQTT_PORT" -t "$MQTT_TOPIC" -C 1 &> /dev/null; then
        echo "✓ MQTT connection successful"
        return 0
    else
        echo "✗ MQTT connection failed"
        return 1
    fi
}

# Main command handler
case "${1:-}" in
    connect)
        shift
        connect_mqtt "$@"
        ;;
    configure)
        shift
        configure_mqtt "$@"
        ;;
    test)
        shift
        test_mqtt "$@"
        ;;
    *)
        echo "MQTT Adapter for Open MCT"
        echo "========================="
        echo ""
        echo "Commands:"
        echo "  connect [broker] [topic]    - Connect to MQTT broker and forward telemetry"
        echo "  configure <broker> [topic]  - Configure MQTT source"
        echo "  test [broker]              - Test MQTT connection"
        echo ""
        echo "Environment Variables:"
        echo "  MQTT_BROKER  - MQTT broker hostname"
        echo "  MQTT_PORT    - MQTT broker port (default: 1883)"
        echo "  MQTT_TOPIC   - MQTT topic to subscribe (default: telemetry/#)"
        ;;
esac