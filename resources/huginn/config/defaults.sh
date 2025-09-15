#!/usr/bin/env bash
# Huginn Configuration Defaults
# All configuration constants and default values

# Get the directory of this script
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
HUGINN_DIR="${APP_ROOT}/resources/huginn"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"

# Huginn port configuration
if [[ -z "${HUGINN_PORT:-}" ]]; then
    readonly HUGINN_PORT="${HUGINN_CUSTOM_PORT:-$(resources::get_default_port "huginn")}"
fi
if [[ -z "${HUGINN_BASE_URL:-}" ]]; then
    readonly HUGINN_BASE_URL="http://localhost:${HUGINN_PORT}"
fi

# Resource metadata
if [[ -z "${RESOURCE_NAME:-}" ]]; then
    readonly RESOURCE_NAME="huginn"
fi
if [[ -z "${RESOURCE_CATEGORY:-}" ]]; then
    readonly RESOURCE_CATEGORY="automation"
fi
if [[ -z "${RESOURCE_DESC:-}" ]]; then
    readonly RESOURCE_DESC="Agent-based workflow automation and monitoring platform"
fi
if [[ -z "${RESOURCE_PORT:-}" ]]; then
    readonly RESOURCE_PORT="${HUGINN_PORT}"
fi

# Container configuration
if [[ -z "${CONTAINER_NAME:-}" ]]; then
    readonly CONTAINER_NAME="huginn"
fi
if [[ -z "${DB_CONTAINER_NAME:-}" ]]; then
    readonly DB_CONTAINER_NAME="huginn-mysql"
fi
if [[ -z "${VOLUME_NAME:-}" ]]; then
    readonly VOLUME_NAME="huginn-data"
fi
if [[ -z "${DB_VOLUME_NAME:-}" ]]; then
    readonly DB_VOLUME_NAME="huginn-mysql-data"
fi
if [[ -z "${NETWORK_NAME:-}" ]]; then
    readonly NETWORK_NAME="vrooli-network"
fi

# Image configuration
if [[ -z "${HUGINN_IMAGE:-}" ]]; then
    readonly HUGINN_IMAGE="ghcr.io/huginn/huginn-single-process:latest"
fi
if [[ -z "${MYSQL_IMAGE:-}" ]]; then
    readonly MYSQL_IMAGE="mysql:5.7"
fi

# Data directories
if [[ -z "${HUGINN_DATA_DIR:-}" ]]; then
    readonly HUGINN_DATA_DIR="${HOME}/.huginn"
fi
if [[ -z "${HUGINN_DB_DIR:-}" ]]; then
    readonly HUGINN_DB_DIR="${HUGINN_DATA_DIR}/mysql"
fi
if [[ -z "${HUGINN_UPLOADS_DIR:-}" ]]; then
    readonly HUGINN_UPLOADS_DIR="${HUGINN_DATA_DIR}/uploads"
fi

# Default credentials
if [[ -z "${DEFAULT_DB_PASSWORD:-}" ]]; then
    # Use a consistent password, not timestamp-based
    readonly DEFAULT_DB_PASSWORD="huginn_secure_password_2025"
fi
if [[ -z "${DEFAULT_ROOT_PASSWORD:-}" ]]; then
    readonly DEFAULT_ROOT_PASSWORD="root_secure_password_2025"
fi
if [[ -z "${DEFAULT_ADMIN_EMAIL:-}" ]]; then
    readonly DEFAULT_ADMIN_EMAIL="admin@huginn.local"
fi
if [[ -z "${DEFAULT_ADMIN_USERNAME:-}" ]]; then
    readonly DEFAULT_ADMIN_USERNAME="admin"
fi
if [[ -z "${DEFAULT_ADMIN_PASSWORD:-}" ]]; then
    readonly DEFAULT_ADMIN_PASSWORD="vrooli_huginn_secure_2025"
fi

