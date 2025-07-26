#!/usr/bin/env bash
# Browserless Configuration Defaults
# All configuration constants and default values

#######################################
# Export configuration constants
# Idempotent - safe to call multiple times
#######################################
browserless::export_config() {
    # Service configuration (only set if not already defined)
    if [[ -z "${BROWSERLESS_PORT:-}" ]]; then
        readonly BROWSERLESS_PORT="${BROWSERLESS_CUSTOM_PORT:-$(resources::get_default_port "browserless")}"
    fi
    if [[ -z "${BROWSERLESS_BASE_URL:-}" ]]; then
        readonly BROWSERLESS_BASE_URL="http://localhost:${BROWSERLESS_PORT}"
    fi
    if [[ -z "${BROWSERLESS_CONTAINER_NAME:-}" ]]; then
        readonly BROWSERLESS_CONTAINER_NAME="browserless"
    fi
    if [[ -z "${BROWSERLESS_DATA_DIR:-}" ]]; then
        readonly BROWSERLESS_DATA_DIR="${HOME}/.browserless"
    fi
    if [[ -z "${BROWSERLESS_IMAGE:-}" ]]; then
        readonly BROWSERLESS_IMAGE="ghcr.io/browserless/chrome:latest"
    fi

    # Browser configuration (only set if not already defined)
    if [[ -z "${BROWSERLESS_MAX_BROWSERS:-}" ]]; then
        readonly BROWSERLESS_MAX_BROWSERS="${MAX_BROWSERS:-5}"
    fi
    if [[ -z "${BROWSERLESS_TIMEOUT:-}" ]]; then
        readonly BROWSERLESS_TIMEOUT="${TIMEOUT:-30000}"
    fi
    if [[ -z "${BROWSERLESS_HEADLESS:-}" ]]; then
        readonly BROWSERLESS_HEADLESS="${HEADLESS:-yes}"
    fi

    # Network configuration (only set if not already defined)
    if [[ -z "${BROWSERLESS_NETWORK_NAME:-}" ]]; then
        readonly BROWSERLESS_NETWORK_NAME="browserless-network"
    fi

    # Health check configuration (only set if not already defined)
    if [[ -z "${BROWSERLESS_HEALTH_CHECK_INTERVAL:-}" ]]; then
        readonly BROWSERLESS_HEALTH_CHECK_INTERVAL=5
    fi
    if [[ -z "${BROWSERLESS_HEALTH_CHECK_MAX_ATTEMPTS:-}" ]]; then
        readonly BROWSERLESS_HEALTH_CHECK_MAX_ATTEMPTS=12
    fi
    if [[ -z "${BROWSERLESS_API_TIMEOUT:-}" ]]; then
        readonly BROWSERLESS_API_TIMEOUT=10
    fi

    # Wait timeouts (only set if not already defined)
    if [[ -z "${BROWSERLESS_STARTUP_MAX_WAIT:-}" ]]; then
        readonly BROWSERLESS_STARTUP_MAX_WAIT=60
    fi
    if [[ -z "${BROWSERLESS_STARTUP_WAIT_INTERVAL:-}" ]]; then
        readonly BROWSERLESS_STARTUP_WAIT_INTERVAL=2
    fi
    if [[ -z "${BROWSERLESS_INITIALIZATION_WAIT:-}" ]]; then
        readonly BROWSERLESS_INITIALIZATION_WAIT=10
    fi

    # Security settings for Docker (only set if not already defined)
    if [[ -z "${BROWSERLESS_DOCKER_SHM_SIZE:-}" ]]; then
        readonly BROWSERLESS_DOCKER_SHM_SIZE="2gb"
    fi
    if [[ -z "${BROWSERLESS_DOCKER_CAPS:-}" ]]; then
        readonly BROWSERLESS_DOCKER_CAPS="SYS_ADMIN"
    fi
    if [[ -z "${BROWSERLESS_DOCKER_SECCOMP:-}" ]]; then
        readonly BROWSERLESS_DOCKER_SECCOMP="unconfined"
    fi

    # Export for global access
    export BROWSERLESS_PORT BROWSERLESS_BASE_URL BROWSERLESS_CONTAINER_NAME
    export BROWSERLESS_DATA_DIR BROWSERLESS_IMAGE BROWSERLESS_NETWORK_NAME
    export BROWSERLESS_MAX_BROWSERS BROWSERLESS_TIMEOUT BROWSERLESS_HEADLESS
    export BROWSERLESS_HEALTH_CHECK_INTERVAL BROWSERLESS_HEALTH_CHECK_MAX_ATTEMPTS
    export BROWSERLESS_API_TIMEOUT BROWSERLESS_STARTUP_MAX_WAIT
    export BROWSERLESS_STARTUP_WAIT_INTERVAL BROWSERLESS_INITIALIZATION_WAIT
    export BROWSERLESS_DOCKER_SHM_SIZE BROWSERLESS_DOCKER_CAPS BROWSERLESS_DOCKER_SECCOMP
}