#!/usr/bin/env bash
# n8n Configuration Defaults
# All configuration constants and default values

# n8n port configuration - check if already set to avoid readonly conflicts in tests
if ! readonly -p | grep -q "^declare -[a-z]*r[a-z]* N8N_PORT="; then
    readonly N8N_PORT="${N8N_CUSTOM_PORT:-$(resources::get_default_port "n8n")}"
fi
if ! readonly -p | grep -q "^declare -[a-z]*r[a-z]* N8N_BASE_URL="; then
    readonly N8N_BASE_URL="http://localhost:${N8N_PORT}"
fi

# Container configuration - check for readonly conflicts
if ! readonly -p | grep -q "^declare -[a-z]*r[a-z]* N8N_SERVICE_NAME="; then
    readonly N8N_SERVICE_NAME="n8n"
fi
if ! readonly -p | grep -q "^declare -[a-z]*r[a-z]* N8N_CONTAINER_NAME="; then
    readonly N8N_CONTAINER_NAME="n8n"
fi
if ! readonly -p | grep -q "^declare -[a-z]*r[a-z]* N8N_DATA_DIR="; then
    readonly N8N_DATA_DIR="/data/n8n"
fi
if ! readonly -p | grep -q "^declare -[a-z]*r[a-z]* N8N_IMAGE="; then
    readonly N8N_IMAGE="docker.n8n.io/n8nio/n8n:latest"
fi

# Custom image configuration
readonly N8N_CUSTOM_IMAGE="${N8N_CUSTOM_IMAGE:-n8n-vrooli:latest}"
readonly N8N_USE_CUSTOM_IMAGE="${N8N_USE_CUSTOM_IMAGE:-no}"

# Database configuration
readonly N8N_DB_TYPE="${N8N_DB_TYPE:-sqlite}"
readonly N8N_DB_CONTAINER_NAME="vrooli-n8n-postgres"
readonly N8N_DB_IMAGE="postgres:13-alpine"
readonly N8N_DB_PORT="5432"
readonly N8N_DB_POSTGRESDB_HOST="localhost"
readonly N8N_DB_POSTGRESDB_PORT="5432"
readonly N8N_DB_POSTGRESDB_DATABASE="n8n"
readonly N8N_DB_POSTGRESDB_USER="n8n"
readonly N8N_DB_POSTGRESDB_PASSWORD="n8n-secure-password"
readonly N8N_DB_PASSWORD="n8n-secure-password-$(date +%s | sha256sum | head -c 16)"

# Network configuration
readonly N8N_NETWORK_NAME="n8n-network"

# Health check configuration
readonly N8N_HEALTH_CHECK_INTERVAL=5
readonly N8N_HEALTH_CHECK_MAX_ATTEMPTS=30

# API configuration
readonly N8N_API_TIMEOUT=30

# Backup configuration
readonly N8N_BACKUP_DIR="${HOME}/.n8n-backup"

# Log configuration
readonly N8N_LOG_LINES=100

# Password reset configuration
readonly N8N_PASSWORD_RESET_TIMEOUT=60

# Default credentials
readonly N8N_DEFAULT_USERNAME="admin"
readonly N8N_DEFAULT_EMAIL="admin@example.com"
readonly N8N_DEFAULT_FIRSTNAME="Admin"
readonly N8N_DEFAULT_LASTNAME="User"

# Feature flags
readonly N8N_ENABLE_BASIC_AUTH="${N8N_ENABLE_BASIC_AUTH:-yes}"
readonly N8N_ENABLE_POSTGRES="${N8N_ENABLE_POSTGRES:-no}"

# Export function to make configuration available
n8n::export_config() {
    # Export all readonly variables
    export N8N_PORT N8N_BASE_URL N8N_CONTAINER_NAME N8N_DATA_DIR N8N_IMAGE
    export N8N_CUSTOM_IMAGE N8N_USE_CUSTOM_IMAGE
    export N8N_DB_TYPE N8N_DB_CONTAINER_NAME N8N_DB_PORT N8N_DB_PASSWORD
    export N8N_NETWORK_NAME
    export N8N_HEALTH_CHECK_INTERVAL N8N_HEALTH_CHECK_MAX_ATTEMPTS
    export N8N_API_TIMEOUT
    export N8N_BACKUP_DIR N8N_LOG_LINES
    export N8N_PASSWORD_RESET_TIMEOUT
    export N8N_DEFAULT_USERNAME N8N_DEFAULT_EMAIL N8N_DEFAULT_FIRSTNAME N8N_DEFAULT_LASTNAME
    export N8N_ENABLE_BASIC_AUTH N8N_ENABLE_POSTGRES
}