# Health check configuration
if [[ -z "${HUGINN_HEALTH_CHECK_INTERVAL:-}" ]]; then
    readonly HUGINN_HEALTH_CHECK_INTERVAL=5
fi
if [[ -z "${HUGINN_HEALTH_CHECK_MAX_ATTEMPTS:-}" ]]; then
    readonly HUGINN_HEALTH_CHECK_MAX_ATTEMPTS=30
fi
if [[ -z "${HUGINN_HEALTH_CHECK_TIMEOUT:-}" ]]; then
    readonly HUGINN_HEALTH_CHECK_TIMEOUT=10
fi

# Docker health check settings
if [[ -z "${DOCKER_HEALTH_INTERVAL:-}" ]]; then
    readonly DOCKER_HEALTH_INTERVAL="30s"
fi
if [[ -z "${DOCKER_HEALTH_TIMEOUT:-}" ]]; then
    readonly DOCKER_HEALTH_TIMEOUT="10s"
fi
if [[ -z "${DOCKER_HEALTH_RETRIES:-}" ]]; then
    readonly DOCKER_HEALTH_RETRIES=3
fi

# API configuration
if [[ -z "${HUGINN_API_TIMEOUT:-}" ]]; then
    readonly HUGINN_API_TIMEOUT=30
fi
if [[ -z "${RAILS_RUNNER_TIMEOUT:-}" ]]; then
    readonly RAILS_RUNNER_TIMEOUT=60
fi

# Authentication configuration
if [[ -z "${AUTH_CHECK_INTERVAL_MS:-}" ]]; then
    readonly AUTH_CHECK_INTERVAL_MS=30000
fi
if [[ -z "${AUTH_SESSION_TIMEOUT_MS:-}" ]]; then
    readonly AUTH_SESSION_TIMEOUT_MS=1800000  # 30 minutes
fi

# Backup configuration
if [[ -z "${BACKUP_RETENTION_DAYS:-}" ]]; then
    readonly BACKUP_RETENTION_DAYS=30
fi
if [[ -z "${BACKUP_COMPRESSION:-}" ]]; then
    readonly BACKUP_COMPRESSION="gzip"
fi

# Integration configuration
if [[ -z "${INTEGRATION_TIMEOUT:-}" ]]; then
    readonly INTEGRATION_TIMEOUT=30
fi
if [[ -z "${WEBHOOK_TIMEOUT:-}" ]]; then
    readonly WEBHOOK_TIMEOUT=60
fi

# Export configuration function
huginn::export_config() {
    # Export all readonly variables
    export HUGINN_PORT HUGINN_BASE_URL
    export RESOURCE_NAME RESOURCE_CATEGORY RESOURCE_DESC RESOURCE_PORT
    export CONTAINER_NAME DB_CONTAINER_NAME VOLUME_NAME DB_VOLUME_NAME NETWORK_NAME
    export HUGINN_IMAGE MYSQL_IMAGE
    export HUGINN_DATA_DIR HUGINN_DB_DIR HUGINN_UPLOADS_DIR
    export DEFAULT_DB_PASSWORD DEFAULT_ROOT_PASSWORD DEFAULT_ADMIN_EMAIL DEFAULT_ADMIN_USERNAME DEFAULT_ADMIN_PASSWORD
    export HUGINN_HEALTH_CHECK_INTERVAL HUGINN_HEALTH_CHECK_MAX_ATTEMPTS HUGINN_HEALTH_CHECK_TIMEOUT
    export DOCKER_HEALTH_INTERVAL DOCKER_HEALTH_TIMEOUT DOCKER_HEALTH_RETRIES
    export HUGINN_API_TIMEOUT RAILS_RUNNER_TIMEOUT
    export AUTH_CHECK_INTERVAL_MS AUTH_SESSION_TIMEOUT_MS
    export BACKUP_RETENTION_DAYS BACKUP_COMPRESSION
    export INTEGRATION_TIMEOUT WEBHOOK_TIMEOUT
}