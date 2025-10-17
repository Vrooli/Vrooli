#!/usr/bin/env bash
# Windmill Configuration Defaults
# All configuration constants and default values for Windmill workflow automation

# Windmill port configuration
readonly WINDMILL_SERVER_PORT="${WINDMILL_CUSTOM_PORT:-$(resources::get_default_port "windmill")}"
readonly WINDMILL_BASE_URL="http://localhost:${WINDMILL_SERVER_PORT}"

# Project and container configuration
readonly WINDMILL_PROJECT_NAME="windmill-vrooli"
readonly WINDMILL_COMPOSE_FILE="${APP_ROOT}/resources/windmill/docker/docker-compose.yml"
readonly WINDMILL_ENV_FILE="${APP_ROOT}/resources/windmill/docker/.env"
readonly WINDMILL_DATA_DIR="${HOME}/.windmill"
readonly WINDMILL_NETWORK_NAME="${WINDMILL_PROJECT_NAME}_default"

# Windmill image configuration
readonly WINDMILL_IMAGE="${WM_IMAGE:-ghcr.io/windmill-labs/windmill:main}"
readonly WINDMILL_IMAGE_TAG="${WINDMILL_IMAGE_TAG:-main}"

# Database configuration
readonly WINDMILL_DB_TYPE="${WINDMILL_DB_TYPE:-postgres}"
readonly WINDMILL_DB_EXTERNAL="${WINDMILL_DB_EXTERNAL:-no}"
readonly WINDMILL_DB_URL="${WINDMILL_DB_URL:-}"
readonly WINDMILL_DB_CONTAINER_NAME="${WINDMILL_PROJECT_NAME}-db"
readonly WINDMILL_DB_PORT="5432"
readonly WINDMILL_DB_NAME="windmill"
readonly WINDMILL_DB_USER="postgres"
# Password will be managed by state system - placeholder for now
WINDMILL_DB_PASSWORD="${WINDMILL_DB_PASSWORD:-}"

# Worker configuration
readonly WINDMILL_WORKER_REPLICAS="${WINDMILL_WORKER_REPLICAS:-3}"
readonly WINDMILL_NATIVE_WORKER="${WINDMILL_NATIVE_WORKER:-yes}"
readonly WINDMILL_WORKER_MEMORY_LIMIT="${WINDMILL_WORKER_MEMORY_LIMIT:-2048M}"
readonly WINDMILL_WORKER_GROUP="${WINDMILL_WORKER_GROUP:-default}"

# Security and authentication
readonly WINDMILL_SUPERADMIN_EMAIL="${WINDMILL_SUPERADMIN_EMAIL:-admin@windmill.dev}"
readonly WINDMILL_SUPERADMIN_PASSWORD="${WINDMILL_SUPERADMIN_PASSWORD:-changeme}"
readonly WINDMILL_JWT_SECRET="${WINDMILL_JWT_SECRET:-windmill-jwt-$(openssl rand -hex 32 2>/dev/null || echo "fallback-secret-$(date +%s)")}"

# API and health check configuration
readonly WINDMILL_API_TIMEOUT=30
readonly WINDMILL_HEALTH_CHECK_INTERVAL=10
readonly WINDMILL_HEALTH_CHECK_MAX_ATTEMPTS=30
readonly WINDMILL_STARTUP_TIMEOUT=120

# Service names
readonly WINDMILL_SERVER_CONTAINER="${WINDMILL_PROJECT_NAME}-app"
readonly WINDMILL_WORKER_CONTAINER="${WINDMILL_PROJECT_NAME}-worker"
readonly WINDMILL_LSP_CONTAINER="${WINDMILL_PROJECT_NAME}-lsp"

# Logging configuration
readonly WINDMILL_LOG_LINES=100
readonly WINDMILL_LOG_LEVEL="${WINDMILL_LOG_LEVEL:-info}"

# Backup configuration
readonly WINDMILL_BACKUP_DIR="${HOME}/.windmill-backup"
readonly WINDMILL_BACKUP_RETENTION_DAYS=7

# Feature flags
readonly WINDMILL_ENABLE_LSP="${WINDMILL_ENABLE_LSP:-yes}"
readonly WINDMILL_ENABLE_MULTIPLAYER="${WINDMILL_ENABLE_MULTIPLAYER:-no}"
readonly WINDMILL_DISABLE_NUSER="${WINDMILL_DISABLE_NUSER:-false}"
readonly WINDMILL_KEEP_JOB_DIR="${WINDMILL_KEEP_JOB_DIR:-false}"

