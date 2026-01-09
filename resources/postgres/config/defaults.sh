#!/usr/bin/env bash

# PostgreSQL Resource Configuration Defaults
# This file contains all configuration constants and defaults for the PostgreSQL resource

# Detect project root for proper configuration paths
_postgres_defaults_detect_project_root() {
    local current_dir
    APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
    current_dir="$APP_ROOT/resources/postgres/config"
    
    # Walk up directory tree looking for .vrooli directory
    while [[ "$current_dir" != "/" ]]; do
        if [[ -d "$current_dir/.vrooli" ]]; then
            echo "$current_dir"
            return 0
        fi
        current_dir="${current_dir%/*}"
    done
    
    # Fallback: use standard scripts directory structure
    local script_dir="$APP_ROOT/resources/postgres/config"
    local postgres_dir="${script_dir%/*}"
    local scripts_dir="${postgres_dir%/*/*}"
    local project_root="${scripts_dir%/*}"
    echo "$project_root"
}

# Set project root for proper config paths
POSTGRES_PROJECT_ROOT="$(_postgres_defaults_detect_project_root)"

# Resource metadata - use guards to prevent "readonly variable" errors when sourced multiple times
[[ -z "${POSTGRES_RESOURCE_NAME:-}" ]] && readonly POSTGRES_RESOURCE_NAME="postgres"
[[ -z "${POSTGRES_RESOURCE_CATEGORY:-}" ]] && readonly POSTGRES_RESOURCE_CATEGORY="storage"
[[ -z "${POSTGRES_DISPLAY_NAME:-}" ]] && readonly POSTGRES_DISPLAY_NAME="PostgreSQL Database"
[[ -z "${POSTGRES_DESCRIPTION:-}" ]] && readonly POSTGRES_DESCRIPTION="Managed PostgreSQL instances for client isolation"

# Docker configuration
[[ -z "${POSTGRES_IMAGE:-}" ]] && readonly POSTGRES_IMAGE="postgres:16-alpine"
[[ -z "${POSTGRES_CONTAINER_PREFIX:-}" ]] && readonly POSTGRES_CONTAINER_PREFIX="vrooli-postgres"
[[ -z "${POSTGRES_VOLUME_PREFIX:-}" ]] && readonly POSTGRES_VOLUME_PREFIX="vrooli-postgres"
[[ -z "${POSTGRES_NETWORK:-}" ]] && readonly POSTGRES_NETWORK="vrooli-network"

# Instance configuration
[[ -z "${POSTGRES_DEFAULT_PORT:-}" ]] && readonly POSTGRES_DEFAULT_PORT=5433
[[ -z "${POSTGRES_DEFAULT_USER:-}" ]] && readonly POSTGRES_DEFAULT_USER="vrooli"
[[ -z "${POSTGRES_DEFAULT_DB:-}" ]] && readonly POSTGRES_DEFAULT_DB="vrooli"
[[ -z "${POSTGRES_MAX_INSTANCES:-}" ]] && readonly POSTGRES_MAX_INSTANCES=67

# Port range for instances
[[ -z "${POSTGRES_INSTANCE_PORT_RANGE_START:-}" ]] && readonly POSTGRES_INSTANCE_PORT_RANGE_START=5433
[[ -z "${POSTGRES_INSTANCE_PORT_RANGE_END:-}" ]] && readonly POSTGRES_INSTANCE_PORT_RANGE_END=5499

# Health check configuration
[[ -z "${POSTGRES_HEALTH_CHECK_INTERVAL:-}" ]] && readonly POSTGRES_HEALTH_CHECK_INTERVAL=30
[[ -z "${POSTGRES_HEALTH_CHECK_TIMEOUT:-}" ]] && readonly POSTGRES_HEALTH_CHECK_TIMEOUT=5
[[ -z "${POSTGRES_HEALTH_CHECK_RETRIES:-}" ]] && readonly POSTGRES_HEALTH_CHECK_RETRIES=5

# Backup configuration (using project root)
[[ -z "${POSTGRES_BACKUP_DIR:-}" ]] && readonly POSTGRES_BACKUP_DIR="${POSTGRES_PROJECT_ROOT}/.vrooli/backups/postgres"
[[ -z "${POSTGRES_BACKUP_RETENTION_DAYS:-}" ]] && readonly POSTGRES_BACKUP_RETENTION_DAYS=7

# Template directory
[[ -z "${POSTGRES_TEMPLATE_DIR:-}" ]] && readonly POSTGRES_TEMPLATE_DIR="${APP_ROOT}/resources/postgres/templates"

# Instance data directory
[[ -z "${POSTGRES_INSTANCES_DIR:-}" ]] && readonly POSTGRES_INSTANCES_DIR="${APP_ROOT}/resources/postgres/instances"

# Configuration directory (using project root)
[[ -z "${POSTGRES_CONFIG_DIR:-}" ]] && readonly POSTGRES_CONFIG_DIR="${POSTGRES_PROJECT_ROOT}/.vrooli/postgres"

# Default PostgreSQL configuration
[[ -z "${POSTGRES_DEFAULT_MAX_CONNECTIONS:-}" ]] && readonly POSTGRES_DEFAULT_MAX_CONNECTIONS=100
[[ -z "${POSTGRES_DEFAULT_SHARED_BUFFERS:-}" ]] && readonly POSTGRES_DEFAULT_SHARED_BUFFERS="128MB"
[[ -z "${POSTGRES_DEFAULT_WORK_MEM:-}" ]] && readonly POSTGRES_DEFAULT_WORK_MEM="4MB"

# Initialization wait time (seconds)
[[ -z "${POSTGRES_INITIALIZATION_WAIT:-}" ]] && readonly POSTGRES_INITIALIZATION_WAIT=5

# Default locale settings
[[ -z "${POSTGRES_DEFAULT_LOCALE:-}" ]] && readonly POSTGRES_DEFAULT_LOCALE="en_US.UTF-8"
[[ -z "${POSTGRES_DEFAULT_ENCODING:-}" ]] && readonly POSTGRES_DEFAULT_ENCODING="UTF8"

# GUI (pgweb) configuration
[[ -z "${POSTGRES_GUI_IMAGE:-}" ]] && readonly POSTGRES_GUI_IMAGE="sosedoff/pgweb:latest"
[[ -z "${POSTGRES_GUI_CONTAINER_PREFIX:-}" ]] && readonly POSTGRES_GUI_CONTAINER_PREFIX="vrooli-pgweb"
[[ -z "${POSTGRES_GUI_DEFAULT_PORT:-}" ]] && readonly POSTGRES_GUI_DEFAULT_PORT=8080
[[ -z "${POSTGRES_GUI_PORT_RANGE_START:-}" ]] && readonly POSTGRES_GUI_PORT_RANGE_START=8080
[[ -z "${POSTGRES_GUI_PORT_RANGE_END:-}" ]] && readonly POSTGRES_GUI_PORT_RANGE_END=8099
[[ -z "${POSTGRES_GUI_MAX_INSTANCES:-}" ]] && readonly POSTGRES_GUI_MAX_INSTANCES=20

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

# Call the export function to make variables available
postgres::export_config
