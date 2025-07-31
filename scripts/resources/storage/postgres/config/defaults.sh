#!/usr/bin/env bash

# PostgreSQL Resource Configuration Defaults
# This file contains all configuration constants and defaults for the PostgreSQL resource

# Resource metadata
readonly POSTGRES_RESOURCE_NAME="postgres"
readonly POSTGRES_RESOURCE_CATEGORY="storage"
readonly POSTGRES_DISPLAY_NAME="PostgreSQL Database"
readonly POSTGRES_DESCRIPTION="Managed PostgreSQL instances for client isolation"

# Docker configuration
readonly POSTGRES_IMAGE="postgres:16-alpine"
readonly POSTGRES_CONTAINER_PREFIX="vrooli-postgres"
readonly POSTGRES_VOLUME_PREFIX="vrooli-postgres"
readonly POSTGRES_NETWORK="vrooli-network"

# Instance configuration
[[ -z "${POSTGRES_DEFAULT_PORT:-}" ]] && readonly POSTGRES_DEFAULT_PORT=5433
readonly POSTGRES_DEFAULT_USER="vrooli"
readonly POSTGRES_DEFAULT_DB="vrooli_client"
readonly POSTGRES_MAX_INSTANCES=67

# Port range for instances  
[[ -z "${POSTGRES_INSTANCE_PORT_RANGE_START:-}" ]] && readonly POSTGRES_INSTANCE_PORT_RANGE_START=5433
[[ -z "${POSTGRES_INSTANCE_PORT_RANGE_END:-}" ]] && readonly POSTGRES_INSTANCE_PORT_RANGE_END=5499

# Health check configuration
readonly POSTGRES_HEALTH_CHECK_INTERVAL=30
readonly POSTGRES_HEALTH_CHECK_TIMEOUT=5
readonly POSTGRES_HEALTH_CHECK_RETRIES=5

# Backup configuration
readonly POSTGRES_BACKUP_DIR="${HOME}/.vrooli/backups/postgres"
readonly POSTGRES_BACKUP_RETENTION_DAYS=7

# Template directory
readonly POSTGRES_TEMPLATE_DIR="$(dirname "${BASH_SOURCE[0]}")/../templates"

# Instance data directory
readonly POSTGRES_INSTANCES_DIR="$(dirname "${BASH_SOURCE[0]}")/../instances"

# Configuration directory
readonly POSTGRES_CONFIG_DIR="${HOME}/.vrooli/postgres"

# Default PostgreSQL configuration
readonly POSTGRES_DEFAULT_MAX_CONNECTIONS=100
readonly POSTGRES_DEFAULT_SHARED_BUFFERS="128MB"
readonly POSTGRES_DEFAULT_WORK_MEM="4MB"

# Initialization wait time (seconds)
readonly POSTGRES_INITIALIZATION_WAIT=5

# Default locale settings
readonly POSTGRES_DEFAULT_LOCALE="en_US.UTF-8"
readonly POSTGRES_DEFAULT_ENCODING="UTF8"

# GUI (pgweb) configuration
readonly POSTGRES_GUI_IMAGE="sosedoff/pgweb:latest"
readonly POSTGRES_GUI_CONTAINER_PREFIX="vrooli-pgweb"
readonly POSTGRES_GUI_DEFAULT_PORT=8080
readonly POSTGRES_GUI_PORT_RANGE_START=8080
readonly POSTGRES_GUI_PORT_RANGE_END=8099
readonly POSTGRES_GUI_MAX_INSTANCES=20

#######################################
# Export configuration variables
#######################################
postgres::export_config() {
    export POSTGRES_RESOURCE_NAME POSTGRES_RESOURCE_CATEGORY POSTGRES_DISPLAY_NAME POSTGRES_DESCRIPTION
    export POSTGRES_IMAGE POSTGRES_CONTAINER_PREFIX POSTGRES_VOLUME_PREFIX POSTGRES_NETWORK
    export POSTGRES_DEFAULT_PORT POSTGRES_DEFAULT_USER POSTGRES_DEFAULT_DB POSTGRES_MAX_INSTANCES
    export POSTGRES_INSTANCE_PORT_RANGE_START POSTGRES_INSTANCE_PORT_RANGE_END
    export POSTGRES_HEALTH_CHECK_INTERVAL POSTGRES_HEALTH_CHECK_TIMEOUT POSTGRES_HEALTH_CHECK_RETRIES
    export POSTGRES_BACKUP_DIR POSTGRES_BACKUP_RETENTION_DAYS
    export POSTGRES_TEMPLATE_DIR POSTGRES_INSTANCES_DIR POSTGRES_CONFIG_DIR
    export POSTGRES_DEFAULT_MAX_CONNECTIONS POSTGRES_DEFAULT_SHARED_BUFFERS POSTGRES_DEFAULT_WORK_MEM
    export POSTGRES_INITIALIZATION_WAIT
    export POSTGRES_DEFAULT_LOCALE POSTGRES_DEFAULT_ENCODING
    export POSTGRES_GUI_IMAGE POSTGRES_GUI_CONTAINER_PREFIX POSTGRES_GUI_DEFAULT_PORT
    export POSTGRES_GUI_PORT_RANGE_START POSTGRES_GUI_PORT_RANGE_END POSTGRES_GUI_MAX_INSTANCES
}