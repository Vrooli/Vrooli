# Node-RED IoT Integration Guide

## Overview

Node-RED provides comprehensive IoT device integration capabilities through support for multiple protocols and device types. This guide covers setup, configuration, and best practices for IoT workflows.

## Supported Protocols

### MQTT (Message Queuing Telemetry Transport)
- **Use Case**: Lightweight pub/sub messaging for sensors and actuators
- **Port**: 1883 (default)
- **QoS Levels**: 0 (at most once), 1 (at least once), 2 (exactly once)

### CoAP (Constrained Application Protocol)
- **Use Case**: RESTful protocol for constrained devices
- **Port**: 5683 (default)
- **Methods**: GET, POST, PUT, DELETE, OBSERVE

### Modbus
- **Use Case**: Industrial automation and SCADA systems
- **Variants**: Modbus TCP, Modbus RTU
- **Port**: 502 (Modbus TCP)

### OPC-UA
- **Use Case**: Industrial interoperability standard
- **Security**: Certificate-based authentication
- **Discovery**: Automatic endpoint discovery

## Quick Setup

### 1. Install IoT Nodes
```bash
# Install all IoT-related nodes
resource-node-red iot-nodes

# Or install individually
docker exec node-red npm install node-red-contrib-mqtt-broker
docker exec node-red npm install node-red-contrib-coap
docker exec node-red npm install node-red-contrib-modbus
```

### 2. Complete IoT Setup
```bash
# Automated setup with templates and examples
resource-node-red iot-setup
```

This command:
- Installs all IoT nodes
- Creates MQTT broker configuration
- Deploys CoAP server
- Sets up device discovery flow
- Configures data persistence

## MQTT Integration

### Basic MQTT Flow
```json
{
    "id": "mqtt-in",
    "type": "mqtt in",
    "topic": "sensors/+/temperature",
    "qos": "1",
    "broker": "mqtt-broker-config"
}
```

### Topic Patterns
- `sensors/+/data` - Single-level wildcard
- `sensors/#` - Multi-level wildcard
- `sensors/temp1/data` - Specific sensor

### Example: Temperature Monitoring
1. Subscribe to sensor topics
2. Process and validate data
3. Store in context or database
4. Trigger alerts on thresholds

## CoAP Integration

### CoAP Server Setup
```javascript
// CoAP endpoint handler
msg.method // GET, POST, PUT, DELETE
msg.url    // Request URL
msg.payload // Request body
```

### Resource Discovery
- Well-known URI: `/.well-known/core`
- Observe mode for real-time updates
- Blockwise transfer for large payloads

## Device Management

### Auto-Discovery Flow
The device discovery flow automatically:
- Scans network every 5 minutes
- Detects MQTT, CoAP, and Modbus devices
- Alerts on new device detection
- Maintains device inventory

### Device Registry
```javascript
// Access discovered devices
const devices = flow.get('discoveredDevices');
// Returns array of device objects with type, name, address, status
```

## Data Processing Patterns

### 1. Edge Computing
Process data locally before sending to cloud:
```javascript
// Aggregate sensor readings
const readings = msg.payload;
const average = readings.reduce((a,b) => a+b) / readings.length;
msg.payload = { avg: average, count: readings.length };
return msg;
```

### 2. Protocol Translation
Convert between different IoT protocols:
- MQTT → HTTP REST API
- CoAP → MQTT
- Modbus → WebSocket

### 3. Data Enrichment
Add metadata and context:
```javascript
msg.payload.timestamp = new Date().toISOString();
msg.payload.location = global.get('deviceLocation');
msg.payload.deviceId = msg.topic.split('/')[1];
```

## Performance Optimization

### Message Batching
Reduce overhead by batching messages:
```javascript
let batch = context.get('batch') || [];
batch.push(msg.payload);

if (batch.length >= 100) {
    msg.payload = batch;
    context.set('batch', []);
    return msg;
}
context.set('batch', batch);
return null; // Don't send yet
```

### Connection Pooling
- Reuse MQTT connections
- Implement connection retry logic
- Use QoS 0 for non-critical data

## Security Best Practices

### MQTT Security
- Enable TLS/SSL encryption
- Use username/password authentication
- Implement ACL for topic access
- Regular credential rotation

### CoAP Security
- DTLS for secure CoAP (CoAPS)
- Pre-shared keys or certificates
- Resource-level access control

### General IoT Security
- Network segmentation
- Device authentication
- Data encryption in transit
- Regular firmware updates
- Anomaly detection

## Troubleshooting

### Common Issues

#### MQTT Connection Failed
```bash
# Check broker status
docker exec node-red mosquitto_sub -t '#' -v

# Verify port accessibility
nc -zv localhost 1883
```

#### CoAP Timeout
```bash
# Test CoAP endpoint
coap-client -m get coap://localhost:5683/.well-known/core

# Check firewall rules
sudo ufw status
```

#### High Message Loss
- Reduce publishing frequency
- Increase QoS level
- Check network stability
- Monitor broker load

## Integration Examples

### Smart Home Automation
```javascript
// Motion sensor triggers lights
if (msg.payload.motion === true) {
    return { 
        topic: 'lights/living-room/set',
        payload: { state: 'on', brightness: 100 }
    };
}
```

### Industrial Monitoring
```javascript
// Temperature anomaly detection
if (msg.payload.temperature > threshold) {
    // Send alert
    node.send([null, {
        topic: 'alerts/temperature',
        payload: { 
            severity: 'high',
            value: msg.payload.temperature,
            threshold: threshold
        }
    }]);
}
```

### Agricultural IoT
```javascript
// Irrigation control based on soil moisture
if (msg.payload.moisture < 30) {
    return {
        topic: 'irrigation/zone1/control',
        payload: { action: 'start', duration: 600 }
    };
}
```

## Performance Metrics

Monitor IoT flow performance:
```bash
resource-node-red monitor
```

Key metrics:
- Message throughput (msg/sec)
- Processing latency (ms)
- Error rate (%)
- Device connectivity status

## Advanced Topics

### Custom Protocol Implementation
Implement custom IoT protocols using:
- TCP/UDP nodes for raw socket communication
- Function nodes for protocol logic
- Buffer manipulation for binary protocols

### Time-Series Data Storage
Integration with databases:
- InfluxDB for metrics
- TimescaleDB for SQL queries
- Prometheus for monitoring

### Edge AI Integration
- TensorFlow.js for in-flow ML
- ONNX runtime for model inference
- Anomaly detection algorithms

## Resources

- [MQTT Specification](https://mqtt.org/mqtt-specification/)
- [CoAP RFC 7252](https://tools.ietf.org/html/rfc7252)
- [Modbus Protocol](https://modbus.org/specs.php)
- [OPC-UA Standards](https://opcfoundation.org/developer-tools/specifications-unified-architecture)
- [Node-RED IoT Cookbook](https://cookbook.nodered.org/iot/)