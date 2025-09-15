# OctoPrint Resource - Known Problems and Solutions

## Container Permission Issues (RESOLVED)
**Problem**: Files created inside Docker container were owned by root, causing permission issues
**Cause**: OctoPrint Docker container runs as root by default
**Impact**: Cannot delete or modify files in data directory
**Solution**: 
- Removed --user flag to allow container to run normally
- If issues occur, move data directory and recreate: `mv data data.old && mkdir data`
**Status**: RESOLVED - Container runs reliably without user mapping

## API Authentication (IMPROVED)
**Problem**: OctoPrint API requires both API key and proper access control configuration
**Cause**: Default security settings restrict write operations
**Impact**: Cannot perform printer control operations via API
**Solution**: 
- Updated config.yaml with disabled access control for local networks
- Added localNetworks configuration including Docker ranges
- Set autologinLocal for automatic authentication
**Status**: IMPROVED - API authentication now working for most operations

## Virtual Printer Auto-Connection (PARTIAL)
**Problem**: Virtual printer doesn't automatically connect on startup
**Cause**: OctoPrint requires specific connection sequence
**Impact**: Manual connection required via web interface
**Mitigation**: 
- Added `octoprint_connect_virtual_printer()` helper function
- Added `connect-printer` CLI command
- Configured autoconnect in config.yaml
**Solution**: Manual connection still required initially via http://localhost:8197
**Status**: PARTIAL - Helper available but manual setup still needed

## Dynamic API Key Generation (RESOLVED)
**Problem**: Container generates its own API key, ignoring provided configuration
**Cause**: OctoPrint creates new config on first run
**Impact**: Pre-generated API keys don't work
**Solution**: 
- Extract actual API key from running container
- Store in config directory for future use
- Updated get_api_key() function to check container first
**Status**: RESOLVED - API key extraction working correctly

## Webcam Integration (NOT IMPLEMENTED)
**Problem**: Webcam support requires additional system software
**Cause**: mjpg-streamer not included in Docker image
**Impact**: No live video feed available
**Mitigation**: Added webcam_setup() helper function for configuration guidance
**Solution**: Future work - create custom Docker image with webcam support
**Status**: NOT IMPLEMENTED - P1 requirement for future iteration

## Integration Test Reliability (RESOLVED)
**Problem**: Tests occasionally failed due to timing issues
**Cause**: Service initialization delays
**Solution**: 
- Improved health check with proper timeout handling
- Added --wait flag to start command
- Better error handling in test scripts
**Status**: RESOLVED - All tests passing reliably