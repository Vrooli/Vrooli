# Traccar Fleet & Telematics Platform

## Overview

Traccar is an open-source GPS tracking server that provides comprehensive fleet management and telematics capabilities. It supports over 3000 different GPS tracking device protocols and offers web, REST, and WebSocket interfaces for real-time location tracking and fleet management.

## Features

- **Multi-Protocol Support**: Compatible with 3000+ GPS device protocols
- **Real-Time Tracking**: Live position updates via WebSocket API
- **Fleet Management**: Complete device lifecycle management
- **Geofencing**: Location-based alerts and automation
- **Reporting**: Comprehensive reports for routes, trips, and stops
- **API-First**: Full REST API for integration
- **WebSocket Support**: Real-time position streaming
- **Docker Deployment**: Containerized for easy scaling

## Quick Start

### Installation
```bash
# Install Traccar and dependencies
vrooli resource traccar manage install

# Start the server
vrooli resource traccar manage start --wait

# Check status
vrooli resource traccar status
```

### Basic Usage
```bash
# Seed demo data (5 vehicles with recent positions)
vrooli resource traccar content execute --name demo

# Create a GPS device
vrooli resource traccar device create --name "Vehicle-001" --type "car"

# Send GPS position
vrooli resource traccar track push \
  --device "Vehicle-001" \
  --lat 37.7749 \
  --lon -122.4194 \
  --speed 45

# View position history
vrooli resource traccar track history --device "Vehicle-001" --days 7

# Start live tracking
vrooli resource traccar track live --device "Vehicle-001" --duration 300
```

## Configuration

### Environment Variables
```bash
# Service Configuration
TRACCAR_PORT=8082              # HTTP port
TRACCAR_HOST=localhost         # Service hostname
TRACCAR_VERSION=5.12          # Traccar version

# Database Configuration
TRACCAR_DB_HOST=localhost      # PostgreSQL host
TRACCAR_DB_PORT=5433          # PostgreSQL port
TRACCAR_DB_NAME=traccar       # Database name
TRACCAR_DB_USER=traccar       # Database user
TRACCAR_DB_PASSWORD=traccar123 # Database password

# Admin Credentials
TRACCAR_ADMIN_EMAIL=admin@example.com
TRACCAR_ADMIN_PASSWORD=admin
```

### Configuration Files
- `config/defaults.sh` - Default environment variables
- `config/runtime.json` - Runtime dependencies and startup order
- `config/schema.json` - Configuration validation schema

## API Reference

### Device Management
```bash
# Create device
vrooli resource traccar device create \
  --name "Fleet-001" \
  --type "truck" \
  --phone "+1234567890"

# List all devices
vrooli resource traccar device list

# Update device
vrooli resource traccar device update \
  --id 123 \
  --name "Fleet-001-Updated"

# Delete device
vrooli resource traccar device delete --id 123
```

### GPS Tracking
```bash
# Push single position
vrooli resource traccar track push \
  --device "Fleet-001" \
  --lat 37.7749 \
  --lon -122.4194 \
  --speed 60 \
  --bearing 180 \
  --altitude 100

# Get position history
vrooli resource traccar track history \
  --device "Fleet-001" \
  --from "2025-01-01T00:00:00Z" \
  --to "2025-01-16T00:00:00Z"

# Live tracking (polls every 5 seconds)
vrooli resource traccar track live \
  --device "Fleet-001" \
  --duration 600
```

### Content Management
```bash
# Add device from JSON file
vrooli resource traccar content add \
  --file device.json \
  --type device

# List all devices
vrooli resource traccar content list

# Export device configuration
vrooli resource traccar content get \
  --name "Fleet-001" \
  --output device-config.json

# Remove device
vrooli resource traccar content remove \
  --name "Fleet-001" \
  --force
```

## Integration Examples

### Node-RED Integration
```javascript
// Node-RED function node for Traccar webhook
msg.traccarEvent = msg.payload;
if (msg.payload.type === 'deviceOnline') {
    msg.payload = {
        device: msg.payload.device.name,
        status: 'online',
        timestamp: new Date().toISOString()
    };
}
return msg;
```

