#!/bin/bash
# PostGIS Configuration Defaults

# Resource metadata
export POSTGIS_RESOURCE_NAME="postgis"
export POSTGIS_RESOURCE_CATEGORY="storage"
export POSTGIS_RESOURCE_DESCRIPTION="Spatial database extension for PostgreSQL"
export POSTGIS_RESOURCE_VERSION="3.4"

# PostgreSQL connection (inherits from postgres resource)
export POSTGIS_PG_HOST="${POSTGRES_HOST:-localhost}"
export POSTGIS_PG_PORT="${POSTGIS_PORT:-5434}"
export POSTGIS_PG_USER="${POSTGRES_USER:-vrooli}"
export POSTGIS_PG_PASSWORD="${POSTGRES_PASSWORD:-vrooli}"
export POSTGIS_PG_DATABASE="${POSTGRES_DATABASE:-vrooli}"

# PostGIS specific settings
export POSTGIS_DEFAULT_SRID="4326"  # WGS84 coordinate system
export POSTGIS_ENABLE_TOPOLOGY="${POSTGIS_ENABLE_TOPOLOGY:-false}"
export POSTGIS_ENABLE_RASTER="${POSTGIS_ENABLE_RASTER:-true}"
export POSTGIS_ENABLE_TIGER="${POSTGIS_ENABLE_TIGER:-false}"

# Data directories
export POSTGIS_DATA_DIR="${HOME}/.vrooli/postgis"
export POSTGIS_IMPORT_DIR="${POSTGIS_DATA_DIR}/import"
export POSTGIS_EXPORT_DIR="${POSTGIS_DATA_DIR}/export"
export POSTGIS_SQL_DIR="${POSTGIS_DATA_DIR}/sql"

# Features to enable
export POSTGIS_EXTENSIONS="postgis postgis_raster"
if [ "$POSTGIS_ENABLE_TOPOLOGY" = "true" ]; then
    POSTGIS_EXTENSIONS="$POSTGIS_EXTENSIONS postgis_topology"
fi
if [ "$POSTGIS_ENABLE_TIGER" = "true" ]; then
    POSTGIS_EXTENSIONS="$POSTGIS_EXTENSIONS postgis_tiger_geocoder"
fi