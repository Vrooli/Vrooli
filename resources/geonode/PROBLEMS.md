# GeoNode Resource - Known Problems

## Docker Image Versions
- **Issue**: GeoNode Docker images have specific version tags that may change
- **Solution**: Use `geonode/geonode:4.4.3` for Django and `geonode/geoserver:2.24.4-latest` for GeoServer  
- **Note**: The tag `4-ubuntu-22.04` doesn't exist, use specific version numbers instead

## Startup Time
- **Issue**: Full stack takes 60-120 seconds to become healthy
- **Solution**: Use `--wait` flag with start command and be patient during initial startup

## Memory Requirements
- **Issue**: Full stack requires significant memory (4GB minimum)
- **Solution**: Ensure adequate system resources before starting

## Port Conflicts
- **Issue**: Default ports 8100-8101 may conflict with other services
- **Solution**: Ports are now properly registered in port_registry.sh to avoid conflicts

## Container Startup Issues (RESOLVED)
- **Issue**: GeoNode Django and GeoServer containers restart continuously with default configuration
- **Solution Applied**: 
  - Django: Added proper entrypoint `/usr/src/geonode/entrypoint.sh` with command `python manage.py runserver`
  - GeoServer: Switched to stable Kartoza GeoServer image which includes proper initialization
- **Status**: RESOLVED - All containers now start and run successfully

## Django Long Startup Time
- **Issue**: Django container takes 2-3 minutes to become fully accessible
- **Cause**: Initial database migrations and setup scripts run on first startup
- **Workaround**: Health checks only verify container is running, not full Django availability
- **Impact**: First requests may fail until Django is fully initialized