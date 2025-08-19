# PostGIS Resource

PostGIS is a spatial database extension for PostgreSQL that adds geographic objects and functions for location-based queries and geospatial analysis.

## Overview

PostGIS extends PostgreSQL with:
- Geographic data types (points, lines, polygons, etc.)
- Spatial indexes for fast queries
- Geographic functions (distance, area, intersection, etc.)
- Support for coordinate systems and projections
- Raster data support
- 3D and topology capabilities

## Features

- **Spatial Queries**: Radius searches, nearest neighbor, containment
- **Geographic Operations**: Union, intersection, buffer, simplification
- **Coordinate Systems**: Support for thousands of spatial reference systems
- **Integration**: Works seamlessly with existing PostgreSQL data
- **Standards**: Implements OGC Simple Features specification

## Usage

```bash
# Check status
vrooli resource postgis status

# Enable PostGIS in a database
resource-postgis enable-database mydb

# Import spatial data
resource-postgis import-shapefile /path/to/data.shp

# Run spatial query examples
resource-postgis examples

# Inject SQL with spatial functions
resource-postgis inject /path/to/spatial.sql
```

## Architecture

PostGIS operates as a PostgreSQL extension, requiring:
- An existing PostgreSQL instance (provided by postgres resource)
- CREATE EXTENSION privilege in target databases
- Spatial reference system data (automatically installed)

See [docs/](docs/) for detailed documentation.