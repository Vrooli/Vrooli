#!/usr/bin/env bash
# Huginn Configuration Defaults
# All configuration constants and default values

# Source common functions for port registry access
HUGINN_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck disable=SC1091
source "${HUGINN_DIR}/../../common.sh"

# Huginn port configuration
readonly HUGINN_PORT="${HUGINN_CUSTOM_PORT:-$(resources::get_default_port "huginn")}"
readonly HUGINN_BASE_URL="http://localhost:${HUGINN_PORT}"

# Resource metadata
readonly RESOURCE_NAME="huginn"
readonly RESOURCE_CATEGORY="automation"
readonly RESOURCE_DESC="Agent-based workflow automation and monitoring platform"
readonly RESOURCE_PORT="${HUGINN_PORT}"

# Container configuration
readonly CONTAINER_NAME="huginn"
readonly DB_CONTAINER_NAME="huginn-postgres"
readonly VOLUME_NAME="huginn-data"
readonly DB_VOLUME_NAME="huginn-postgres-data"
readonly NETWORK_NAME="vrooli-network"

# Image configuration
readonly HUGINN_IMAGE="huginn/huginn:latest"
readonly POSTGRES_IMAGE="postgres:15-alpine"

# Data directories
readonly HUGINN_DATA_DIR="${HOME}/.huginn"
readonly HUGINN_DB_DIR="${HUGINN_DATA_DIR}/postgres"
readonly HUGINN_UPLOADS_DIR="${HUGINN_DATA_DIR}/uploads"

# Default credentials
readonly DEFAULT_DB_PASSWORD="huginn_secure_password_$(date +%s)"
readonly DEFAULT_ADMIN_EMAIL="admin@huginn.local"
readonly DEFAULT_ADMIN_USERNAME="admin"
readonly DEFAULT_ADMIN_PASSWORD="vrooli_huginn_secure_2025"

# Health check configuration
readonly HUGINN_HEALTH_CHECK_INTERVAL=5
readonly HUGINN_HEALTH_CHECK_MAX_ATTEMPTS=30
readonly HUGINN_HEALTH_CHECK_TIMEOUT=10

# Docker health check settings
readonly DOCKER_HEALTH_INTERVAL="30s"
readonly DOCKER_HEALTH_TIMEOUT="10s"
readonly DOCKER_HEALTH_RETRIES=3

# API configuration
readonly HUGINN_API_TIMEOUT=30
readonly RAILS_RUNNER_TIMEOUT=60

# Authentication configuration
readonly AUTH_CHECK_INTERVAL_MS=30000
readonly AUTH_SESSION_TIMEOUT_MS=1800000  # 30 minutes

# Backup configuration
readonly BACKUP_RETENTION_DAYS=30
readonly BACKUP_COMPRESSION="gzip"

# Integration configuration
readonly INTEGRATION_TIMEOUT=30
readonly WEBHOOK_TIMEOUT=60

# Export configuration function
huginn::export_config() {
    # Export all readonly variables
    export HUGINN_PORT HUGINN_BASE_URL
    export RESOURCE_NAME RESOURCE_CATEGORY RESOURCE_DESC RESOURCE_PORT
    export CONTAINER_NAME DB_CONTAINER_NAME VOLUME_NAME DB_VOLUME_NAME NETWORK_NAME
    export HUGINN_IMAGE POSTGRES_IMAGE
    export HUGINN_DATA_DIR HUGINN_DB_DIR HUGINN_UPLOADS_DIR
    export DEFAULT_DB_PASSWORD DEFAULT_ADMIN_EMAIL DEFAULT_ADMIN_USERNAME DEFAULT_ADMIN_PASSWORD
    export HUGINN_HEALTH_CHECK_INTERVAL HUGINN_HEALTH_CHECK_MAX_ATTEMPTS HUGINN_HEALTH_CHECK_TIMEOUT
    export DOCKER_HEALTH_INTERVAL DOCKER_HEALTH_TIMEOUT DOCKER_HEALTH_RETRIES
    export HUGINN_API_TIMEOUT RAILS_RUNNER_TIMEOUT
    export AUTH_CHECK_INTERVAL_MS AUTH_SESSION_TIMEOUT_MS
    export BACKUP_RETENTION_DAYS BACKUP_COMPRESSION
    export INTEGRATION_TIMEOUT WEBHOOK_TIMEOUT
}