### N8n Workflow
```json
{
  "nodes": [{
    "name": "Traccar Webhook",
    "type": "n8n-nodes-base.webhook",
    "parameters": {
      "path": "traccar",
      "responseMode": "onReceived",
      "responseData": "success"
    }
  }]
}
```

### Eclipse Ditto Digital Twin
```bash
# Create vehicle twin with Traccar data
curl -X PUT http://localhost:8089/api/2/things/vehicle:fleet-001 \
  -H "Content-Type: application/json" \
  -d '{
    "attributes": {
      "trackingSystem": "traccar",
      "deviceId": "123"
    },
    "features": {
      "location": {
        "properties": {
          "latitude": 37.7749,
          "longitude": -122.4194,
          "speed": 45
        }
      }
    }
  }'
```

## Testing

```bash
# Run all tests
vrooli resource traccar test all

# Run specific test phases
vrooli resource traccar test smoke      # Quick health check (30s)
vrooli resource traccar test integration # Full functionality (120s)
vrooli resource traccar test unit       # Library validation (60s)
```

## Troubleshooting

### Common Issues

#### Server Not Starting
```bash
# Check logs
vrooli resource traccar logs --tail 100

# Verify PostgreSQL is running
vrooli resource postgres status

# Check port availability
sudo lsof -i :8082
```

#### Authentication Failed
```bash
# Reset admin password in database
docker exec -it vrooli-traccar psql -U traccar -d traccar \
  -c "UPDATE tc_users SET hashedpassword='admin' WHERE email='admin@example.com';"
```

#### Device Not Reporting
```bash
# Check device status
vrooli resource traccar device list

# Test position update manually
vrooli resource traccar track push \
  --device "Device-Name" \
  --lat 0 --lon 0
```

## Advanced Features

### Geofencing
```bash
# Create geofence (via API)
curl -X POST http://localhost:8082/api/geofences \
  -u admin@example.com:admin \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Warehouse",
    "area": "CIRCLE (37.7749 -122.4194, 500)"
  }'
```

### Reports
```bash
# Generate route report
curl -G http://localhost:8082/api/reports/route \
  -u admin@example.com:admin \
  --data-urlencode "deviceId=123" \
  --data-urlencode "from=2025-01-01T00:00:00Z" \
  --data-urlencode "to=2025-01-16T00:00:00Z"
```

### WebSocket Real-Time Updates
```javascript
// JavaScript WebSocket client
const ws = new WebSocket('ws://localhost:8082/api/socket');
ws.onmessage = (event) => {
    const data = JSON.parse(event.data);
    if (data.positions) {
        console.log('New position:', data.positions);
    }
};
```

## Performance Optimization

### Database Indexing
```sql
-- Add indexes for better query performance
CREATE INDEX idx_positions_deviceid ON tc_positions(deviceid);
CREATE INDEX idx_positions_devicetime ON tc_positions(devicetime);
CREATE INDEX idx_positions_servertime ON tc_positions(servertime);
```

### Container Resources
```bash
# Increase memory limits for high-load scenarios
docker update --memory="2g" --memory-swap="4g" vrooli-traccar
```

## Security Considerations

- Always change default admin credentials
- Use HTTPS in production environments
- Implement API rate limiting for public deployments
- Regular backup of PostgreSQL database
- Monitor for unauthorized device registrations

## Related Resources

- [Traccar Official Documentation](https://www.traccar.org/documentation/)
- [API Documentation](https://www.traccar.org/api-reference/)
- [Device Protocols](https://www.traccar.org/protocols/)
- [Forum & Community](https://www.traccar.org/forums/)

## License

Traccar is licensed under Apache License 2.0. This Vrooli resource wrapper maintains compatibility with the original license.

---
*Resource Version: 1.0.0*
*Traccar Version: 5.12*
*Last Updated: 2025-01-16*