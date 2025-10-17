# Open MCT Known Issues and Solutions

## Common Problems

### 1. Docker Build Fails
**Problem**: Docker build fails with npm install errors
**Solution**: 
- Ensure Docker has sufficient memory (at least 2GB)
- Try building with `--no-cache` flag
- Check network connectivity for npm registry access

### 2. Port Already in Use
**Problem**: Port 8099 or 8100 already in use
**Solution**:
```bash
# Check what's using the port
lsof -i :8099
lsof -i :8100

# Stop conflicting service or change Open MCT port in config/defaults.sh
export OPENMCT_PORT=8199  # Alternative port
```

### 3. WebSocket Connection Fails
**Problem**: WebSocket clients cannot connect
**Solution**:
- Ensure both HTTP and WebSocket ports are exposed in Docker
- Check firewall rules allow WebSocket connections
- Verify WebSocket port is correctly configured (default: 8100)

### 4. Database Lock Errors
**Problem**: SQLite database locked errors
**Solution**:
- Enable WAL mode (Write-Ahead Logging) - already set by default
- Reduce concurrent write operations
- Consider upgrading to PostgreSQL for high-load scenarios

### 5. Demo Telemetry Not Appearing
**Problem**: Demo telemetry streams not showing data
**Solution**:
- Verify OPENMCT_ENABLE_DEMO is set to "true"
- Check container logs for demo telemetry initialization
- Restart the container to reinitialize demo streams

### 6. Memory Usage Growing
**Problem**: Container memory usage continuously increasing
**Solution**:
- Configure history retention: `OPENMCT_HISTORY_DAYS=7`
- Enable data compression: `OPENMCT_COMPRESS_HISTORY=true`
- Implement periodic cleanup of old telemetry data

### 7. Dashboard Not Loading
**Problem**: Web interface shows blank page or errors
**Solution**:
- Clear browser cache and cookies
- Check browser console for JavaScript errors
- Verify all required npm packages installed correctly
- Try accessing from different browser

### 8. Authentication Issues
**Problem**: Cannot enable authentication
**Solution**:
- Set OPENMCT_AUTH_ENABLED=true
- Configure OPENMCT_AUTH_PASSWORD environment variable
- Restart container for changes to take effect

## Performance Optimization

### High CPU Usage
- Reduce telemetry sample rate: `OPENMCT_SAMPLE_RATE=5000`
- Limit maximum concurrent streams: `OPENMCT_MAX_STREAMS=50`
- Disable unnecessary demo streams

### Slow Historical Queries
- Add database indexes (already created by default)
- Limit query time ranges
- Enable caching: `OPENMCT_CACHE_ENABLED=true`

## Integration Issues

### MQTT Connection Problems
- Verify MQTT broker is accessible from container
- Check MQTT authentication credentials
- Ensure topic patterns are correctly formatted

### Traccar Integration Fails
- Verify Traccar API is accessible
- Check API authentication tokens
- Ensure device IDs are valid

## Docker-Specific Issues

### Container Restart Loop
**Problem**: Container continuously restarts
**Solution**:
- Check container logs: `docker logs vrooli-openmct`
- Verify all environment variables are set correctly
- Ensure data directory has proper permissions

### Volume Mount Errors
**Problem**: Cannot mount data directories
**Solution**:
- Create directories before starting container
- Check directory permissions
- Use absolute paths for volume mounts

## Debugging Commands

```bash
# Check container status
docker ps -a | grep openmct

# View container logs
docker logs -f vrooli-openmct

# Access container shell
docker exec -it vrooli-openmct sh

# Check database integrity
sqlite3 ~/.vrooli/openmct/data/telemetry.db "PRAGMA integrity_check;"

# Monitor resource usage
docker stats vrooli-openmct

# Test health endpoint
curl -v http://localhost:8099/health

# Check WebSocket connectivity
wscat -c ws://localhost:8100/api/telemetry/live
```

## Recovery Procedures

### Full Reset
```bash
resource-openmct manage stop
resource-openmct manage uninstall
rm -rf ~/.vrooli/openmct
resource-openmct manage install
resource-openmct manage start --wait
```

### Database Recovery
```bash
# Backup current database
cp ~/.vrooli/openmct/data/telemetry.db ~/.vrooli/openmct/data/telemetry.db.backup

# Export data
sqlite3 ~/.vrooli/openmct/data/telemetry.db .dump > backup.sql

# Create new database
rm ~/.vrooli/openmct/data/telemetry.db
resource-openmct manage restart

# Import data if needed
sqlite3 ~/.vrooli/openmct/data/telemetry.db < backup.sql
```

## Future Improvements

1. **PostgreSQL Support**: Add option for PostgreSQL backend for better concurrency
2. **Clustering**: Implement multi-instance deployment for high availability
3. **Plugin Management**: Create plugin marketplace for custom visualizations
4. **Advanced Auth**: Integrate with Keycloak for enterprise authentication
5. **Metrics Export**: Add Prometheus metrics endpoint for monitoring