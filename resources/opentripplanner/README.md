# OpenTripPlanner Resource

Multimodal trip planning engine for transit, biking, walking, and demand-responsive transport.

## Quick Start

```bash
# Install and start
vrooli resource opentripplanner manage install
vrooli resource opentripplanner manage start --wait

# Check status
vrooli resource opentripplanner status

# Plan a trip (using CLI with GraphQL API)
vrooli resource opentripplanner content execute --action plan-trip \
  --from-lat 45.5 --from-lon -122.6 \
  --to-lat 45.52 --to-lon -122.65 \
  --modes "bus,rail,tram"

# View available data
vrooli resource opentripplanner content list
```

## Features

### Core Capabilities
- **Multimodal Routing**: Transit, walk, bike, car, and combinations
- **GTFS Support**: Import and process transit feed data
- **OpenStreetMap Integration**: Street network for walking/biking
- **Real-time Updates**: GTFS-RT feed support for live transit positions
- **Isochrone Analysis**: Reachability maps for accessibility
- **OTP v2.9 API**: Latest version with debug UI interface
- **PostGIS Export**: Export transit stops and routes to spatial database

### Integration Points
- **PostGIS**: Spatial analysis and route storage
- **Qdrant**: Semantic search for locations
- **Traccar**: Fleet tracking integration
- **N8n**: Workflow automation
- **Eclipse Ditto**: Digital twin synchronization

## Usage Examples

### Import GTFS Data
```bash
# Add transit feed from file or URL
vrooli resource opentripplanner content add \
  --type gtfs \
  --file portland-transit.zip \
  --name portland

# Or download directly
vrooli resource opentripplanner content add \
  --type gtfs \
  --url https://developer.trimet.org/schedule/gtfs.zip \
  --name trimet

# Add GTFS-RT real-time feed
vrooli resource opentripplanner content add \
  --type gtfs-rt \
  --name trimet-realtime \
  --url https://developer.trimet.org/ws/V1/FeedSpecAlerts

# Build routing graph
vrooli resource opentripplanner content execute \
  --action build-graph
```

### Access Debug UI
```bash
# Get the configured port
OTP_PORT=$(./scripts/resources/port_registry.sh opentripplanner | grep opentripplanner | awk '{print $3}')

# Open in browser
xdg-open "http://localhost:${OTP_PORT}/"

# Or check API status
curl "http://localhost:${OTP_PORT}/otp/" | jq '.version'
```

### Export to PostGIS
```bash
# Export transit stops to PostGIS
vrooli resource opentripplanner content execute \
  --action export-stops

# Analyze routes with spatial storage
vrooli resource opentripplanner content execute \
  --action analyze-routes \
  --store-results
```

### Plan a Trip
```bash
# Plan multimodal trip using GraphQL API
vrooli resource opentripplanner content execute --action plan-trip \
  --from-lat 45.5231 --from-lon -122.6765 \
  --to-lat 45.5234 --to-lon -122.6762 \
  --modes "bus,rail,tram" \
  --format summary
```

## Configuration

### Environment Variables
```bash
OTP_PORT                          # API port (from port_registry.sh)
OTP_HEAP_SIZE=2G                  # JVM heap memory
OTP_BUILD_TIMEOUT=300             # Graph build timeout (seconds)
OTP_DATA_DIR=~/.vrooli/opentripplanner/data   # Data directory
OTP_CACHE_DIR=~/.vrooli/opentripplanner/cache # Graph cache location
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

### Core Endpoints (OTP v2.9)
- `GET /` - Debug UI interface
- `GET /otp/` - API info and version
- `POST /otp/transmodel/v3` - GraphQL API for trip planning (Transmodel v3)

**Note:** OTP v2.9 uses GraphQL for trip planning. Use the CLI command `content execute --action plan-trip` for trip planning instead of direct API calls. The old REST endpoints (`/otp/routers/default/plan`) are deprecated in v2.9.

### Trip Planning (CLI)
Use the CLI command for trip planning with these parameters:
- `--from-lat` / `--from-lon` - Origin coordinates
- `--to-lat` / `--to-lon` - Destination coordinates
- `--modes` - Comma-separated transport modes (bus, rail, tram, metro, ferry)
- `--format` - Output format: `summary` (default) or `json`

Example:
```bash
vrooli resource opentripplanner content execute --action plan-trip \
  --from-lat 45.5152 --from-lon -122.6784 \
  --to-lat 45.5234 --to-lon -122.6762 \
  --modes "bus,rail" --format summary
```

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
# Export transit stops to PostGIS
vrooli resource opentripplanner content execute \
  --action export-stops

# Query stops in PostGIS
docker exec -i vrooli-postgres psql -U postgres -d postgres \
  -c "SELECT COUNT(*) FROM opentripplanner.transit_stops;"
```

### With N8n Workflows
```javascript
// N8n webhook for trip planning using GraphQL
// Note: Use Docker network hostname 'opentripplanner' or localhost with ${OTP_PORT}
const query = `{
  trip(
    from: { coordinates: { latitude: ${$json.fromLat}, longitude: ${$json.fromLon} } }
    to: { coordinates: { latitude: ${$json.toLat}, longitude: ${$json.toLon} } }
    modes: { transportModes: [{ transportMode: bus }, { transportMode: rail }] }
  ) {
    tripPatterns {
      duration
      walkDistance
      legs { mode distance }
    }
  }
}`;

// Use Docker network hostname (container-to-container communication)
// Note: 8080 is the container's internal port, not the host port
const otp = await $http.post('http://vrooli-opentripplanner:8080/otp/transmodel/v3', {
  body: { query }
});
```

## License

OpenTripPlanner is licensed under LGPL. This resource wrapper is part of the Vrooli ecosystem.