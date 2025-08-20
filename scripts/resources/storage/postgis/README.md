# PostGIS Resource

PostGIS is a spatial database extension for PostgreSQL that adds geographic objects and functions for location-based queries and geospatial analysis. It transforms PostgreSQL into a complete Geographic Information System (GIS) that enables Vrooli to build powerful location-aware applications.

## Quick Start

```bash
# Check PostGIS status and health
vrooli resource postgis status

# Start PostGIS if not running
vrooli resource postgis start

# Inject spatial SQL queries
vrooli resource postgis inject /path/to/spatial.sql

# Import geographic data
vrooli resource postgis import-shapefile /path/to/data.shp

# View example queries
vrooli resource postgis examples
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
- **Standards Compliant**: OGC Simple Features, SQL/MM Spatial
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