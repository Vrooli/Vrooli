#!/usr/bin/env bash
# OpenEMR Default Configuration
# These values can be overridden via environment variables

# Service Ports (allocated from port registry)
export OPENEMR_PORT="${OPENEMR_PORT:-8010}"
export OPENEMR_PORTAL_PORT="${OPENEMR_PORTAL_PORT:-443}"  # Internal HTTPS port
export OPENEMR_API_PORT="${OPENEMR_API_PORT:-8011}"
export OPENEMR_FHIR_PORT="${OPENEMR_FHIR_PORT:-8012}"
export OPENEMR_DB_PORT="${OPENEMR_DB_PORT:-3316}"  # Avoid conflict with system MySQL

# Database Configuration
export OPENEMR_DB_HOST="${OPENEMR_DB_HOST:-openemr-mysql}"
export OPENEMR_DB_NAME="${OPENEMR_DB_NAME:-openemr}"
export OPENEMR_DB_USER="${OPENEMR_DB_USER:-openemr}"
export OPENEMR_DB_PASS="${OPENEMR_DB_PASS:-openemr_secure_pass_$(date +%s | sha256sum | head -c 8)}"
export OPENEMR_DB_ROOT_PASS="${OPENEMR_DB_ROOT_PASS:-root_secure_pass_$(date +%s | sha256sum | head -c 8)}"

# Admin Credentials  
export OPENEMR_ADMIN_USER="${OPENEMR_ADMIN_USER:-admin}"
export OPENEMR_ADMIN_PASS="${OPENEMR_ADMIN_PASS:-admin_secure_pass_$(date +%s | sha256sum | head -c 8)}"
export OPENEMR_ADMIN_EMAIL="${OPENEMR_ADMIN_EMAIL:-admin@clinic.local}"

# OpenEMR Configuration
export OPENEMR_SITE_ID="${OPENEMR_SITE_ID:-default}"
export OPENEMR_ENABLE_FHIR="${OPENEMR_ENABLE_FHIR:-true}"
export OPENEMR_ENABLE_API="${OPENEMR_ENABLE_API:-true}"
export OPENEMR_ENABLE_PORTAL="${OPENEMR_ENABLE_PORTAL:-true}"
export OPENEMR_TIMEZONE="${OPENEMR_TIMEZONE:-America/New_York}"
export OPENEMR_LANGUAGE="${OPENEMR_LANGUAGE:-English}"

# Docker Configuration
export OPENEMR_DOCKER_IMAGE="${OPENEMR_DOCKER_IMAGE:-openemr/openemr:7.0.2}"
export OPENEMR_MYSQL_IMAGE="${OPENEMR_MYSQL_IMAGE:-mysql:8.0}"
export OPENEMR_NETWORK="${OPENEMR_NETWORK:-openemr-network}"
export OPENEMR_VOLUME_PREFIX="${OPENEMR_VOLUME_PREFIX:-openemr}"

# Container Names
export OPENEMR_WEB_CONTAINER="${OPENEMR_WEB_CONTAINER:-openemr-web}"
export OPENEMR_DB_CONTAINER="${OPENEMR_DB_CONTAINER:-openemr-mysql}"

# Paths
export OPENEMR_DATA_DIR="${OPENEMR_DATA_DIR:-${HOME}/.vrooli/resources/openemr/data}"
export OPENEMR_CONFIG_DIR="${OPENEMR_CONFIG_DIR:-${HOME}/.vrooli/resources/openemr/config}"
export OPENEMR_LOGS_DIR="${OPENEMR_LOGS_DIR:-${HOME}/.vrooli/resources/openemr/logs}"
export OPENEMR_BACKUP_DIR="${OPENEMR_BACKUP_DIR:-${HOME}/.vrooli/resources/openemr/backups}"

# Demo Data
export OPENEMR_LOAD_DEMO_DATA="${OPENEMR_LOAD_DEMO_DATA:-true}"
export OPENEMR_DEMO_PATIENTS="${OPENEMR_DEMO_PATIENTS:-10}"
export OPENEMR_DEMO_PROVIDERS="${OPENEMR_DEMO_PROVIDERS:-3}"

# Performance Tuning
export OPENEMR_PHP_MEMORY="${OPENEMR_PHP_MEMORY:-256M}"
export OPENEMR_MAX_UPLOAD="${OPENEMR_MAX_UPLOAD:-30M}"
export OPENEMR_MAX_EXECUTION="${OPENEMR_MAX_EXECUTION:-60}"
export OPENEMR_MYSQL_BUFFER="${OPENEMR_MYSQL_BUFFER:-256M}"

# Security
export OPENEMR_ENABLE_AUDIT="${OPENEMR_ENABLE_AUDIT:-true}"
export OPENEMR_FORCE_HTTPS="${OPENEMR_FORCE_HTTPS:-false}"
export OPENEMR_SESSION_TIMEOUT="${OPENEMR_SESSION_TIMEOUT:-7200}"
export OPENEMR_PASSWORD_COMPLEXITY="${OPENEMR_PASSWORD_COMPLEXITY:-strong}"

# API Configuration
export OPENEMR_API_RATE_LIMIT="${OPENEMR_API_RATE_LIMIT:-100}"
export OPENEMR_API_TOKEN_EXPIRY="${OPENEMR_API_TOKEN_EXPIRY:-3600}"
export OPENEMR_API_CORS_ORIGIN="${OPENEMR_API_CORS_ORIGIN:-*}"

# Timeouts
export OPENEMR_STARTUP_TIMEOUT="${OPENEMR_STARTUP_TIMEOUT:-60}"
export OPENEMR_HEALTH_CHECK_INTERVAL="${OPENEMR_HEALTH_CHECK_INTERVAL:-30}"
export OPENEMR_HEALTH_CHECK_RETRIES="${OPENEMR_HEALTH_CHECK_RETRIES:-3}"