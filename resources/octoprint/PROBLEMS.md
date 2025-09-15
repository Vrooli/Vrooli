# OctoPrint Resource - Known Problems

## API Write Operations Require Additional Permissions
**Problem**: OctoPrint API write operations (settings, connection control) return 403 Forbidden even with valid API key
**Cause**: OctoPrint requires user authentication for certain operations, not just API key
**Status**: API read operations work (version, status), web interface fully functional
**Solution**: Would need to implement OctoPrint user authentication system or use application keys with appropriate permissions

## Permission Issues with Docker Volumes
**Problem**: Files created inside the Docker container are owned by root
**Cause**: OctoPrint Docker container runs as root user by default
**Impact**: Cannot easily clean up or modify configuration files
**Mitigation**: Added --user flag to Docker run command, but may cause issues with some operations
**Solution**: Need to configure container with proper permissions and volume ownership

## Virtual Printer Configuration
**Problem**: Virtual printer requires manual connection through web interface
**Cause**: OctoPrint's connection API requires elevated permissions
**Impact**: Virtual printer not automatically connected, temperature data only simulated
**Mitigation**: Added simulated temperature display when printer not connected
**Solution**: Connect virtual printer via web interface for live data, or implement user authentication

## Integration Test Timing
**Problem**: Integration tests occasionally fail due to service startup timing
**Cause**: OctoPrint takes time to fully initialize after container starts
**Mitigation**: Added sleep delays and --wait flag, but may still have issues
**Solution**: Implement proper health check polling with exponential backoff