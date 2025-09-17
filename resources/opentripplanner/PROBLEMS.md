# OpenTripPlanner Known Issues and Solutions

## Issues Encountered

### 1. Docker Startup Command Issues (RESOLVED)
**Problem**: OTP container failed to start with "Parameter error" messages.

**Root Cause**: The OpenTripPlanner Docker image uses a custom entrypoint that automatically adds `/var/opentripplanner/` as the base directory. Providing additional directory paths caused conflicts.

**Solution**: 
- Use simple flags like `--load --serve` or `--build --save` without directory paths
- The entrypoint script handles the directory automatically

### 2. Health Check Endpoint Changes (RESOLVED)
**Problem**: Traditional health endpoints like `/otp/routers` return 404.

**Root Cause**: OTP 2.x changed its API structure. The debug UI is now served at root.

**Solution**:
- Use root endpoint `/` for health checks (serves debug UI HTML)
- Check for "OTP Debug" string in response

### 3. Large Download Sizes (MITIGATED)
**Problem**: Full Oregon OSM data is >500MB, slow to download.

**Solution**:
- Use BBBike Portland extract instead (56MB vs 500MB+)
- URL: `https://download.bbbike.org/osm/bbbike/Portland/Portland.osm.pbf`

### 4. Graph Building Time
**Problem**: Graph building can take 1-2 minutes for medium-sized cities.

**Mitigation**:
- Build graph once and reuse
- Graph is persisted in `graph.obj` file
- Server automatically loads existing graph on startup

### 5. API Response Times
**Problem**: Complex routing queries can be slow (>5 seconds).

**Mitigation**:
- Warm up the cache with initial queries
- Use appropriate timeout values in API calls
- Consider limiting search radius for faster responses

## Configuration Notes

### Router Configuration
The router-config.json file controls routing behavior:
```json
{
  "routingDefaults": {
    "walkSpeed": 1.34,
    "numItineraries": 3
  }
}
```

### Memory Requirements
- Minimum: 2GB heap for small cities
- Recommended: 4GB+ for large metropolitan areas
- Configure via `OTP_HEAP_SIZE` environment variable

### Port Configuration
- Default port: 8080
- Configure via `OTP_PORT` environment variable
- Never hardcode ports in scripts

## Testing Tips

### Quick Health Check
```bash
timeout 5 curl -sf http://localhost:8080/ | grep -q "OTP Debug"
```

### Test Graph Building
```bash
vrooli resource opentripplanner content execute --action build-graph
```

### Sample Route Query
```bash
curl "http://localhost:8080/otp/routers/default/plan?fromPlace=45.5152,-122.6784&toPlace=45.5234,-122.6762&mode=TRANSIT,WALK"
```

## Future Improvements

1. **GTFS-RT Support**: Add real-time transit feed ingestion
2. **PostGIS Integration**: Store route analytics in spatial database
3. **Fare Calculation**: Enable fare computation in router config
4. **Isochrone API**: Add time-based accessibility analysis
5. **Bike Share Integration**: Include bike share station data