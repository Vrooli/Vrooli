# Eclipse Ditto - Digital Twin Platform

Eclipse Ditto provides a comprehensive framework for managing digital twins of IoT devices, enabling real-time state synchronization, policy-based access control, and seamless integration with various IoT protocols.

## Quick Start

```bash
# Install Eclipse Ditto
vrooli resource eclipse-ditto manage install

# Start services
vrooli resource eclipse-ditto manage start --wait

# Create a digital twin
vrooli resource eclipse-ditto twin create "device:sensor:001"

# Update twin properties
vrooli resource eclipse-ditto twin update "device:sensor:001" "attributes/temperature" "25.5"

# Query twins
vrooli resource eclipse-ditto twin query "eq(attributes/type,\"sensor\")"

# View service status
vrooli resource eclipse-ditto status
```

## Features

- **Digital Twin Management**: Create and manage virtual representations of physical devices
- **Real-time Synchronization**: WebSocket support for instant state updates
- **REST API**: Comprehensive HTTP API for all twin operations
- **Search Capabilities**: Query twins using RQL (Resource Query Language)
- **Policy Management**: Fine-grained access control for multi-tenant scenarios
- **Protocol Support**: REST, WebSocket, with optional MQTT/AMQP bridges
- **MongoDB Persistence**: Reliable storage with built-in MongoDB

## CLI Commands

### Lifecycle Management
```bash
# Install Eclipse Ditto
resource-eclipse-ditto manage install

# Start services
resource-eclipse-ditto manage start [--wait]

# Stop services
resource-eclipse-ditto manage stop

# Restart services
resource-eclipse-ditto manage restart

# Uninstall and cleanup
resource-eclipse-ditto manage uninstall
```

### Digital Twin Operations
```bash
# Create a new twin
resource-eclipse-ditto twin create <twin-id> [definition.json]

# Update twin properties
resource-eclipse-ditto twin update <twin-id> <property.path> <value>

# Query twins
resource-eclipse-ditto twin query [filter]

# Watch twin changes (WebSocket)
resource-eclipse-ditto twin watch <twin-id>

# Send command to twin
resource-eclipse-ditto twin command <twin-id> <feature> <command> [payload]
```

### Content Management
```bash
# List all twins
resource-eclipse-ditto content list

# Add twin from file
resource-eclipse-ditto content add <file.json>

# Get twin by ID
resource-eclipse-ditto content get <twin-id>

# Remove twin
resource-eclipse-ditto content remove <twin-id>
```

### Monitoring & Info
```bash
# Show service status
resource-eclipse-ditto status [--json]

# View logs
resource-eclipse-ditto logs [--tail N]

# Display credentials
resource-eclipse-ditto credentials

# Show configuration
resource-eclipse-ditto info [--json]
```

## Testing

```bash
# Run quick health check
vrooli resource eclipse-ditto test smoke

# Run integration tests
vrooli resource eclipse-ditto test integration

# Run all tests
vrooli resource eclipse-ditto test all
```

## Configuration

Configuration is managed through environment variables or the `config/defaults.sh` file:

- `DITTO_GATEWAY_PORT`: API gateway port (default: 8089)
- `DITTO_USERNAME`: Admin username (default: ditto)
- `DITTO_PASSWORD`: Admin password (default: ditto)
- `DITTO_MEMORY_LIMIT`: Container memory limit (default: 2g)
- `DITTO_ENABLE_WEBSOCKET`: Enable WebSocket support (default: true)
- `DITTO_ENABLE_MQTT`: Enable MQTT bridge (default: false)
- `DITTO_ENABLE_AMQP`: Enable AMQP bridge (default: false)

## Example Digital Twins

The resource includes example twins for common use cases:

### Industrial Sensor
```json
{
  "thingId": "industrial:sensor:temp-001",
  "attributes": {
    "type": "temperature-sensor",
    "location": "Factory Floor A"
  },
  "features": {
    "temperature": {
      "properties": {
        "value": 22.5,
        "unit": "celsius"
      }
    }
  }
}
```

### Connected Vehicle
```json
{
  "thingId": "vehicle:car:tesla-001",
  "attributes": {
    "manufacturer": "Tesla",
    "model": "Model 3"
  },
  "features": {
    "battery": {
      "properties": {
        "level": 75,
        "range": 280
      }
    }
  }
}
```

## API Examples

### Create Twin
```bash
curl -X PUT \
  -u ditto:ditto \
  -H "Content-Type: application/json" \
  -d '{"thingId":"my:device:001","attributes":{"name":"My Device"}}' \
  http://localhost:8089/api/2/things/my:device:001
```

### Update Feature
```bash
curl -X PUT \
  -u ditto:ditto \
  -H "Content-Type: application/json" \
  -d '{"temperature":25.5}' \
  http://localhost:8089/api/2/things/my:device:001/features/sensor/properties/temperature
```

### Search Twins
```bash
curl -u ditto:ditto \
  "http://localhost:8089/api/2/search/things?filter=eq(attributes/type,\"sensor\")"
```

## Integration with Vrooli Resources

Eclipse Ditto integrates seamlessly with other Vrooli resources:

- **N8n**: Trigger workflows based on twin state changes
- **Redis**: Cache frequently accessed twin states
- **Qdrant**: Store twin embeddings for semantic search
- **Postgres**: Analyze twin data with SQL queries
- **Traccar**: Sync vehicle location data to vehicle twins
- **FarmOS**: Update agricultural equipment twins
- **OpenEMR**: Maintain medical device twins

## Architecture

```
┌─────────────┐     ┌──────────┐     ┌────────────┐
│   Gateway   │────▶│  Things  │────▶│  MongoDB   │
└─────────────┘     └──────────┘     └────────────┘
       │                  │
       ▼                  ▼
┌─────────────┐     ┌──────────┐
│   Search    │     │ Policies │
└─────────────┘     └──────────┘
       │
       ▼
┌─────────────┐
│Connectivity │
└─────────────┘
```

## Troubleshooting

### Service Won't Start
- Check Docker is running: `docker info`
- Verify port 8089 is available: `lsof -i :8089`
- Check logs: `resource-eclipse-ditto logs`

### Cannot Create Twins
- Verify authentication: `resource-eclipse-ditto credentials`
- Check service health: `resource-eclipse-ditto status`
- Ensure MongoDB is running: `docker ps | grep mongodb`

### WebSocket Connection Failed
- Verify WebSocket is enabled in configuration
- Check firewall rules for port 8089
- Test with: `curl -I http://localhost:8089/ws/2`

## Resources

- [Official Documentation](https://eclipse.dev/ditto/)
- [API Documentation](https://eclipse.dev/ditto/httpapi-overview.html)
- [GitHub Repository](https://github.com/eclipse-ditto/ditto)
- [Docker Hub](https://hub.docker.com/u/eclipse)