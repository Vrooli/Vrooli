#!/usr/bin/env bash
# Keycloak Resource Default Configuration

# Container configuration
[[ -z "${KEYCLOAK_CONTAINER_NAME:-}" ]] && KEYCLOAK_CONTAINER_NAME="vrooli-keycloak"
[[ -z "${KEYCLOAK_IMAGE:-}" ]] && KEYCLOAK_IMAGE="keycloak/keycloak:latest"
[[ -z "${KEYCLOAK_NETWORK:-}" ]] && KEYCLOAK_NETWORK="vrooli-network"

# Admin configuration (development defaults)
[[ -z "${KEYCLOAK_ADMIN_USER:-}" ]] && KEYCLOAK_ADMIN_USER="admin"
[[ -z "${KEYCLOAK_ADMIN_PASSWORD:-}" ]] && KEYCLOAK_ADMIN_PASSWORD="admin"

# Database configuration
# Use PostgreSQL if available, otherwise fallback to H2 for development
if [[ -z "${KEYCLOAK_DB:-}" ]]; then
    # Check if PostgreSQL is available
    if command -v psql &>/dev/null || docker ps --format "table {{.Names}}" 2>/dev/null | grep -q "vrooli-postgres"; then
        KEYCLOAK_DB="postgres"
    else
        KEYCLOAK_DB="dev-file"
    fi
fi

# PostgreSQL configuration (when using postgres)
[[ -z "${KEYCLOAK_DB_URL:-}" ]] && KEYCLOAK_DB_URL="jdbc:postgresql://vrooli-postgres-main:5432/keycloak"
[[ -z "${KEYCLOAK_DB_USERNAME:-}" ]] && KEYCLOAK_DB_USERNAME="keycloak"
[[ -z "${KEYCLOAK_DB_PASSWORD:-}" ]] && KEYCLOAK_DB_PASSWORD="keycloak"
[[ -z "${KEYCLOAK_DB_SCHEMA:-}" ]] && KEYCLOAK_DB_SCHEMA="public"

# Keycloak configuration
[[ -z "${KEYCLOAK_HOSTNAME_STRICT:-}" ]] && KEYCLOAK_HOSTNAME_STRICT="false"
[[ -z "${KEYCLOAK_HOSTNAME_STRICT_HTTPS:-}" ]] && KEYCLOAK_HOSTNAME_STRICT_HTTPS="false"
[[ -z "${KEYCLOAK_HTTP_ENABLED:-}" ]] && KEYCLOAK_HTTP_ENABLED="true"
[[ -z "${KEYCLOAK_HEALTH_ENABLED:-}" ]] && KEYCLOAK_HEALTH_ENABLED="true"
[[ -z "${KEYCLOAK_METRICS_ENABLED:-}" ]] && KEYCLOAK_METRICS_ENABLED="true"

# Feature flags
[[ -z "${KEYCLOAK_FEATURES:-}" ]] && KEYCLOAK_FEATURES="token-exchange,admin-fine-grained-authz"

# Logging
[[ -z "${KEYCLOAK_LOG_LEVEL:-}" ]] && KEYCLOAK_LOG_LEVEL="INFO"

# Performance tuning
[[ -z "${KEYCLOAK_JVM_OPTS:-}" ]] && KEYCLOAK_JVM_OPTS="-Xms512m -Xmx1024m"