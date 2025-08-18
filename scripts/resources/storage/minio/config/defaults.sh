#!/usr/bin/env bash
# MinIO Configuration Defaults
# All configuration constants and default values

#######################################
# Export configuration constants
# Idempotent - safe to call multiple times
#######################################
minio::export_config() {
    # Service configuration (only set if not already defined)
    if [[ -z "${MINIO_PORT:-}" ]]; then
        # Try to get port from resources function, fallback to 9000
        local default_port="9000"
        if command -v resources::get_default_port &>/dev/null; then
            default_port=$(resources::get_default_port "minio" 2>/dev/null || echo "9000")
        fi
        readonly MINIO_PORT="${MINIO_CUSTOM_PORT:-$default_port}"
    fi
    if [[ -z "${MINIO_CONSOLE_PORT:-}" ]]; then
        readonly MINIO_CONSOLE_PORT="${MINIO_CUSTOM_CONSOLE_PORT:-9001}"
    fi
    if [[ -z "${MINIO_BASE_URL:-}" ]]; then
        readonly MINIO_BASE_URL="http://localhost:${MINIO_PORT}"
    fi
    if [[ -z "${MINIO_CONSOLE_URL:-}" ]]; then
        readonly MINIO_CONSOLE_URL="http://localhost:${MINIO_CONSOLE_PORT}"
    fi
    if [[ -z "${MINIO_CONTAINER_NAME:-}" ]]; then
        readonly MINIO_CONTAINER_NAME="minio"
    fi
    if [[ -z "${MINIO_DATA_DIR:-}" ]]; then
        readonly MINIO_DATA_DIR="${HOME}/.minio/data"
    fi
    if [[ -z "${MINIO_CONFIG_DIR:-}" ]]; then
        readonly MINIO_CONFIG_DIR="${HOME}/.minio/config"
    fi
    if [[ -z "${MINIO_IMAGE:-}" ]]; then
        readonly MINIO_IMAGE="minio/minio:latest"
    fi
    if [[ -z "${MINIO_VERSION:-}" ]]; then
        readonly MINIO_VERSION="latest"
    fi

    # Credentials (only set if not already defined)
    if [[ -z "${MINIO_ROOT_USER:-}" ]]; then
        # Generate secure username if not provided
        if [[ -z "${MINIO_CUSTOM_ROOT_USER:-}" ]]; then
            readonly MINIO_ROOT_USER="minioadmin"
        else
            readonly MINIO_ROOT_USER="${MINIO_CUSTOM_ROOT_USER}"
        fi
    fi
    if [[ -z "${MINIO_ROOT_PASSWORD:-}" ]]; then
        # Generate secure password if not provided
        if [[ -z "${MINIO_CUSTOM_ROOT_PASSWORD:-}" ]]; then
            readonly MINIO_ROOT_PASSWORD="minioadmin"
        else
            readonly MINIO_ROOT_PASSWORD="${MINIO_CUSTOM_ROOT_PASSWORD}"
        fi
    fi

    # Default buckets for Vrooli
    if [[ -z "${MINIO_DEFAULT_BUCKETS:-}" ]]; then
        readonly MINIO_DEFAULT_BUCKETS=(
            "vrooli-user-uploads"
            "vrooli-agent-artifacts"
            "vrooli-model-cache"
            "vrooli-temp-storage"
        )
    fi

    # Network configuration (only set if not already defined)
    if [[ -z "${MINIO_NETWORK_NAME:-}" ]]; then
        readonly MINIO_NETWORK_NAME="minio-network"
    fi

    # Health check configuration (only set if not already defined)
    if [[ -z "${MINIO_HEALTH_CHECK_INTERVAL:-}" ]]; then
        readonly MINIO_HEALTH_CHECK_INTERVAL=5
    fi
    if [[ -z "${MINIO_HEALTH_CHECK_MAX_ATTEMPTS:-}" ]]; then
        readonly MINIO_HEALTH_CHECK_MAX_ATTEMPTS=12
    fi
    if [[ -z "${MINIO_API_TIMEOUT:-}" ]]; then
        readonly MINIO_API_TIMEOUT=10
    fi

    # Wait timeouts (only set if not already defined)
    if [[ -z "${MINIO_STARTUP_MAX_WAIT:-}" ]]; then
        readonly MINIO_STARTUP_MAX_WAIT=60
    fi
    if [[ -z "${MINIO_STARTUP_WAIT_INTERVAL:-}" ]]; then
        readonly MINIO_STARTUP_WAIT_INTERVAL=2
    fi
    if [[ -z "${MINIO_INITIALIZATION_WAIT:-}" ]]; then
        readonly MINIO_INITIALIZATION_WAIT=10
    fi

    # MinIO specific settings
    if [[ -z "${MINIO_REGION:-}" ]]; then
        readonly MINIO_REGION="us-east-1"
    fi
    if [[ -z "${MINIO_BROWSER:-}" ]]; then
        readonly MINIO_BROWSER="on"
    fi

    # Storage limits
    if [[ -z "${MINIO_MIN_DISK_SPACE_GB:-}" ]]; then
        readonly MINIO_MIN_DISK_SPACE_GB=5
    fi

    # Export for global access
    export MINIO_PORT MINIO_CONSOLE_PORT MINIO_BASE_URL MINIO_CONSOLE_URL
    export MINIO_CONTAINER_NAME MINIO_DATA_DIR MINIO_CONFIG_DIR MINIO_IMAGE
    export MINIO_ROOT_USER MINIO_ROOT_PASSWORD MINIO_DEFAULT_BUCKETS
    export MINIO_NETWORK_NAME MINIO_REGION MINIO_BROWSER
    export MINIO_HEALTH_CHECK_INTERVAL MINIO_HEALTH_CHECK_MAX_ATTEMPTS
    export MINIO_API_TIMEOUT MINIO_STARTUP_MAX_WAIT
    export MINIO_STARTUP_WAIT_INTERVAL MINIO_INITIALIZATION_WAIT
    export MINIO_MIN_DISK_SPACE_GB
}