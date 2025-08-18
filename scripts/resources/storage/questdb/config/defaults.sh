#!/usr/bin/env bash
# QuestDB Configuration Defaults
# All configuration constants and default values

# Source common functions for port registry access
QUESTDB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck disable=SC1091
source "${QUESTDB_DIR}/../../common.sh"

#######################################
# Export configuration constants
# Idempotent - safe to call multiple times
#######################################
questdb::export_config() {
    # Service configuration (only set if not already defined)
    if [[ -z "${QUESTDB_HTTP_PORT:-}" ]]; then
        readonly QUESTDB_HTTP_PORT="${QUESTDB_CUSTOM_HTTP_PORT:-$(resources::get_default_port "questdb")}"
    fi
    if [[ -z "${QUESTDB_PG_PORT:-}" ]]; then
        readonly QUESTDB_PG_PORT="${QUESTDB_CUSTOM_PG_PORT:-8812}"
    fi
    if [[ -z "${QUESTDB_ILP_PORT:-}" ]]; then
        readonly QUESTDB_ILP_PORT="${QUESTDB_CUSTOM_ILP_PORT:-9011}"
    fi
    if [[ -z "${QUESTDB_BASE_URL:-}" ]]; then
        readonly QUESTDB_BASE_URL="http://localhost:${QUESTDB_HTTP_PORT}"
    fi
    if [[ -z "${QUESTDB_PG_URL:-}" ]]; then
        readonly QUESTDB_PG_URL="postgresql://admin:quest@localhost:${QUESTDB_PG_PORT}/qdb"
    fi
    if [[ -z "${QUESTDB_CONTAINER_NAME:-}" ]]; then
        readonly QUESTDB_CONTAINER_NAME="vrooli-questdb"
    fi
    if [[ -z "${QUESTDB_DATA_DIR:-}" ]]; then
        readonly QUESTDB_DATA_DIR="${HOME}/.questdb/data"
    fi
    if [[ -z "${QUESTDB_CONFIG_DIR:-}" ]]; then
        readonly QUESTDB_CONFIG_DIR="${HOME}/.questdb/config"
    fi
    if [[ -z "${QUESTDB_LOG_DIR:-}" ]]; then
        readonly QUESTDB_LOG_DIR="${HOME}/.questdb/logs"
    fi
    if [[ -z "${QUESTDB_IMAGE:-}" ]]; then
        readonly QUESTDB_IMAGE="questdb/questdb:8.1.2"
    fi

    # Performance configuration (only set if not already defined)
    if [[ -z "${QUESTDB_SHARED_WORKER_COUNT:-}" ]]; then
        readonly QUESTDB_SHARED_WORKER_COUNT="2"
    fi
    if [[ -z "${QUESTDB_HTTP_WORKER_COUNT:-}" ]]; then
        readonly QUESTDB_HTTP_WORKER_COUNT="2"
    fi
    if [[ -z "${QUESTDB_WAL_ENABLED:-}" ]]; then
        readonly QUESTDB_WAL_ENABLED="true"
    fi
    if [[ -z "${QUESTDB_COMMIT_LAG:-}" ]]; then
        readonly QUESTDB_COMMIT_LAG="300000"  # 5 minutes in milliseconds
    fi

    # Security configuration (only set if not already defined)
    if [[ -z "${QUESTDB_HTTP_SECURITY_READONLY:-}" ]]; then
        readonly QUESTDB_HTTP_SECURITY_READONLY="false"
    fi
    if [[ -z "${QUESTDB_PG_USER:-}" ]]; then
        readonly QUESTDB_PG_USER="admin"
    fi
    if [[ -z "${QUESTDB_PG_PASSWORD:-}" ]]; then
        readonly QUESTDB_PG_PASSWORD="quest"
    fi

    # Network configuration (only set if not already defined)
    if [[ -z "${QUESTDB_NETWORK_NAME:-}" ]]; then
        readonly QUESTDB_NETWORK_NAME="questdb-network"
    fi

    # Health check configuration (only set if not already defined)
    if [[ -z "${QUESTDB_HEALTH_CHECK_INTERVAL:-}" ]]; then
        readonly QUESTDB_HEALTH_CHECK_INTERVAL=10
    fi
    if [[ -z "${QUESTDB_HEALTH_CHECK_MAX_ATTEMPTS:-}" ]]; then
        readonly QUESTDB_HEALTH_CHECK_MAX_ATTEMPTS=6
    fi
    if [[ -z "${QUESTDB_API_TIMEOUT:-}" ]]; then
        readonly QUESTDB_API_TIMEOUT=15
    fi

    # Wait timeouts (only set if not already defined)
    if [[ -z "${QUESTDB_STARTUP_MAX_WAIT:-}" ]]; then
        readonly QUESTDB_STARTUP_MAX_WAIT=60
    fi
    if [[ -z "${QUESTDB_STARTUP_WAIT_INTERVAL:-}" ]]; then
        readonly QUESTDB_STARTUP_WAIT_INTERVAL=2
    fi

    # Default tables for time-series data
    if [[ -z "${QUESTDB_DEFAULT_TABLES:-}" ]]; then
        readonly QUESTDB_DEFAULT_TABLES=(
            "system_metrics"
            "ai_inference"
            "resource_health"
            "workflow_metrics"
        )
    fi

    # Export for global access
    export QUESTDB_HTTP_PORT QUESTDB_PG_PORT QUESTDB_ILP_PORT
    export QUESTDB_BASE_URL QUESTDB_PG_URL QUESTDB_CONTAINER_NAME
    export QUESTDB_DATA_DIR QUESTDB_CONFIG_DIR QUESTDB_LOG_DIR QUESTDB_IMAGE
    export QUESTDB_SHARED_WORKER_COUNT QUESTDB_HTTP_WORKER_COUNT
    export QUESTDB_WAL_ENABLED QUESTDB_COMMIT_LAG
    export QUESTDB_HTTP_SECURITY_READONLY QUESTDB_PG_USER QUESTDB_PG_PASSWORD
    export QUESTDB_NETWORK_NAME
    export QUESTDB_HEALTH_CHECK_INTERVAL QUESTDB_HEALTH_CHECK_MAX_ATTEMPTS
    export QUESTDB_API_TIMEOUT QUESTDB_STARTUP_MAX_WAIT QUESTDB_STARTUP_WAIT_INTERVAL
    export QUESTDB_DEFAULT_TABLES
}