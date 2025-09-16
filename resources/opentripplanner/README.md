# OpenTripPlanner Resource

Multimodal trip planning engine for transit, biking, walking, and demand-responsive transport.

## Quick Start

```bash
# Install and start
vrooli resource opentripplanner manage install
vrooli resource opentripplanner manage start --wait

# Check status
vrooli resource opentripplanner status

# Plan a trip
curl "http://localhost:8080/otp/routers/default/plan?fromPlace=45.5,-122.6&toPlace=45.52,-122.65&mode=TRANSIT,WALK"

# View available data
vrooli resource opentripplanner content list
```

## Features

### Core Capabilities
- **Multimodal Routing**: Transit, walk, bike, car, and combinations
- **GTFS Support**: Import and process transit feed data
- **OpenStreetMap Integration**: Street network for walking/biking
- **Real-time Updates**: GTFS-RT for live transit positions
- **Isochrone Analysis**: Reachability maps for accessibility

### Integration Points
- **PostGIS**: Spatial analysis and route storage
- **Qdrant**: Semantic search for locations
- **Traccar**: Fleet tracking integration
- **N8n**: Workflow automation
- **Eclipse Ditto**: Digital twin synchronization

## Usage Examples

### Import GTFS Data
```bash
# Add transit feed
vrooli resource opentripplanner content add \
  --type gtfs \
  --file portland-transit.zip \
  --name portland

# Build routing graph
vrooli resource opentripplanner content execute \
  --action build-graph \
  --name portland
```

### Plan Multimodal Trip
```bash
# Bus + Rail journey
curl "http://localhost:8080/otp/routers/default/plan?\
fromPlace=45.5231,-122.6765&\
toPlace=45.5152,-122.6784&\
mode=TRANSIT,WALK&\
time=8:00am"
```

### Generate Isochrones
```bash
# 30-minute reachability from location
curl "http://localhost:8080/otp/routers/default/isochrone?\
fromPlace=45.5231,-122.6765&\
mode=TRANSIT,WALK&\
cutoffSec=1800"
```

## Configuration

### Environment Variables
```bash
OTP_PORT=8080              # API port
OTP_HEAP_SIZE=2G          # JVM heap memory
OTP_BUILD_TIMEOUT=300     # Graph build timeout (seconds)
OTP_CACHE_DIR=/var/cache/otp  # Graph cache location
```

### Graph Configuration
```yaml
# config/router-config.json
{
  "routingDefaults": {
    "walkSpeed": 1.34,
    "bikeSpeed": 5.0,
    "carSpeed": 20.0,
    "maxTransfers": 3
  }
}
```

## API Endpoints

### Core Endpoints
- `GET /health` - Health check
- `GET /otp/routers` - List available routers
- `GET /otp/routers/{id}/plan` - Plan journey
- `GET /otp/routers/{id}/isochrone` - Generate isochrone
- `GET /otp/routers/{id}/index/stops` - List transit stops

### Planning Parameters
- `fromPlace` - Origin coordinates (lat,lon)
- `toPlace` - Destination coordinates
- `mode` - Travel modes (TRANSIT,WALK,BIKE,CAR)
- `time` - Departure time
- `maxWalkDistance` - Maximum walking distance
- `wheelchair` - Accessibility requirements

## Testing

```bash
# Run all tests
vrooli resource opentripplanner test all

# Quick health check
vrooli resource opentripplanner test smoke

# Full integration tests
vrooli resource opentripplanner test integration
```

## Troubleshooting

### Common Issues

**Graph Build Fails**
- Check GTFS data validity
- Ensure sufficient memory allocated
- Verify OSM data coverage matches GTFS area

**Slow Route Planning**
- Increase JVM heap size
- Enable graph caching
- Reduce search timeout limits

**No Routes Found**
- Verify graph contains requested area
- Check time/date parameters
- Ensure modes are available in region

## Performance Tuning

### Memory Optimization
```bash
# Increase heap for large regions
export OTP_HEAP_SIZE=4G

# Enable garbage collection logging
export OTP_JVM_OPTS="-XX:+UseG1GC -XX:MaxGCPauseMillis=200"
```

### Caching Strategy
```bash
# Enable Redis caching for repeated queries
vrooli resource opentripplanner config set \
  --cache-backend redis \
  --cache-ttl 3600
```

## Integration Examples

### With PostGIS
```bash
# Store route analytics
vrooli resource opentripplanner content execute \
  --analyze-routes \
  --store-postgis \
  --table-name transit_routes
```

### With N8n Workflows
```javascript
// N8n webhook for trip planning
const otp = await $http.get('http://opentripplanner:8080/otp/routers/default/plan', {
  params: {
    fromPlace: $json.origin,
    toPlace: $json.destination,
    mode: 'TRANSIT,WALK'
  }
});
```

## License

OpenTripPlanner is licensed under LGPL. This resource wrapper is part of the Vrooli ecosystem.