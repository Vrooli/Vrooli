#!/usr/bin/env bash
################################################################################
# Node-RED IoT Integration Library
# 
# Functions for integrating IoT devices and protocols
################################################################################

set -euo pipefail

# IoT protocol configurations
MQTT_BROKER_HOST="${MQTT_BROKER_HOST:-localhost}"
MQTT_BROKER_PORT="${MQTT_BROKER_PORT:-1883}"
COAP_SERVER_PORT="${COAP_SERVER_PORT:-5683}"
MODBUS_TCP_PORT="${MODBUS_TCP_PORT:-502}"

# Install IoT-related nodes
node_red::install_iot_nodes() {
    log::info "Installing IoT integration nodes..."
    
    if ! docker ps --format "{{.Names}}" | grep -q "^${NODE_RED_CONTAINER_NAME:-node-red}$"; then
        log::error "Node-RED container must be running"
        return 1
    fi
    
    local iot_nodes=(
        "node-red-contrib-mqtt-broker"      # MQTT broker/client
        "node-red-contrib-coap"              # CoAP protocol
        "node-red-contrib-modbus"            # Modbus protocol
        "node-red-contrib-opcua"             # OPC-UA protocol
        "node-red-contrib-zigbee2mqtt"       # Zigbee devices
        "node-red-contrib-home-assistant"    # Home Assistant integration
        "node-red-contrib-influxdb"          # Time-series data storage
        "node-red-contrib-sensor-ds18b20"    # Temperature sensors
        "node-red-contrib-gpio"              # GPIO control
    )
    
    for node in "${iot_nodes[@]}"; do
        log::info "Installing $node..."
        docker exec "${NODE_RED_CONTAINER_NAME:-node-red}" \
            npm install --no-audit --no-fund "$node" 2>/dev/null || {
            log::warning "Failed to install $node (may already exist)"
        }
    done
    
    # Restart to load new nodes
    docker restart "${NODE_RED_CONTAINER_NAME:-node-red}" > /dev/null
    
    log::success "IoT nodes installed"
}

# Create MQTT broker flow template
node_red::create_mqtt_template() {
    log::info "Creating MQTT integration template..."
    
    cat > "${HOME}/.local/share/node-red/mqtt-template.json" <<EOF
[
    {
        "id": "mqtt-flow",
        "type": "tab",
        "label": "MQTT IoT Integration",
        "disabled": false
    },
    {
        "id": "mqtt-broker-config",
        "type": "mqtt-broker",
        "name": "Local MQTT Broker",
        "broker": "${MQTT_BROKER_HOST}",
        "port": "${MQTT_BROKER_PORT}",
        "clientid": "node-red-client",
        "autoConnect": true,
        "usetls": false,
        "protocolVersion": "4",
        "keepalive": "60",
        "cleansession": true
    },
    {
        "id": "mqtt-in-sensor",
        "type": "mqtt in",
        "z": "mqtt-flow",
        "name": "Sensor Data Input",
        "topic": "sensors/+/data",
        "qos": "1",
        "datatype": "json",
        "broker": "mqtt-broker-config",
        "x": 100,
        "y": 100,
        "wires": [["process-sensor-data"]]
    },
    {
        "id": "process-sensor-data",
        "type": "function",
        "z": "mqtt-flow",
        "name": "Process Sensor Data",
        "func": "// Extract sensor info from topic\\nconst parts = msg.topic.split('/');\\nconst sensorId = parts[1];\\n\\n// Add metadata\\nmsg.payload.sensorId = sensorId;\\nmsg.payload.timestamp = new Date().toISOString();\\n\\n// Data validation\\nif (msg.payload.temperature !== undefined) {\\n    // Temperature range check\\n    if (msg.payload.temperature < -50 || msg.payload.temperature > 150) {\\n        msg.payload.anomaly = true;\\n    }\\n}\\n\\nreturn msg;",
        "outputs": 1,
        "noerr": 0,
        "initialize": "",
        "finalize": "",
        "x": 300,
        "y": 100,
        "wires": [["store-data", "check-alerts"]]
    },
    {
        "id": "store-data",
        "type": "function",
        "z": "mqtt-flow",
        "name": "Store to Context",
        "func": "// Store in flow context for persistence\\nlet sensorData = flow.get('sensorData') || {};\\nsensorData[msg.payload.sensorId] = msg.payload;\\nflow.set('sensorData', sensorData);\\n\\nreturn msg;",
        "outputs": 1,
        "x": 500,
        "y": 80,
        "wires": [[]]
    },
    {
        "id": "check-alerts",
        "type": "switch",
        "z": "mqtt-flow",
        "name": "Alert Conditions",
        "property": "payload.anomaly",
        "propertyType": "msg",
        "rules": [
            {"t": "true"}
        ],
        "checkall": "true",
        "repair": false,
        "outputs": 1,
        "x": 500,
        "y": 120,
        "wires": [["send-alert"]]
    },
    {
        "id": "send-alert",
        "type": "mqtt out",
        "z": "mqtt-flow",
        "name": "Send Alert",
        "topic": "alerts/sensor",
        "qos": "2",
        "retain": true,
        "broker": "mqtt-broker-config",
        "x": 700,
        "y": 120,
        "wires": []
    }
]
EOF
    
    log::success "MQTT template created"
}

