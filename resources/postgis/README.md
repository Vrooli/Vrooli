# PostGIS Resource

PostGIS is a spatial database extension for PostgreSQL that adds geographic objects and functions for location-based queries and geospatial analysis. It transforms PostgreSQL into a complete Geographic Information System (GIS) that enables Vrooli to build powerful location-aware applications.

**Status**: Production Ready - All P0, P1, and P2 features implemented and tested (100% PRD completion)

## Features

- **PostGIS 3.4** with full spatial database capabilities
- **pgRouting 3.8.0** for advanced network analysis and routing
- **GDAL Tools** including ogr2ogr for GIS format import/export
- **Complete spatial analysis** including proximity, watersheds, viewsheds
- **Geocoding services** for address/coordinate conversion
- **Visualization support** for maps and spatial data

## Quick Start

```bash
# Check PostGIS status and health
vrooli resource postgis status

# Start PostGIS if not running
vrooli resource postgis start

# Import spatial data (SQL format - recommended)
vrooli resource postgis content add /path/to/spatial.sql

# Import GIS formats (when ogr2ogr available)
vrooli resource postgis content import-shapefile /path/to/data.shp
vrooli resource postgis content import-geojson /path/to/data.geojson
vrooli resource postgis content import-kml /path/to/data.kml
vrooli resource postgis content import-gis /path/to/data.gpx  # Auto-detects format

# View example queries
vrooli resource postgis examples
```

## Data Import

PostGIS supports importing various GIS formats:

### SQL Import (Recommended)
The most reliable method for importing spatial data:
```bash
# Import SQL file with PostGIS geometries
vrooli resource postgis content add /path/to/spatial.sql
```

### GIS Format Import
When ogr2ogr is available, these formats are supported:
- **Shapefile** (.shp) - Common GIS vector format
- **GeoJSON** (.geojson, .json) - Web-friendly format
- **KML/KMZ** (.kml, .kmz) - Google Earth format
- **GPX** (.gpx) - GPS track format
- **CSV** - With longitude/latitude columns

```bash
# Auto-detect format and import
vrooli resource postgis content import-gis /path/to/data.geojson

# Format-specific commands
vrooli resource postgis content import-shapefile cities.shp
vrooli resource postgis content import-geojson boundaries.geojson
vrooli resource postgis content import-kml locations.kml
```

## Advanced Features (P2 Capabilities)

### Map Visualization
Generate various map visualizations from spatial data:

```bash
# Generate GeoJSON from query
vrooli resource postgis visualization geojson "SELECT name, geom FROM cities"

# Create heat map from point data
vrooli resource postgis visualization heatmap crime_data location

# Generate choropleth (colored regions) map
vrooli resource postgis visualization choropleth counties population

# Generate interactive HTML map viewer
vrooli resource postgis visualization viewer /tmp/map.geojson
```

### Geocoding Services
Convert between addresses and coordinates:

```bash
# Initialize geocoding tables
vrooli resource postgis geocoding init

# Convert address to coordinates
vrooli resource postgis geocoding geocode "1600 Pennsylvania Ave, Washington DC"

# Convert coordinates to address
vrooli resource postgis geocoding reverse 38.8977 -77.0365

# Batch geocode addresses from file
vrooli resource postgis geocoding batch addresses.txt results.csv
```

### Advanced Spatial Analysis
Perform complex geographic calculations:

```bash
# Find shortest path between points
vrooli resource postgis spatial shortest-path 40.7128 -74.0060 40.7614 -73.9776

# Find nearby features within radius
vrooli resource postgis spatial proximity 40.7128 -74.0060 1000

# Calculate service area (isochrone)
vrooli resource postgis spatial service-area 40.7128 -74.0060 15 50

# Perform spatial clustering
vrooli resource postgis spatial cluster locations geom 5 0.01

# Calculate spatial statistics
vrooli resource postgis spatial statistics counties geom population
```

## What PostGIS Enables

PostGIS unlocks powerful geographic capabilities for Vrooli scenarios:

- **Location-Based Services**: Find nearby resources, calculate optimal routes
- **Geospatial Analytics**: Analyze spatial patterns, create heat maps
- **Asset Tracking**: Monitor deliveries, optimize fleet routes
- **Real Estate Analysis**: Property valuations based on location factors
- **Environmental Monitoring**: Track sensor data across geographic regions
- **Emergency Planning**: Optimize service locations based on population
- **Geofencing**: Trigger automations when objects enter/exit zones

## Architecture

PostGIS runs as a standalone containerized PostgreSQL instance:

- **Container**: `postgis-main` (PostgreSQL 16 + PostGIS 3.4)
- **Port**: 5434
- **Database**: `spatial`
- **Credentials**: `vrooli/vrooli`
- **Extensions**: PostGIS, PostGIS Raster, PostGIS Topology

## Key Features

- **300+ Spatial Functions**: Distance, area, intersection, buffer, and more
- **Spatial Indexing**: Lightning-fast geographic queries using R-tree indexes
- **Advanced Routing**: pgRouting 3.8.0 for shortest path, Dijkstra, A* algorithms
- **Standards Compliant**: OGC Simple Features, SQL/MM Spatial
- **Map Visualization**: Generate GeoJSON, heat maps, choropleth maps, and HTML viewers
- **Geocoding Services**: Convert addresses to coordinates and vice versa
- **Advanced Spatial Analysis**: Network analysis, proximity, service areas, watersheds, viewsheds
- **Format Support**: GeoJSON, KML, Shapefile, GPX, WKT/WKB
- **Coordinate Systems**: Support for 4000+ spatial reference systems
- **Integration Ready**: Works with QGIS, Leaflet, Mapbox, GeoPandas

## Documentation

- [Overview](docs/overview.md) - Architecture, concepts, and integration patterns
- [Usage Guide](docs/usage.md) - Commands, queries, and best practices
- [Scenario Examples](docs/scenarios.md) - Real-world use cases and implementations

## Integration with Vrooli Resources

PostGIS enhances other resources:
- **n8n/Huginn**: Trigger workflows based on geographic events
- **Ollama**: Generate location descriptions from coordinates
- **MinIO**: Store geographic files (KML, GeoJSON, shapefiles)
- **QuestDB**: Time-series tracking of moving objects
- **Browserless**: Generate map visualizations

## Status

Run `vrooli resource postgis status --format json` for detailed health information including:
- Container status
- PostGIS version
- Available extensions
- Connection details
- Spatial table count