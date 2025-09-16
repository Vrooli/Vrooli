# OpenEMS Known Issues and Solutions

## Container Issues

### Issue: Official OpenEMS Docker images may not be available
**Solution**: The implementation includes a fallback mode using openjdk:17-slim that provides basic simulation capabilities when the official images are unavailable.

### Issue: Port conflicts with other services
**Solution**: All ports are configured via port_registry.sh. If conflicts occur:
- Check `scripts/resources/port_registry.sh` for port assignments
- Ensure no other services are using ports 8084, 8085, 8086, or 502

## Dependency Issues

### Issue: QuestDB or Redis not available
**Solution**: The resource gracefully handles missing dependencies:
- Telemetry data falls back to local file storage (telemetry.jsonl)
- Real-time state uses in-memory cache instead of Redis
- Tests won't fail if optional dependencies are missing

## API Limitations

### Issue: Full OpenEMS API requires complete installation
**Solution**: In simulation mode, basic health and status endpoints are available. For full API functionality:
- Use official OpenEMS images when available
- Deploy with actual hardware connections for complete Modbus support

## Testing

### Issue: Integration tests may timeout on first run
**Solution**: Docker image pulls can take time. Run `vrooli resource openems manage install` first to pre-download images.

### Issue: Modbus tests may fail without proper permissions
**Solution**: Port 502 (standard Modbus port) may require elevated permissions. The resource will continue to function without Modbus if permissions are insufficient.

## Performance

### Issue: High telemetry rates can impact performance
**Solution**: Adjust these environment variables:
- `OPENEMS_TELEMETRY_INTERVAL`: Increase for less frequent updates (default: 1000ms)
- `OPENEMS_TELEMETRY_BATCH_SIZE`: Reduce for smaller batches (default: 100)

## P1 Integration Considerations

### n8n Integration
**Issue**: n8n must be running to deploy workflows
**Solution**: Workflow templates are created locally in /tmp/ and can be imported manually through n8n UI when service is available

### Apache Superset Integration  
**Issue**: QuestDB connection must be configured manually
**Solution**: Use PostgreSQL wire protocol at localhost:8812 with credentials admin:quest

### Eclipse Ditto Integration
**Issue**: Ditto requires manual policy creation for things
**Solution**: Use default policy "openems:energy-policy" or create custom policies through Ditto UI

### Forecast Models
**Issue**: Python 3 required for forecast models
**Solution**: Models are written in Python 3.6+ compatible code. Install python3 if not available

## Workarounds

### Running without Docker
If Docker is unavailable, you can still use the DER simulator directly:
```bash
# Run simulation scripts directly
./lib/der_simulator.sh solar 5000 30
./lib/der_simulator.sh microgrid 60
```

### Manual Health Checks
If automated health checks fail:
```bash
# Check container status
docker ps | grep openems

# Check logs
docker logs openems-edge

# Test API manually
curl -sf http://localhost:8084/health
```