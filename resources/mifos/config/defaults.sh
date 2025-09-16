#!/usr/bin/env bash
################################################################################
# Mifos Resource - Default Configuration
# 
# Provides default values for all Mifos configuration variables
# Can be overridden by environment variables or runtime configuration
################################################################################

# Service Configuration
export MIFOS_PORT="${MIFOS_PORT:-8030}"
export MIFOS_HOST="${MIFOS_HOST:-localhost}"
export MIFOS_NETWORK="${MIFOS_NETWORK:-mifos-network}"
export MIFOS_DATA_DIR="${MIFOS_DATA_DIR:-${HOME}/.vrooli/mifos/data}"
export MIFOS_LOG_DIR="${MIFOS_LOG_DIR:-${HOME}/.vrooli/mifos/logs}"

# Fineract Backend Configuration  
export FINERACT_VERSION="${FINERACT_VERSION:-latest}"
export FINERACT_DB_HOST="${FINERACT_DB_HOST:-localhost}"
export FINERACT_DB_PORT="${FINERACT_DB_PORT:-5433}"
export FINERACT_DB_NAME="${FINERACT_DB_NAME:-fineract}"
export FINERACT_DB_USER="${FINERACT_DB_USER:-fineract}"
export FINERACT_DB_PASSWORD="${FINERACT_DB_PASSWORD:-fineract_password}"
export FINERACT_TENANT="${FINERACT_TENANT:-default}"

# Web App Configuration
export MIFOS_WEBAPP_IMAGE="${MIFOS_WEBAPP_IMAGE:-openmf/community-app}"
export MIFOS_WEBAPP_VERSION="${MIFOS_WEBAPP_VERSION:-latest}"
export MIFOS_WEBAPP_PORT="${MIFOS_WEBAPP_PORT:-8031}"

# API Configuration
export MIFOS_API_URL="${MIFOS_API_URL:-https://${MIFOS_HOST}:${MIFOS_PORT}}"
export MIFOS_API_VERSION="${MIFOS_API_VERSION:-v1}"
export MIFOS_DEFAULT_USER="${MIFOS_DEFAULT_USER:-mifos}"
export MIFOS_DEFAULT_PASSWORD="${MIFOS_DEFAULT_PASSWORD:-password}"

# Docker Configuration
export MIFOS_DOCKER_COMPOSE="${MIFOS_DOCKER_COMPOSE:-${MIFOS_CLI_DIR}/docker/docker-compose.yml}"
export MIFOS_CONTAINER_PREFIX="${MIFOS_CONTAINER_PREFIX:-mifos}"

# Resource Dependencies
export MIFOS_POSTGRES_RESOURCE="${MIFOS_POSTGRES_RESOURCE:-postgres}"
export MIFOS_REDIS_RESOURCE="${MIFOS_REDIS_RESOURCE:-redis}"

# Timeout Configuration
export MIFOS_STARTUP_TIMEOUT="${MIFOS_STARTUP_TIMEOUT:-120}"
export MIFOS_HEALTH_CHECK_INTERVAL="${MIFOS_HEALTH_CHECK_INTERVAL:-5}"
export MIFOS_HEALTH_CHECK_RETRIES="${MIFOS_HEALTH_CHECK_RETRIES:-24}"

# Demo Data Configuration
export MIFOS_SEED_DEMO_DATA="${MIFOS_SEED_DEMO_DATA:-true}"
export MIFOS_DEMO_CLIENTS_COUNT="${MIFOS_DEMO_CLIENTS_COUNT:-10}"
export MIFOS_DEMO_LOANS_COUNT="${MIFOS_DEMO_LOANS_COUNT:-5}"

# Multi-Currency Configuration
export MIFOS_CURRENCIES="${MIFOS_CURRENCIES:-USD,EUR,GBP}"
export MIFOS_BASE_CURRENCY="${MIFOS_BASE_CURRENCY:-USD}"

# Security Configuration
export MIFOS_SSL_ENABLED="${MIFOS_SSL_ENABLED:-false}"
export MIFOS_TWO_FACTOR_AUTH="${MIFOS_TWO_FACTOR_AUTH:-false}"

# Performance Configuration
export MIFOS_MAX_CONNECTIONS="${MIFOS_MAX_CONNECTIONS:-100}"
export MIFOS_CONNECTION_TIMEOUT="${MIFOS_CONNECTION_TIMEOUT:-30}"

# Logging Configuration
export MIFOS_LOG_LEVEL="${MIFOS_LOG_LEVEL:-INFO}"
export MIFOS_LOG_FORMAT="${MIFOS_LOG_FORMAT:-json}"