# Create CoAP server flow
node_red::create_coap_template() {
    log::info "Creating CoAP integration template..."
    
    cat > "${HOME}/.local/share/node-red/coap-template.json" <<'EOF'
[
    {
        "id": "coap-flow",
        "type": "tab",
        "label": "CoAP IoT Integration",
        "disabled": false
    },
    {
        "id": "coap-server",
        "type": "coap-server",
        "z": "coap-flow",
        "name": "CoAP Server",
        "port": "5683",
        "x": 100,
        "y": 100,
        "wires": [["process-coap"]]
    },
    {
        "id": "process-coap",
        "type": "function",
        "z": "coap-flow",
        "name": "Process CoAP Request",
        "func": "// Handle different CoAP methods\nswitch(msg.method) {\n    case 'GET':\n        // Return sensor data\n        msg.payload = {\n            status: 'ok',\n            data: flow.get('sensorData') || {}\n        };\n        break;\n    case 'POST':\n        // Update sensor data\n        let data = flow.get('sensorData') || {};\n        data[msg.payload.id] = msg.payload.value;\n        flow.set('sensorData', data);\n        msg.payload = { status: 'updated' };\n        break;\n    case 'PUT':\n        // Replace sensor data\n        flow.set('sensorData', msg.payload);\n        msg.payload = { status: 'replaced' };\n        break;\n    case 'DELETE':\n        // Clear sensor data\n        flow.set('sensorData', {});\n        msg.payload = { status: 'cleared' };\n        break;\n}\n\nreturn msg;",
        "outputs": 1,
        "x": 300,
        "y": 100,
        "wires": [["coap-response"]]
    },
    {
        "id": "coap-response",
        "type": "coap-response",
        "z": "coap-flow",
        "name": "CoAP Response",
        "x": 500,
        "y": 100,
        "wires": []
    }
]
EOF
    
    log::success "CoAP template created"
}

