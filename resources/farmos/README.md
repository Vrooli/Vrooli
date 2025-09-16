# farmOS Resource

farmOS is an open-source agricultural management platform that provides comprehensive farm management capabilities including crop tracking, livestock management, equipment monitoring, and activity logging. As a Vrooli resource, it anchors the Agriculture/Food Systems branch, enabling scenarios to orchestrate farm operations, integrate IoT sensors, and manage supply chains.

## Quick Start

```bash
# Install farmOS
vrooli resource farmos manage install

# Start farmOS with demo data
vrooli resource farmos manage start --wait

# Check status
vrooli resource farmos status

# View credentials
vrooli resource farmos credentials

# Access web interface
# Default: http://localhost:8004
# Username: admin / Password: admin
```

## Features

### Core Capabilities ✅
- **Farm Management**: Fields, crops, livestock, equipment tracking
- **Activity Logging**: Plant, harvest, treatment, observation logs  
- **Asset Tracking**: Equipment, animals, plants, land management
- **Data Export**: CSV, JSON export of all farm records
- **API Access**: REST API v2.x with Basic authentication
- **Demo Data**: Automatic seeding with sample farm data
- **Auto-Installation**: Automatic setup completion on first startup

### Integrations
- **IoT Sensors**: MQTT/Node-RED for sensor data ingestion
- **Time Series**: QuestDB for sensor analytics
- **Digital Twins**: Eclipse Ditto for equipment modeling
- **Spatial Data**: PostgreSQL/PostGIS for field mapping
- **Finance**: Mifos for agricultural loans and insurance

## CLI Commands

### Management
```bash
# Lifecycle management
resource-farmos manage install       # Install farmOS
resource-farmos manage start [--wait] # Start services
resource-farmos manage stop          # Stop services
resource-farmos manage restart       # Restart services
resource-farmos manage uninstall     # Remove farmOS

# Status and monitoring
resource-farmos status               # Show service status
resource-farmos logs [--tail N]      # View logs
resource-farmos credentials          # Display API credentials
```

### Farm Operations
```bash
# Field management
resource-farmos farm create-field --name "North Field" --size 10 --unit acres
resource-farmos farm list-fields

# Activity logging
resource-farmos farm log-activity --type planting --field "North Field" --crop "Corn"
resource-farmos farm log-activity --type harvest --field "South Field" --quantity 500 --unit bushels

# Data export
resource-farmos farm export --format csv --output farm_records.csv
resource-farmos farm export --format json --type assets

# Demo data
resource-farmos farm seed-demo      # Load demo farm data
```

### IoT Integration (P1 - Coming Soon)
```bash
# Connect sensors
resource-farmos iot connect --broker mqtt://localhost:1883
resource-farmos iot ingest --sensor soil-moisture --field "North Field"
resource-farmos iot sync             # Sync sensor data to farmOS
```

### Testing
```bash
# Run tests
resource-farmos test all             # Run all tests
resource-farmos test smoke           # Quick health check
resource-farmos test integration     # Full functionality tests
resource-farmos test unit            # Library function tests
```

## Configuration

### Environment Variables
```bash
# Service configuration
FARMOS_PORT=8004                     # Web interface port
FARMOS_HOST=localhost                # Hostname
FARMOS_BASE_URL=http://localhost:8004 # Full URL

# Admin credentials
FARMOS_ADMIN_USER=admin              # Admin username
FARMOS_ADMIN_PASSWORD=admin          # Admin password
FARMOS_ADMIN_EMAIL=admin@vrooli.local # Admin email

# Features
FARMOS_DEMO_DATA=true                # Enable demo data
FARMOS_OAUTH_ENABLED=true            # Enable OAuth2
FARMOS_API_ENABLED=true              # Enable REST API

# IoT Integration (optional)
FARMOS_MQTT_ENABLED=false            # Enable MQTT
FARMOS_MQTT_BROKER=                  # MQTT broker address
FARMOS_QUESTDB_ENABLED=false         # QuestDB integration
```

## API Usage

### Authentication
```bash
# Get OAuth token
curl -X POST http://localhost:8004/oauth/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=password&username=admin&password=admin"
```

