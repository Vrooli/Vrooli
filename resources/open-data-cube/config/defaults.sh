#!/bin/bash

# Open Data Cube Default Configuration
# Defines default values for ODC resource

# Resource identification
export ODC_RESOURCE_NAME="open-data-cube"
export ODC_VERSION="latest"

# Port configuration 
# These will be set by the lifecycle system from port_registry.sh
# Initialize as empty if not set to avoid unbound variable errors
export ODC_PORT="${ODC_PORT:-}"
export ODC_DB_PORT="${ODC_DB_PORT:-}"
export DATACUBE_OWS_PORT="${DATACUBE_OWS_PORT:-}"

# Database configuration
export ODC_DB_NAME="datacube"
export ODC_DB_USER="datacube"
export ODC_DB_PASSWORD="datacube_password"

# Container settings
export ODC_CONTAINER_PREFIX="open-data-cube"
export ODC_NETWORK_NAME="odc-network"

# Data paths
export ODC_DATA_PATH="${RESOURCE_DIR}/data"
export ODC_PRODUCTS_PATH="${ODC_DATA_PATH}/products"
export ODC_INDEXED_PATH="${ODC_DATA_PATH}/indexed"
export ODC_CACHE_PATH="${ODC_DATA_PATH}/cache"

# Performance settings
export ODC_MAX_WORKERS="4"
export ODC_CHUNK_SIZE="1024"
export ODC_CACHE_SIZE="1GB"

# Timeout settings
export ODC_STARTUP_TIMEOUT="60"
export ODC_HEALTH_CHECK_TIMEOUT="5"
export ODC_SHUTDOWN_TIMEOUT="30"

# Sample data settings
export ODC_SAMPLE_DATA_ENABLED="true"
export ODC_SAMPLE_SENTINEL2="true"
export ODC_SAMPLE_LANDSAT8="true"

# WMS/WCS settings
export DATACUBE_OWS_ENABLED="true"
export DATACUBE_OWS_MAX_DATASETS="1000"

# Integration settings
export ODC_MINIO_INTEGRATION="${ODC_MINIO_INTEGRATION:-false}"
export ODC_QDRANT_INTEGRATION="${ODC_QDRANT_INTEGRATION:-false}"
export ODC_N8N_INTEGRATION="${ODC_N8N_INTEGRATION:-false}"

# Logging
export ODC_LOG_LEVEL="${ODC_LOG_LEVEL:-INFO}"
export ODC_LOG_FILE="${RESOURCE_DIR}/logs/odc.log"