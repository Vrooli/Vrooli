#!/bin/bash
# Step-CA Default Configuration

# Resource identity
export STEPCA_RESOURCE_NAME="step-ca"
export STEPCA_CONTAINER_NAME="vrooli-step-ca"
export STEPCA_DOCKER_IMAGE="smallstep/step-ca:latest"

# Network configuration
# Port is retrieved from port registry, never hardcoded
export STEPCA_NETWORK="vrooli-network"

# Data persistence
export STEPCA_DATA_ROOT="${VROOLI_ROOT:-$HOME/Vrooli}/data/step-ca"
export STEPCA_CONFIG_DIR="$STEPCA_DATA_ROOT/config"
export STEPCA_CERTS_DIR="$STEPCA_DATA_ROOT/certs"

# CA Configuration
export STEPCA_CA_NAME="${STEPCA_CA_NAME:-Vrooli CA}"
export STEPCA_CA_DNS_NAMES="${STEPCA_CA_DNS_NAMES:-localhost,vrooli-step-ca}"
export STEPCA_PROVISIONER_NAME="${STEPCA_PROVISIONER_NAME:-admin}"
export STEPCA_DEFAULT_DURATION="${STEPCA_DEFAULT_DURATION:-24h}"
export STEPCA_MAX_DURATION="${STEPCA_MAX_DURATION:-720h}"

# Database backend (optional)
export STEPCA_DB_TYPE="${STEPCA_DB_TYPE:-badger}"  # badger, bbolt, postgresql, mysql
export STEPCA_DB_DATASOURCE="${STEPCA_DB_DATASOURCE:-}"  # For PostgreSQL/MySQL

# Performance tuning
export STEPCA_MAX_CONNECTIONS="${STEPCA_MAX_CONNECTIONS:-100}"
export STEPCA_CACHE_SIZE="${STEPCA_CACHE_SIZE:-10000}"

# Security settings
export STEPCA_REQUIRE_OIDC="${STEPCA_REQUIRE_OIDC:-false}"
export STEPCA_OIDC_CLIENT_ID="${STEPCA_OIDC_CLIENT_ID:-}"
export STEPCA_OIDC_CLIENT_SECRET="${STEPCA_OIDC_CLIENT_SECRET:-}"
export STEPCA_OIDC_ISSUER="${STEPCA_OIDC_ISSUER:-}"

# Operational settings
export STEPCA_STARTUP_TIMEOUT="${STEPCA_STARTUP_TIMEOUT:-60}"
export STEPCA_SHUTDOWN_TIMEOUT="${STEPCA_SHUTDOWN_TIMEOUT:-30}"
export STEPCA_HEALTH_CHECK_INTERVAL="${STEPCA_HEALTH_CHECK_INTERVAL:-30}"
export STEPCA_LOG_LEVEL="${STEPCA_LOG_LEVEL:-info}"

# Docker resource limits
export STEPCA_MEMORY_LIMIT="${STEPCA_MEMORY_LIMIT:-512m}"
export STEPCA_CPU_LIMIT="${STEPCA_CPU_LIMIT:-0.5}"

# Backup settings
export STEPCA_BACKUP_ENABLED="${STEPCA_BACKUP_ENABLED:-true}"
export STEPCA_BACKUP_RETENTION_DAYS="${STEPCA_BACKUP_RETENTION_DAYS:-30}"