# Create device discovery flow
node_red::create_device_discovery() {
    log::info "Creating device discovery flow..."
    
    cat > "${HOME}/.local/share/node-red/device-discovery.json" <<'EOF'
[
    {
        "id": "discovery-flow",
        "type": "tab",
        "label": "Device Discovery",
        "disabled": false
    },
    {
        "id": "scan-trigger",
        "type": "inject",
        "z": "discovery-flow",
        "name": "Scan Every 5 min",
        "repeat": "300",
        "crontab": "",
        "once": true,
        "topic": "",
        "payload": "{}",
        "x": 100,
        "y": 100,
        "wires": [["scan-network"]]
    },
    {
        "id": "scan-network",
        "type": "function",
        "z": "discovery-flow",
        "name": "Network Scanner",
        "func": "// Simulate network scanning for IoT devices\n// In production, use actual scanning libraries\n\nconst devices = [];\n\n// Check for MQTT devices\ndevices.push({\n    type: 'mqtt',\n    name: 'Temperature Sensor 1',\n    topic: 'sensors/temp1/data',\n    status: 'online'\n});\n\n// Check for CoAP devices\ndevices.push({\n    type: 'coap',\n    name: 'Light Controller',\n    address: 'coap://192.168.1.100',\n    status: 'online'\n});\n\n// Check for Modbus devices\ndevices.push({\n    type: 'modbus',\n    name: 'PLC Unit',\n    address: '192.168.1.150',\n    port: 502,\n    status: 'online'\n});\n\nmsg.payload = {\n    timestamp: new Date().toISOString(),\n    devices: devices,\n    count: devices.length\n};\n\nreturn msg;",
        "outputs": 1,
        "x": 300,
        "y": 100,
        "wires": [["store-devices", "notify-new"]]
    },
    {
        "id": "store-devices",
        "type": "function",
        "z": "discovery-flow",
        "name": "Store Device List",
        "func": "// Store discovered devices\nflow.set('discoveredDevices', msg.payload.devices);\nflow.set('lastScan', msg.payload.timestamp);\n\nreturn msg;",
        "outputs": 1,
        "x": 500,
        "y": 80,
        "wires": [[]]
    },
    {
        "id": "notify-new",
        "type": "function",
        "z": "discovery-flow",
        "name": "Check for New Devices",
        "func": "// Compare with previous scan\nconst prevDevices = flow.get('prevDevices') || [];\nconst currentDevices = msg.payload.devices;\n\nconst newDevices = currentDevices.filter(d => \n    !prevDevices.find(p => p.name === d.name)\n);\n\nif (newDevices.length > 0) {\n    msg.payload = {\n        alert: 'New devices discovered',\n        devices: newDevices\n    };\n    flow.set('prevDevices', currentDevices);\n    return msg;\n}\n\nflow.set('prevDevices', currentDevices);\nreturn null;",
        "outputs": 1,
        "x": 500,
        "y": 120,
        "wires": [["debug"]]
    },
    {
        "id": "debug",
        "type": "debug",
        "z": "discovery-flow",
        "name": "New Device Alert",
        "active": true,
        "console": false,
        "complete": "payload",
        "x": 700,
        "y": 120,
        "wires": []
    }
]
EOF
    
    log::success "Device discovery flow created"
}

# Deploy IoT integration flows
node_red::deploy_iot_flows() {
    log::info "Deploying IoT integration flows..."
    
    local url="http://localhost:${NODE_RED_PORT:-1880}"
    
    # Combine all templates
    local combined_flows='[]'
    
    for template in mqtt-template.json coap-template.json device-discovery.json; do
        local template_file="${HOME}/.local/share/node-red/${template}"
        if [[ -f "$template_file" ]]; then
            local template_content
            template_content=$(cat "$template_file")
            combined_flows=$(echo "$combined_flows" | jq ". + $template_content")
        fi
    done
    
    # Deploy flows
    local response
    response=$(timeout 10 curl -sf -X POST \
        -H "Content-Type: application/json" \
        -d "$combined_flows" \
        "${url}/flows" 2>/dev/null || echo "failed")
    
    if [[ "$response" != "failed" ]]; then
        log::success "IoT flows deployed successfully"
    else
        log::error "Failed to deploy IoT flows"
        return 1
    fi
}

# Setup complete IoT integration
node_red::setup_iot_integration() {
    log::header "Setting up IoT integration for Node-RED..."
    
    # Install nodes
    node_red::install_iot_nodes
    
    # Create templates
    node_red::create_mqtt_template
    node_red::create_coap_template
    node_red::create_device_discovery
    
    # Deploy flows
    node_red::deploy_iot_flows
    
    log::success "IoT integration setup complete"
    
    log::info "Available IoT endpoints:"
    echo "  MQTT: mqtt://${MQTT_BROKER_HOST}:${MQTT_BROKER_PORT}"
    echo "  CoAP: coap://localhost:${COAP_SERVER_PORT}"
    echo "  Modbus TCP: localhost:${MODBUS_TCP_PORT}"
}