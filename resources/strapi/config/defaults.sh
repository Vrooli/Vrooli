#!/usr/bin/env bash
# Strapi Resource Default Configuration
# All values can be overridden via environment variables

# Service configuration
export STRAPI_PORT="${STRAPI_PORT:-1337}"
export STRAPI_HOST="${STRAPI_HOST:-0.0.0.0}"
export STRAPI_VERSION="${STRAPI_VERSION:-5.0}"

# Database configuration
export POSTGRES_HOST="${POSTGRES_HOST:-localhost}"
export POSTGRES_PORT="${POSTGRES_PORT:-5433}"
export POSTGRES_USER="${POSTGRES_USER:-postgres}"
export POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-postgres}"
export STRAPI_DATABASE_NAME="${STRAPI_DATABASE_NAME:-strapi}"

# Admin configuration
export STRAPI_ADMIN_EMAIL="${STRAPI_ADMIN_EMAIL:-admin@vrooli.local}"
export STRAPI_ADMIN_PASSWORD="${STRAPI_ADMIN_PASSWORD:-}"

# Storage configuration
export STRAPI_DATA_DIR="${STRAPI_DATA_DIR:-${HOME}/.vrooli/strapi}"
export STRAPI_UPLOAD_DIR="${STRAPI_UPLOAD_DIR:-${STRAPI_DATA_DIR}/uploads}"

# Performance configuration
export STRAPI_MAX_CONNECTIONS="${STRAPI_MAX_CONNECTIONS:-10}"
export STRAPI_CACHE_ENABLED="${STRAPI_CACHE_ENABLED:-true}"
export STRAPI_RATE_LIMIT="${STRAPI_RATE_LIMIT:-100}"

# MinIO integration (optional)
export MINIO_ENDPOINT="${MINIO_ENDPOINT:-localhost:9000}"
export MINIO_ACCESS_KEY="${MINIO_ACCESS_KEY:-}"
export MINIO_SECRET_KEY="${MINIO_SECRET_KEY:-}"
export MINIO_BUCKET="${MINIO_BUCKET:-strapi-uploads}"
export STORAGE_PROVIDER="${STORAGE_PROVIDER:-local}"

# Timeouts
export STRAPI_STARTUP_TIMEOUT="${STRAPI_STARTUP_TIMEOUT:-60}"
export STRAPI_SHUTDOWN_TIMEOUT="${STRAPI_SHUTDOWN_TIMEOUT:-30}"
export STRAPI_HEALTH_CHECK_INTERVAL="${STRAPI_HEALTH_CHECK_INTERVAL:-30}"

# Logging
export STRAPI_LOG_LEVEL="${STRAPI_LOG_LEVEL:-info}"
export STRAPI_LOG_FILE="${STRAPI_LOG_FILE:-${STRAPI_DATA_DIR}/strapi.log}"

# Development mode
export NODE_ENV="${NODE_ENV:-production}"
export STRAPI_DISABLE_TELEMETRY="${STRAPI_DISABLE_TELEMETRY:-true}"