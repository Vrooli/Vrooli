#!/bin/bash

# GeoNode Default Configuration

# Get dynamic port from registry 
# NOTE: Direct sourcing can cause issues in test context due to port_registry.sh command execution
# Using static ports from registry for now (geonode=8100, geonode-geoserver=8101)
GEONODE_PORT="${GEONODE_PORT:-8100}"
GEONODE_GEOSERVER_PORT="${GEONODE_GEOSERVER_PORT:-8101}"

# Service ports - no fallbacks allowed per Vrooli standards
export GEONODE_PORT="${GEONODE_PORT}"
export GEONODE_GEOSERVER_PORT="${GEONODE_GEOSERVER_PORT}"

# Database configuration
export GEONODE_DB_HOST="${GEONODE_DB_HOST:-geonode-postgres}"
export GEONODE_DB_PORT="${GEONODE_DB_PORT:-5432}"
export GEONODE_DB_NAME="${GEONODE_DB_NAME:-geonode}"
export GEONODE_DB_USER="${GEONODE_DB_USER:-geonode}"
export GEONODE_DB_PASSWORD="${GEONODE_DB_PASSWORD:-geonode_secure_pass}"

# Admin configuration
export GEONODE_ADMIN_USER="${GEONODE_ADMIN_USER:-admin}"
export GEONODE_ADMIN_PASSWORD="${GEONODE_ADMIN_PASSWORD:-admin}"
export GEONODE_ADMIN_EMAIL="${GEONODE_ADMIN_EMAIL:-admin@vrooli.local}"

# Django settings
export GEONODE_SECRET_KEY="${GEONODE_SECRET_KEY:-default_secret_key_change_in_production}"
export GEONODE_DEBUG="${GEONODE_DEBUG:-False}"
export GEONODE_ALLOWED_HOSTS="${GEONODE_ALLOWED_HOSTS:-localhost,geonode-django,*}"

# Feature flags
export GEONODE_MONITORING_ENABLED="${GEONODE_MONITORING_ENABLED:-True}"
export GEONODE_OAUTH_ENABLED="${GEONODE_OAUTH_ENABLED:-True}"
export GEONODE_API_ENABLED="${GEONODE_API_ENABLED:-True}"

# Storage configuration (for P1 integration)
export GEONODE_USE_MINIO="${GEONODE_USE_MINIO:-false}"
export GEONODE_MINIO_ENDPOINT="${GEONODE_MINIO_ENDPOINT:-localhost:9000}"
export GEONODE_MINIO_BUCKET="${GEONODE_MINIO_BUCKET:-geonode-data}"

# Integration flags (for P1/P2 features)
export GEONODE_USE_EXTERNAL_POSTGIS="${GEONODE_USE_EXTERNAL_POSTGIS:-false}"
export GEONODE_EXTERNAL_POSTGIS_HOST="${GEONODE_EXTERNAL_POSTGIS_HOST:-localhost}"
export GEONODE_EXTERNAL_POSTGIS_PORT="${GEONODE_EXTERNAL_POSTGIS_PORT:-5434}"