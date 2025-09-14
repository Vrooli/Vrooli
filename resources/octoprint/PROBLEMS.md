# OctoPrint Resource - Known Problems

## API Authentication Issues
**Problem**: OctoPrint API returns 403 Forbidden even with correct API key
**Cause**: OctoPrint's config.yaml is created inside the Docker container with root ownership, making it difficult to update with the generated API key
**Workaround**: The web interface works and file management operations work through direct file system access
**Solution**: Need to properly configure OctoPrint's config.yaml before starting the container, or use OctoPrint's REST API to configure the API key after startup

## Permission Issues with Docker Volumes
**Problem**: Files created inside the Docker container are owned by root
**Cause**: OctoPrint Docker container runs as root user by default
**Impact**: Cannot easily clean up or modify configuration files
**Solution**: Configure Docker container to run with proper user ID mapping

## Virtual Printer Configuration
**Problem**: Virtual printer mode doesn't fully simulate temperature readings and printer state
**Cause**: OctoPrint's virtual printer plugin requires additional configuration
**Impact**: Temperature monitoring and print control commands return empty responses in virtual mode
**Solution**: Need to configure virtual printer plugin with simulated responses

## Integration Test Timing
**Problem**: Integration tests occasionally fail due to service startup timing
**Cause**: OctoPrint takes time to fully initialize after container starts
**Mitigation**: Added sleep delays and --wait flag, but may still have issues
**Solution**: Implement proper health check polling with exponential backoff