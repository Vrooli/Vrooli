# GeoNode Resource

GeoNode is a comprehensive geospatial content management system that brings enterprise-grade mapping and spatial data capabilities to Vrooli. It combines Django web portal, GeoServer map services, and PostGIS spatial database to create a complete geographic information system.

## Features

- **Complete GIS Stack**: Django portal + GeoServer + PostGIS in containerized deployment
- **Spatial Data Management**: Upload, catalog, and share geospatial datasets
- **Map Services**: WMS, WFS, WCS, and CSW OGC-compliant services
- **Web Portal**: User-friendly interface for browsing and managing spatial data
- **API Access**: RESTful API for programmatic data management
- **Multi-Format Support**: Shapefile, GeoTIFF, KML, GeoJSON, and more

## Quick Start

```bash
# Check GeoNode status
vrooli resource geonode status

# Start GeoNode stack
vrooli resource geonode manage start

# Upload a shapefile
vrooli resource geonode content add-layer /path/to/data.shp

# List available layers
vrooli resource geonode content list-layers

# View GeoNode portal
# Navigate to http://localhost:${GEONODE_PORT} (port shown in status)
```

## Architecture

GeoNode runs as a multi-container stack:

- **geonode-django**: Web portal and REST API (port varies)
- **geonode-geoserver**: Map tile server and OGC services (port + 1)
- **geonode-postgres**: PostGIS spatial database (internal)
- **geonode-redis**: Caching layer (internal)

## CLI Commands

### Lifecycle Management
```bash
vrooli resource geonode manage install   # Install dependencies
vrooli resource geonode manage start     # Start GeoNode stack
vrooli resource geonode manage stop      # Stop GeoNode stack
vrooli resource geonode manage restart   # Restart all services
```

### Content Management
```bash
# Layer management
vrooli resource geonode content add-layer <file>      # Upload spatial data
vrooli resource geonode content list-layers           # List all layers
vrooli resource geonode content get-layer <name>      # Get layer details
vrooli resource geonode content remove-layer <name>   # Delete layer

# Map management
vrooli resource geonode content create-map            # Create new map
vrooli resource geonode content list-maps             # List all maps
vrooli resource geonode content export-map <name>     # Export map configuration
```

### Data Operations
```bash
# Import various formats
vrooli resource geonode import shapefile <file>       # Import shapefile
vrooli resource geonode import geotiff <file>         # Import raster
vrooli resource geonode import geojson <file>         # Import GeoJSON

# Export operations
vrooli resource geonode export layer <name> <format>  # Export layer
vrooli resource geonode export map <name>             # Export map config

# Metadata operations
vrooli resource geonode metadata update <layer>       # Update metadata
vrooli resource geonode metadata search <query>       # Search catalog
```

## API Access

GeoNode provides comprehensive REST API:

```bash
# Get API token
vrooli resource geonode credentials api

# Example API calls
curl -H "Authorization: Bearer <token>" \
  http://localhost:${GEONODE_PORT}/api/layers/

# Upload via API
curl -X POST -H "Authorization: Bearer <token>" \
  -F "file=@data.shp" \
  http://localhost:${GEONODE_PORT}/api/upload/
```

## Integration Examples

### With PostGIS Resource
```bash
# Use existing PostGIS for spatial database
vrooli resource geonode config set-postgres \
  --host localhost --port 5434 --database spatial
```

### With MinIO Resource
```bash
# Configure MinIO for raster storage
vrooli resource geonode config set-storage \
  --type s3 --endpoint localhost:9000 --bucket geonode-data
```

### With Node-RED Resource
```bash
# Set up automated data ingestion
vrooli resource geonode webhook create \
  --name "iot-updates" --url "http://localhost:1880/geonode"
```

## What GeoNode Enables

GeoNode unlocks powerful geospatial capabilities:

- **Digital Twins**: Create spatial representations of physical infrastructure
- **Climate Analysis**: Overlay weather and environmental data on maps
- **Urban Planning**: Analyze land use, zoning, and development patterns
- **Supply Chain**: Track logistics and optimize routing on maps
- **Emergency Response**: Coordinate resources based on geographic data
- **Asset Management**: Map and monitor distributed infrastructure
- **Environmental Monitoring**: Track changes in land cover and resources

## Web Portal Features

Access the GeoNode portal at `http://localhost:${GEONODE_PORT}`:

- **Layer Catalog**: Browse and search available spatial datasets
- **Map Composer**: Create interactive maps with multiple layers
- **Metadata Editor**: Document datasets with rich metadata
- **User Management**: Control access to layers and maps
- **Style Editor**: Customize layer appearance and symbols
- **Download Center**: Export data in various formats

## Troubleshooting

### Stack Won't Start
```bash
# Check container status
vrooli resource geonode status --verbose

# View logs
vrooli resource geonode logs

# Reset and restart
vrooli resource geonode manage stop
vrooli resource geonode manage start --force
```

### Upload Failures
```bash
# Check file format
vrooli resource geonode validate <file>

# View detailed upload logs
vrooli resource geonode logs geoserver

# Try alternative import method
vrooli resource geonode import --method ogr2ogr <file>
```

### Performance Issues
```bash
# Check resource usage
vrooli resource geonode stats

# Optimize tile cache
vrooli resource geonode optimize cache

# Rebuild spatial indexes
vrooli resource geonode optimize indexes
```

## Best Practices

1. **Data Organization**: Use consistent naming and folder structure
2. **Metadata**: Always document layers with complete metadata
3. **Projections**: Standardize on Web Mercator (EPSG:3857) or WGS84
4. **Caching**: Enable tile caching for frequently accessed layers
5. **Security**: Use API tokens for programmatic access
6. **Backups**: Regular export of important layers and maps

## Resources

- [Official GeoNode Documentation](https://docs.geonode.org)
- [GeoServer Documentation](https://docs.geoserver.org)
- [OGC Standards](https://www.ogc.org/standards)
- [PostGIS Spatial SQL](https://postgis.net/docs/)