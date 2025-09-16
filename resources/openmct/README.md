# Open MCT - NASA Mission Control Framework

NASA's open-source mission control framework for visualizing telemetry and building operations dashboards.

## Overview

Open MCT (Open Mission Control Technologies) is a next-generation mission control framework developed by NASA's Ames Research Center. Originally designed for spacecraft mission control, it provides a robust platform for visualizing time-series telemetry data from any source.

## Features

- **Real-time Telemetry Visualization** - Display live data streams with minimal latency
- **Historical Data Analysis** - Browse and analyze past telemetry with time navigation
- **Customizable Dashboards** - Drag-and-drop interface for creating mission-specific layouts
- **Plugin Architecture** - Extend with custom visualizations and data adapters
- **Multi-Source Integration** - Combine data from diverse telemetry providers

## Quick Start

```bash
# Install and start Open MCT
vrooli resource openmct manage install
vrooli resource openmct manage start --wait

# Check status
vrooli resource openmct status

# Access dashboard
open http://localhost:8099
```

## Usage

### Managing Telemetry Sources

```bash
# Register a new telemetry provider
vrooli resource openmct content add --type websocket --url ws://localhost:8080/telemetry

# List configured sources
vrooli resource openmct content list

# View telemetry stream details
vrooli resource openmct content get stream-1
```

### Sample Telemetry Streams

The resource comes with pre-configured demo telemetry streams:

1. **Satellite Telemetry** - Simulated spacecraft position and health data
2. **Sensor Network** - IoT sensor readings (temperature, humidity, pressure)
3. **System Metrics** - CPU, memory, and network utilization

### Creating Custom Dashboards

1. Access the Open MCT interface at `http://localhost:8099`
2. Click "Create" to start a new layout
3. Drag telemetry points from the tree to your layout
4. Arrange and resize widgets as needed
5. Save your layout for future use

## Integration Examples

### MQTT Integration
```bash
# Configure MQTT telemetry source
vrooli resource openmct content add \
  --type mqtt \
  --broker localhost:1883 \
  --topic "sensors/+/data"
```

### Traccar GPS Integration
```bash
# Connect to Traccar for vehicle tracking
vrooli resource openmct content add \
  --type traccar \
  --url http://localhost:8082 \
  --devices "all"
```

### Simulation Data
```bash
# Stream simulation outputs
vrooli resource openmct content execute \
  --source simulation \
  --file /path/to/simulation/output.csv
```

## API Reference

### REST API

Push telemetry data via HTTP:

```bash
curl -X POST http://localhost:8099/api/telemetry/mysensor/data \
  -H "Content-Type: application/json" \
  -d '{
    "timestamp": 1704096000000,
    "value": 23.5,
    "unit": "celsius"
  }'
```

### WebSocket API

Connect for real-time streaming:

```javascript
const ws = new WebSocket('ws://localhost:8099/api/telemetry/live');
ws.send(JSON.stringify({
  stream: 'mysensor',
  timestamp: Date.now(),
  value: 23.5
}));
```

## Configuration

### Environment Variables

- `OPENMCT_PORT` - Web interface port (default: 8099)
- `OPENMCT_DATA_DIR` - Data storage directory
- `OPENMCT_HISTORY_DAYS` - Days of history to retain (default: 30)
- `OPENMCT_AUTH_ENABLED` - Enable authentication (default: false)

### Advanced Configuration

Edit `config/defaults.sh` for detailed settings:

```bash
# Maximum concurrent telemetry streams
OPENMCT_MAX_STREAMS=100

# Historical data compression
OPENMCT_COMPRESS_HISTORY=true

# WebSocket buffer size
OPENMCT_WS_BUFFER_SIZE=65536
```

## Testing

```bash
# Run all tests
vrooli resource openmct test all

# Quick health check
vrooli resource openmct test smoke

# Integration tests
vrooli resource openmct test integration
```

## Troubleshooting

### Dashboard Not Loading
- Check if port 8099 is available: `lsof -i :8099`
- Verify health: `curl http://localhost:8099/health`
- Check logs: `vrooli resource openmct logs`

### Telemetry Not Updating
- Verify WebSocket connection in browser console
- Check network connectivity to telemetry sources
- Review adapter logs for errors

### Historical Data Issues
- Ensure sufficient disk space in data directory
- Check SQLite database integrity
- Verify time synchronization between sources

## Architecture

```
┌─────────────────────────────────────────────┐
│            Web Browser                      │
│         Open MCT Interface                  │
└─────────────────┬───────────────────────────┘
                  │ HTTP/WebSocket
┌─────────────────┴───────────────────────────┐
│         Open MCT Server (Node.js)           │
├─────────────────────────────────────────────┤
│   Telemetry │ Plugin  │ Storage │ API       │
│   Providers │ System  │ Layer   │ Routes    │
└──────┬──────┴────┬────┴────┬────┴───────────┘
       │           │         │
   External    Custom    SQLite
   Sources     Plugins   Database
```

## Resources

- [Official Open MCT Documentation](https://nasa.github.io/openmct/)
- [GitHub Repository](https://github.com/nasa/openmct)
- [Plugin Development Guide](https://github.com/nasa/openmct/blob/master/API.md)
- [Telemetry Tutorial](https://github.com/nasa/openmct-tutorial)

## License

Open MCT is open source software licensed under the Apache License, Version 2.0.