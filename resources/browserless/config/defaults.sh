#!/usr/bin/env bash
# Browserless Configuration Defaults
# All configuration constants and default values

# Get the directory of this script
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"

# Source required dependencies
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"

#######################################
# Export configuration constants
# Idempotent - safe to call multiple times
#######################################
browserless::export_config() {
    # Guard against re-initialization to prevent readonly conflicts
    [[ -n "${BROWSERLESS_CONFIG_INITIALIZED:-}" ]] && return 0
    
    # Service configuration (only set if not already defined)
    if [[ -z "${BROWSERLESS_PORT:-}" ]]; then
        BROWSERLESS_PORT="${BROWSERLESS_CUSTOM_PORT:-$(resources::get_default_port "browserless")}"
        readonly BROWSERLESS_PORT
        export BROWSERLESS_PORT
    fi
    if [[ -z "${BROWSERLESS_BASE_URL:-}" ]]; then
        BROWSERLESS_BASE_URL="http://localhost:${BROWSERLESS_PORT}"
        readonly BROWSERLESS_BASE_URL
        export BROWSERLESS_BASE_URL
    fi
    if [[ -z "${BROWSERLESS_CONTAINER_NAME:-}" ]]; then
        BROWSERLESS_CONTAINER_NAME="vrooli-browserless"
        readonly BROWSERLESS_CONTAINER_NAME
        export BROWSERLESS_CONTAINER_NAME
    fi
    if [[ -z "${BROWSERLESS_DATA_DIR:-}" ]]; then
        BROWSERLESS_DATA_DIR="${HOME}/.browserless"
        readonly BROWSERLESS_DATA_DIR
        export BROWSERLESS_DATA_DIR
    fi
    if [[ -z "${BROWSERLESS_IMAGE:-}" ]]; then
        BROWSERLESS_IMAGE="ghcr.io/browserless/chrome:latest"
        readonly BROWSERLESS_IMAGE
        export BROWSERLESS_IMAGE
    fi

    # Browser configuration (only set if not already defined)
    if [[ -z "${BROWSERLESS_MAX_BROWSERS:-}" ]]; then
        BROWSERLESS_MAX_BROWSERS="${MAX_BROWSERS:-5}"
        readonly BROWSERLESS_MAX_BROWSERS
        export BROWSERLESS_MAX_BROWSERS
    fi
    if [[ -z "${BROWSERLESS_TIMEOUT:-}" ]]; then
        BROWSERLESS_TIMEOUT="${TIMEOUT:-30000}"
        readonly BROWSERLESS_TIMEOUT
        export BROWSERLESS_TIMEOUT
    fi
    if [[ -z "${BROWSERLESS_HEADLESS:-}" ]]; then
        BROWSERLESS_HEADLESS="${HEADLESS:-yes}"
        readonly BROWSERLESS_HEADLESS
        export BROWSERLESS_HEADLESS
    fi

    # Network configuration (only set if not already defined)
    if [[ -z "${BROWSERLESS_NETWORK_NAME:-}" ]]; then
        BROWSERLESS_NETWORK_NAME="browserless-network"
        readonly BROWSERLESS_NETWORK_NAME
        export BROWSERLESS_NETWORK_NAME
    fi

    # Health check configuration (only set if not already defined)
    if [[ -z "${BROWSERLESS_HEALTH_CHECK_INTERVAL:-}" ]]; then
        BROWSERLESS_HEALTH_CHECK_INTERVAL=5
        readonly BROWSERLESS_HEALTH_CHECK_INTERVAL
        export BROWSERLESS_HEALTH_CHECK_INTERVAL
    fi
    if [[ -z "${BROWSERLESS_HEALTH_CHECK_MAX_ATTEMPTS:-}" ]]; then
        BROWSERLESS_HEALTH_CHECK_MAX_ATTEMPTS=12
        readonly BROWSERLESS_HEALTH_CHECK_MAX_ATTEMPTS
        export BROWSERLESS_HEALTH_CHECK_MAX_ATTEMPTS
    fi
    if [[ -z "${BROWSERLESS_API_TIMEOUT:-}" ]]; then
        BROWSERLESS_API_TIMEOUT=10
        readonly BROWSERLESS_API_TIMEOUT
        export BROWSERLESS_API_TIMEOUT
    fi

    # Wait timeouts (only set if not already defined)
    if [[ -z "${BROWSERLESS_STARTUP_MAX_WAIT:-}" ]]; then
        BROWSERLESS_STARTUP_MAX_WAIT=60
        readonly BROWSERLESS_STARTUP_MAX_WAIT
        export BROWSERLESS_STARTUP_MAX_WAIT
    fi
    if [[ -z "${BROWSERLESS_STARTUP_WAIT_INTERVAL:-}" ]]; then
        BROWSERLESS_STARTUP_WAIT_INTERVAL=2
        readonly BROWSERLESS_STARTUP_WAIT_INTERVAL
        export BROWSERLESS_STARTUP_WAIT_INTERVAL
    fi
    if [[ -z "${BROWSERLESS_INITIALIZATION_WAIT:-}" ]]; then
        BROWSERLESS_INITIALIZATION_WAIT=10
        readonly BROWSERLESS_INITIALIZATION_WAIT
        export BROWSERLESS_INITIALIZATION_WAIT
    fi

    # Security settings for Docker (only set if not already defined)
    if [[ -z "${BROWSERLESS_DOCKER_SHM_SIZE:-}" ]]; then
        BROWSERLESS_DOCKER_SHM_SIZE="2gb"
        readonly BROWSERLESS_DOCKER_SHM_SIZE
        export BROWSERLESS_DOCKER_SHM_SIZE
    fi
    if [[ -z "${BROWSERLESS_DOCKER_CAPS:-}" ]]; then
        BROWSERLESS_DOCKER_CAPS="SYS_ADMIN"
        readonly BROWSERLESS_DOCKER_CAPS
        export BROWSERLESS_DOCKER_CAPS
    fi
    if [[ -z "${BROWSERLESS_DOCKER_SECCOMP:-}" ]]; then
        BROWSERLESS_DOCKER_SECCOMP="unconfined"
        readonly BROWSERLESS_DOCKER_SECCOMP
        export BROWSERLESS_DOCKER_SECCOMP
    fi

    # Mark configuration as initialized to prevent re-running
    readonly BROWSERLESS_CONFIG_INITIALIZED=1
    export BROWSERLESS_CONFIG_INITIALIZED
}

# Export function for subshell availability
export -f browserless::export_config