# Development and debugging
readonly WINDMILL_DEBUG="${WINDMILL_DEBUG:-no}"
readonly WINDMILL_DEV_MODE="${WINDMILL_DEV_MODE:-no}"

# Resource requirements (minimum)
readonly WINDMILL_MIN_MEMORY_GB=4
readonly WINDMILL_MIN_DISK_GB=5
readonly WINDMILL_MIN_CPU_CORES=2

#######################################
# Export function to make configuration available
#######################################
windmill::export_config() {
    # Export all readonly variables for use by other scripts
    export WINDMILL_SERVER_PORT WINDMILL_BASE_URL
    export WINDMILL_PROJECT_NAME WINDMILL_COMPOSE_FILE WINDMILL_ENV_FILE
    export WINDMILL_DATA_DIR WINDMILL_NETWORK_NAME
    export WINDMILL_IMAGE WINDMILL_IMAGE_TAG
    export WINDMILL_DB_TYPE WINDMILL_DB_EXTERNAL WINDMILL_DB_URL
    export WINDMILL_DB_CONTAINER_NAME WINDMILL_DB_PORT WINDMILL_DB_NAME
    export WINDMILL_DB_USER WINDMILL_DB_PASSWORD
    export WINDMILL_WORKER_REPLICAS WINDMILL_NATIVE_WORKER
    export WINDMILL_WORKER_MEMORY_LIMIT WINDMILL_WORKER_GROUP
    export WINDMILL_SUPERADMIN_EMAIL WINDMILL_SUPERADMIN_PASSWORD WINDMILL_JWT_SECRET
    export WINDMILL_API_TIMEOUT WINDMILL_HEALTH_CHECK_INTERVAL
    export WINDMILL_HEALTH_CHECK_MAX_ATTEMPTS WINDMILL_STARTUP_TIMEOUT
    export WINDMILL_SERVER_CONTAINER WINDMILL_WORKER_CONTAINER WINDMILL_LSP_CONTAINER
    export WINDMILL_LOG_LINES WINDMILL_LOG_LEVEL
    export WINDMILL_BACKUP_DIR WINDMILL_BACKUP_RETENTION_DAYS
    export WINDMILL_ENABLE_LSP WINDMILL_ENABLE_MULTIPLAYER
    export WINDMILL_DISABLE_NUSER WINDMILL_KEEP_JOB_DIR
    export WINDMILL_DEBUG WINDMILL_DEV_MODE
    export WINDMILL_MIN_MEMORY_GB WINDMILL_MIN_DISK_GB WINDMILL_MIN_CPU_CORES
}

#######################################
# Validate configuration
#######################################
windmill::validate_config() {
    local errors=()
    
    # Check worker replicas is a valid number
    if ! [[ "$WINDMILL_WORKER_REPLICAS" =~ ^[1-9][0-9]*$ ]]; then
        errors+=("WINDMILL_WORKER_REPLICAS must be a positive integer, got: $WINDMILL_WORKER_REPLICAS")
    fi
    
    # Check memory limit format
    if ! [[ "$WINDMILL_WORKER_MEMORY_LIMIT" =~ ^[0-9]+[MG]$ ]]; then
        errors+=("WINDMILL_WORKER_MEMORY_LIMIT must be in format like '2048M' or '2G', got: $WINDMILL_WORKER_MEMORY_LIMIT")
    fi
    
    # Check port is valid
    if ! [[ "$WINDMILL_SERVER_PORT" =~ ^[1-9][0-9]*$ ]] || [[ $WINDMILL_SERVER_PORT -lt 1024 ]] || [[ $WINDMILL_SERVER_PORT -gt 65535 ]]; then
        errors+=("WINDMILL_SERVER_PORT must be between 1024-65535, got: $WINDMILL_SERVER_PORT")
    fi
    
    # Check email format (basic)
    if ! [[ "$WINDMILL_SUPERADMIN_EMAIL" =~ ^[^@]+@[^@]+\.[^@]+$ ]]; then
        errors+=("WINDMILL_SUPERADMIN_EMAIL must be a valid email address, got: $WINDMILL_SUPERADMIN_EMAIL")
    fi
    
    if [[ ${#errors[@]} -gt 0 ]]; then
        log::error "Configuration validation failed:"
        for error in "${errors[@]}"; do
            log::error "  • $error"
        done
        return 1
    fi
    
    return 0
}