### CRUD Operations
```bash
# Create a field
curl -X POST http://localhost:8004/api/field \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "North Field", "size": 10, "unit": "acres"}'

# List assets
curl http://localhost:8004/api/asset \
  -H "Authorization: Bearer $TOKEN"

# Log an activity
curl -X POST http://localhost:8004/api/log \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"type": "planting", "field": "North Field", "crop": "Corn"}'
```

## Integration Examples

### With QuestDB for Sensor Analytics
```bash
# Enable QuestDB integration
export FARMOS_QUESTDB_ENABLED=true

# Sensor data automatically flows to QuestDB
# Query in QuestDB: SELECT * FROM farm_sensors WHERE field='North Field'
```

### With Node-RED for IoT Workflows
```bash
# farmOS nodes available in Node-RED palette
# Create flows to:
# - Ingest sensor data via MQTT
# - Trigger irrigation based on soil moisture
# - Alert on equipment issues
```

### With Eclipse Ditto for Digital Twins
```bash
# Model farm equipment as digital twins
# Track tractor location, fuel level, maintenance
# Sync twin state to farmOS assets
```

## Architecture

```
┌─────────────────────────────────────────┐
│           farmOS Web Interface          │
├─────────────────────────────────────────┤
│          farmOS Core (Drupal)           │
├─────────────────────────────────────────┤
│            REST API v2.x                │
├──────────────┬──────────────────────────┤
│   PostgreSQL │      Redis (Optional)     │
└──────────────┴──────────────────────────┘
        ↓                ↓
   [Scenarios]      [IoT Sensors]
```

## Use Cases

### Precision Agriculture
- Sensor-driven irrigation management
- Soil health monitoring
- Weather-based planting decisions
- Yield optimization

### Supply Chain Traceability  
- Farm-to-table tracking
- Organic certification records
- Food safety compliance
- Export documentation

### Financial Integration
- Crop insurance claims
- Agricultural loans
- Subsidy applications  
- Cost-benefit analysis

### Multi-Farm Cooperatives
- Shared equipment management
- Bulk purchasing coordination
- Knowledge sharing
- Market price aggregation

## Troubleshooting

### Service Won't Start
```bash
# Check Docker status
docker ps -a | grep farmos

# View logs
resource-farmos logs --tail 100

# Verify port availability
lsof -i :8004
```

### API Authentication Failed
```bash
# Verify OAuth is enabled
echo $FARMOS_OAUTH_ENABLED

# Check admin credentials
resource-farmos credentials

# Test with basic auth first
curl -u admin:admin http://localhost:8004/api
```

### Demo Data Not Loading
```bash
# Enable demo data
export FARMOS_DEMO_DATA=true

# Restart services
resource-farmos manage restart

# Wait for initialization (2-3 minutes)
# Check logs for progress
resource-farmos logs --follow
```

## Performance Considerations

- **Startup Time**: 60-90 seconds for full initialization
- **Demo Data**: Additional 2-3 minutes on first start
- **API Response**: <500ms for most operations
- **Export Time**: ~5 seconds per 1000 records
- **Memory Usage**: ~1.5GB typical, 2GB recommended

## Security

- OAuth2 authentication for API access
- Environment-based configuration (no hardcoded secrets)
- Docker network isolation
- CORS configuration for cross-origin requests
- Regular security updates via Docker images

## Future Enhancements

### P1 Features (In Development)
- Full IoT sensor integration via MQTT
- QuestDB time-series pipeline
- Redis caching layer
- Open Data Cube integration

### P2 Features (Planned)
- Mifos financial dashboards
- Geospatial visualization overlays
- Multi-farm cooperative templates
- Mobile application support

## Related Resources

- **node-red**: IoT workflow automation
- **questdb**: Time-series analytics
- **eclipse-ditto**: Digital twin management
- **postgres**: Spatial database with PostGIS
- **mifos**: Microfinance integration

## Support

For issues or questions:
1. Check the troubleshooting section above
2. Review logs with `resource-farmos logs`
3. Consult the [farmOS documentation](https://farmos.org/docs)
4. File issues in the Vrooli repository

## License

farmOS is licensed under GPL-2.0. The Vrooli resource wrapper is part of the Vrooli ecosystem.