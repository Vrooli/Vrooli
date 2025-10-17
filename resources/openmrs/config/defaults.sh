#!/usr/bin/env bash
# OpenMRS Default Configuration

# Service ports (from port registry)
export OPENMRS_PORT="${OPENMRS_PORT:-8005}"
export OPENMRS_API_PORT="${OPENMRS_API_PORT:-8006}"
export OPENMRS_FHIR_PORT="${OPENMRS_FHIR_PORT:-8007}"
export OPENMRS_DB_PORT="${OPENMRS_DB_PORT:-5444}"

# Directory paths
export OPENMRS_DIR="${OPENMRS_DIR:-${HOME}/.vrooli/openmrs}"
export OPENMRS_DATA_DIR="${OPENMRS_DATA_DIR:-${OPENMRS_DIR}/data}"
export OPENMRS_CONFIG_DIR="${OPENMRS_CONFIG_DIR:-${OPENMRS_DIR}/config}"
export OPENMRS_LOGS_DIR="${OPENMRS_LOGS_DIR:-${OPENMRS_DIR}/logs}"

# Docker configuration
export OPENMRS_VERSION="${OPENMRS_VERSION:-3.0.0}"
export OPENMRS_NETWORK="${OPENMRS_NETWORK:-openmrs-network}"
export OPENMRS_DB_CONTAINER="${OPENMRS_DB_CONTAINER:-openmrs-postgres}"
export OPENMRS_APP_CONTAINER="${OPENMRS_APP_CONTAINER:-openmrs-app}"

# Database configuration
export OPENMRS_DB_NAME="${OPENMRS_DB_NAME:-openmrs}"
export OPENMRS_DB_USER="${OPENMRS_DB_USER:-openmrs}"

# Admin configuration
export OPENMRS_ADMIN_USER="${OPENMRS_ADMIN_USER:-admin}"
export OPENMRS_ADMIN_PASS="${OPENMRS_ADMIN_PASS:-Admin123}"

# Feature flags
export OPENMRS_ENABLE_FHIR="${OPENMRS_ENABLE_FHIR:-true}"
export OPENMRS_ENABLE_DEMO_DATA="${OPENMRS_ENABLE_DEMO_DATA:-true}"
export OPENMRS_ENABLE_API="${OPENMRS_ENABLE_API:-true}"