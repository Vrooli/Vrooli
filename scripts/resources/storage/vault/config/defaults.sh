#!/usr/bin/env bash
# Vault Configuration Defaults
# All configuration constants and default values

#######################################
# Export configuration constants
# Idempotent - safe to call multiple times
#######################################
vault::export_config() {
    # Service configuration (only set if not already defined)
    if [[ -z "${VAULT_PORT:-}" ]]; then
        readonly VAULT_PORT="${VAULT_CUSTOM_PORT:-$(resources::get_default_port "vault")}"
    fi
    if [[ -z "${VAULT_BASE_URL:-}" ]]; then
        readonly VAULT_BASE_URL="http://localhost:${VAULT_PORT}"
    fi
    if [[ -z "${VAULT_CONTAINER_NAME:-}" ]]; then
        readonly VAULT_CONTAINER_NAME="vault"
    fi
    if [[ -z "${VAULT_DATA_DIR:-}" ]]; then
        readonly VAULT_DATA_DIR="${HOME}/.vault/data"
    fi
    if [[ -z "${VAULT_CONFIG_DIR:-}" ]]; then
        readonly VAULT_CONFIG_DIR="${HOME}/.vault/config"
    fi
    if [[ -z "${VAULT_LOGS_DIR:-}" ]]; then
        readonly VAULT_LOGS_DIR="${HOME}/.vault/logs"
    fi
    if [[ -z "${VAULT_IMAGE:-}" ]]; then
        readonly VAULT_IMAGE="hashicorp/vault:1.17"
    fi

    # Vault mode configuration
    if [[ -z "${VAULT_MODE:-}" ]]; then
        VAULT_MODE="${VAULT_CUSTOM_MODE:-dev}"  # dev or prod - defaulting to dev for now until TLS issues are resolved
    fi
    
    # Storage strategy configuration
    if [[ -z "${VAULT_STORAGE_STRATEGY:-}" ]]; then
        # Options: volumes (Docker volumes), bind (bind mounts), auto (auto-detect)
        VAULT_STORAGE_STRATEGY="${VAULT_CUSTOM_STORAGE_STRATEGY:-volumes}"
    fi
    
    # Docker volume names (for volume strategy)
    if [[ -z "${VAULT_VOLUME_DATA:-}" ]]; then
        readonly VAULT_VOLUME_DATA="vault-data"
    fi
    if [[ -z "${VAULT_VOLUME_CONFIG:-}" ]]; then
        readonly VAULT_VOLUME_CONFIG="vault-config"
    fi
    if [[ -z "${VAULT_VOLUME_LOGS:-}" ]]; then
        readonly VAULT_VOLUME_LOGS="vault-logs"
    fi
    
    # Container UID/GID for permission mapping
    if [[ -z "${VAULT_CONTAINER_UID:-}" ]]; then
        readonly VAULT_CONTAINER_UID="100"
    fi
    if [[ -z "${VAULT_CONTAINER_GID:-}" ]]; then
        readonly VAULT_CONTAINER_GID="1000"
    fi

    # Development mode configuration
    if [[ -z "${VAULT_DEV_ROOT_TOKEN_ID:-}" ]]; then
        readonly VAULT_DEV_ROOT_TOKEN_ID="${VAULT_CUSTOM_DEV_ROOT_TOKEN_ID:-myroot}"
    fi
    if [[ -z "${VAULT_DEV_LISTEN_ADDRESS:-}" ]]; then
        readonly VAULT_DEV_LISTEN_ADDRESS="0.0.0.0:${VAULT_PORT}"
    fi

    # Production mode configuration
    if [[ -z "${VAULT_TLS_DISABLE:-}" ]]; then
        # Note: TLS will be enabled automatically in prod mode when certificates are generated
        readonly VAULT_TLS_DISABLE="${VAULT_CUSTOM_TLS_DISABLE:-1}"  # 1 to disable TLS (will be overridden in prod)
    fi
    if [[ -z "${VAULT_STORAGE_TYPE:-}" ]]; then
        readonly VAULT_STORAGE_TYPE="${VAULT_CUSTOM_STORAGE_TYPE:-file}"  # file or consul
    fi

    # Secret namespace configuration
    if [[ -z "${VAULT_NAMESPACE_PREFIX:-}" ]]; then
        readonly VAULT_NAMESPACE_PREFIX="vrooli"
    fi
    if [[ -z "${VAULT_SECRET_ENGINE:-}" ]]; then
        readonly VAULT_SECRET_ENGINE="secret"
    fi
    if [[ -z "${VAULT_SECRET_VERSION:-}" ]]; then
        readonly VAULT_SECRET_VERSION="2"  # KV version 2
    fi

    # Network configuration
    if [[ -z "${VAULT_NETWORK_NAME:-}" ]]; then
        readonly VAULT_NETWORK_NAME="vault-network"
    fi

    # Health check configuration
    if [[ -z "${VAULT_HEALTH_CHECK_INTERVAL:-}" ]]; then
        readonly VAULT_HEALTH_CHECK_INTERVAL=5
    fi
    if [[ -z "${VAULT_HEALTH_CHECK_MAX_ATTEMPTS:-}" ]]; then
        readonly VAULT_HEALTH_CHECK_MAX_ATTEMPTS=12
    fi
    if [[ -z "${VAULT_API_TIMEOUT:-}" ]]; then
        readonly VAULT_API_TIMEOUT=10
    fi

    # Wait timeouts
    if [[ -z "${VAULT_STARTUP_MAX_WAIT:-}" ]]; then
        readonly VAULT_STARTUP_MAX_WAIT=60
    fi
    if [[ -z "${VAULT_STARTUP_WAIT_INTERVAL:-}" ]]; then
        readonly VAULT_STARTUP_WAIT_INTERVAL=2
    fi
    if [[ -z "${VAULT_INITIALIZATION_WAIT:-}" ]]; then
        readonly VAULT_INITIALIZATION_WAIT=10
    fi

    # Token and authentication
    if [[ -z "${VAULT_TOKEN_FILE:-}" ]]; then
        readonly VAULT_TOKEN_FILE="${VAULT_CONFIG_DIR}/root-token"
    fi
    if [[ -z "${VAULT_UNSEAL_KEYS_FILE:-}" ]]; then
        readonly VAULT_UNSEAL_KEYS_FILE="${VAULT_CONFIG_DIR}/unseal-keys"
    fi

    # Audit logging
    if [[ -z "${VAULT_AUDIT_LOG_FILE:-}" ]]; then
        readonly VAULT_AUDIT_LOG_FILE="${VAULT_LOGS_DIR}/audit.log"
    fi

    # Default secret paths for Vrooli integration
    if [[ -z "${VAULT_DEFAULT_PATHS:-}" ]]; then
        readonly VAULT_DEFAULT_PATHS=(
            "environments/development"
            "environments/staging"
            "environments/production"
            "resources"
            "clients"
            "ephemeral"
        )
    fi

    # Export for global access
    export VAULT_PORT VAULT_BASE_URL VAULT_CONTAINER_NAME
    export VAULT_DATA_DIR VAULT_CONFIG_DIR VAULT_LOGS_DIR VAULT_IMAGE
    export VAULT_MODE VAULT_DEV_ROOT_TOKEN_ID VAULT_DEV_LISTEN_ADDRESS
    export VAULT_TLS_DISABLE VAULT_STORAGE_TYPE
    export VAULT_STORAGE_STRATEGY VAULT_VOLUME_DATA VAULT_VOLUME_CONFIG VAULT_VOLUME_LOGS
    export VAULT_CONTAINER_UID VAULT_CONTAINER_GID
    export VAULT_NAMESPACE_PREFIX VAULT_SECRET_ENGINE VAULT_SECRET_VERSION
    export VAULT_NETWORK_NAME
    export VAULT_HEALTH_CHECK_INTERVAL VAULT_HEALTH_CHECK_MAX_ATTEMPTS
    export VAULT_API_TIMEOUT VAULT_STARTUP_MAX_WAIT
    export VAULT_STARTUP_WAIT_INTERVAL VAULT_INITIALIZATION_WAIT
    export VAULT_TOKEN_FILE VAULT_UNSEAL_KEYS_FILE VAULT_AUDIT_LOG_FILE
    export VAULT_DEFAULT_PATHS
}