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
    # Pin to a known digest to avoid silent image drift (Chrome 142 latest builds
    # have started pulling in DBus/thread requirements that crash in our env).
    # If you need to update, bump the digest explicitly.
    if [[ -z "${BROWSERLESS_IMAGE:-}" ]]; then
        BROWSERLESS_IMAGE="ghcr.io/browserless/chrome@sha256:96cc9039f44c8a7b277846783f18c1ec501a7f8b1b12bdfc2bc1f9c3f84a9a17"
        readonly BROWSERLESS_IMAGE
        export BROWSERLESS_IMAGE
    fi

    # Browser configuration (only set if not already defined)
    if [[ -z "${BROWSERLESS_MAX_BROWSERS:-}" ]]; then
        # Keep default concurrency low for CI/local to reduce Chrome launch pressure
        BROWSERLESS_MAX_BROWSERS="${MAX_BROWSERS:-1}"
        readonly BROWSERLESS_MAX_BROWSERS
        export BROWSERLESS_MAX_BROWSERS
    fi
    if [[ -z "${BROWSERLESS_MAX_CONCURRENT_SESSIONS:-}" ]]; then
        BROWSERLESS_MAX_CONCURRENT_SESSIONS="${BROWSERLESS_MAX_BROWSERS}"
        readonly BROWSERLESS_MAX_CONCURRENT_SESSIONS
        export BROWSERLESS_MAX_CONCURRENT_SESSIONS
    fi
    if [[ -z "${BROWSERLESS_PREBOOT_CHROME:-}" ]]; then
        # Preboot off by default; avoid spawning Chrome until needed
        BROWSERLESS_PREBOOT_CHROME="false"
        readonly BROWSERLESS_PREBOOT_CHROME
        export BROWSERLESS_PREBOOT_CHROME
    fi
    if [[ -z "${BROWSERLESS_KEEP_ALIVE:-}" ]]; then
        BROWSERLESS_KEEP_ALIVE="true"
        readonly BROWSERLESS_KEEP_ALIVE
        export BROWSERLESS_KEEP_ALIVE
    fi
    if [[ -z "${BROWSERLESS_MAX_QUEUE_LENGTH:-}" ]]; then
        BROWSERLESS_MAX_QUEUE_LENGTH="${MAX_QUEUE_LENGTH:-10}"
        readonly BROWSERLESS_MAX_QUEUE_LENGTH
        export BROWSERLESS_MAX_QUEUE_LENGTH
    fi
    if [[ -z "${BROWSERLESS_CONNECTION_TIMEOUT:-}" ]]; then
        BROWSERLESS_CONNECTION_TIMEOUT=20000
        readonly BROWSERLESS_CONNECTION_TIMEOUT
        export BROWSERLESS_CONNECTION_TIMEOUT
    fi
    if [[ -z "${BROWSERLESS_TIMEOUT:-}" ]]; then
        BROWSERLESS_TIMEOUT="${TIMEOUT:-60000}"
        readonly BROWSERLESS_TIMEOUT
        export BROWSERLESS_TIMEOUT
    fi
    if [[ -z "${BROWSERLESS_WORKER_TIMEOUT:-}" ]]; then
        BROWSERLESS_WORKER_TIMEOUT=120000
        readonly BROWSERLESS_WORKER_TIMEOUT
        export BROWSERLESS_WORKER_TIMEOUT
    fi
    if [[ -z "${BROWSERLESS_CHROME_REFRESH_MS:-}" ]]; then
        # Refresh browsers more frequently (10m) to prevent memory accumulation
        # Shorter refresh prevents long-running instances from consuming excessive memory
        BROWSERLESS_CHROME_REFRESH_MS=600000
        readonly BROWSERLESS_CHROME_REFRESH_MS
        export BROWSERLESS_CHROME_REFRESH_MS
    fi
    if [[ -z "${BROWSERLESS_SOCKET_CLOSE_TIMEOUT:-}" ]]; then
        BROWSERLESS_SOCKET_CLOSE_TIMEOUT=5000
        readonly BROWSERLESS_SOCKET_CLOSE_TIMEOUT
        export BROWSERLESS_SOCKET_CLOSE_TIMEOUT
    fi
    if [[ -z "${BROWSERLESS_ENABLE_PREWARM:-}" ]]; then
        # Disable prewarm to lower idle resource usage; enable explicitly if needed
        BROWSERLESS_ENABLE_PREWARM="false"
        readonly BROWSERLESS_ENABLE_PREWARM
        export BROWSERLESS_ENABLE_PREWARM
    fi
    if [[ -z "${BROWSERLESS_PREWARM_COUNT:-}" ]]; then
        BROWSERLESS_PREWARM_COUNT="0"
        readonly BROWSERLESS_PREWARM_COUNT
        export BROWSERLESS_PREWARM_COUNT
    fi
    if [[ -z "${BROWSERLESS_DEFAULT_LAUNCH_ARGS:-}" ]]; then
        # Harden Chrome startup in headless Docker (DBus-less, lower resource churn)
        BROWSERLESS_DEFAULT_LAUNCH_ARGS="--no-sandbox --disable-dev-shm-usage --disable-gpu --disable-software-rasterizer --disable-dev-tools --disable-features=TranslateUI --disable-extensions --disable-background-networking --no-first-run --mute-audio"
        readonly BROWSERLESS_DEFAULT_LAUNCH_ARGS
        export BROWSERLESS_DEFAULT_LAUNCH_ARGS
    fi
    if [[ -z "${BROWSERLESS_HEADLESS:-}" ]]; then
        BROWSERLESS_HEADLESS="${HEADLESS:-yes}"
        readonly BROWSERLESS_HEADLESS
        export BROWSERLESS_HEADLESS
    fi
    if [[ -z "${BROWSERLESS_BENCHMARK_ITERATIONS:-}" ]]; then
        BROWSERLESS_BENCHMARK_ITERATIONS=3
        readonly BROWSERLESS_BENCHMARK_ITERATIONS
        export BROWSERLESS_BENCHMARK_ITERATIONS
    fi
    if [[ -z "${BROWSERLESS_BENCHMARK_TIMEOUT:-}" ]]; then
        BROWSERLESS_BENCHMARK_TIMEOUT=45
        readonly BROWSERLESS_BENCHMARK_TIMEOUT
        export BROWSERLESS_BENCHMARK_TIMEOUT
    fi

    # Network configuration (only set if not already defined)
    if [[ -z "${BROWSERLESS_USE_HOST_NETWORK:-}" ]]; then
        BROWSERLESS_USE_HOST_NETWORK="${BROWSERLESS_HOST_NETWORKING:-yes}"
        readonly BROWSERLESS_USE_HOST_NETWORK
        export BROWSERLESS_USE_HOST_NETWORK
    fi
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
    if [[ -z "${BROWSERLESS_DOCKER_MEMORY:-}" ]]; then
        # Increased from 4g to 6g to handle concurrent test execution
        # With 8 max browsers at ~500MB each, 6g provides headroom for Chrome overhead
        BROWSERLESS_DOCKER_MEMORY="6g"
        readonly BROWSERLESS_DOCKER_MEMORY
        export BROWSERLESS_DOCKER_MEMORY
    fi
    if [[ -z "${BROWSERLESS_DOCKER_CPUS:-}" ]]; then
        BROWSERLESS_DOCKER_CPUS="4"
        readonly BROWSERLESS_DOCKER_CPUS
        export BROWSERLESS_DOCKER_CPUS
    fi
    if [[ -z "${BROWSERLESS_DOCKER_PIDS_LIMIT:-}" ]]; then
        BROWSERLESS_DOCKER_PIDS_LIMIT="8192"
        readonly BROWSERLESS_DOCKER_PIDS_LIMIT
        export BROWSERLESS_DOCKER_PIDS_LIMIT
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

    # Auto-scaling configuration (only set if not already defined)
    if [[ -z "${BROWSERLESS_ENABLE_AUTOSCALING:-}" ]]; then
        BROWSERLESS_ENABLE_AUTOSCALING="false"
        readonly BROWSERLESS_ENABLE_AUTOSCALING
        export BROWSERLESS_ENABLE_AUTOSCALING
    fi
    if [[ -z "${BROWSERLESS_POOL_MIN_SIZE:-}" ]]; then
        BROWSERLESS_POOL_MIN_SIZE="2"
        readonly BROWSERLESS_POOL_MIN_SIZE
        export BROWSERLESS_POOL_MIN_SIZE
    fi
    if [[ -z "${BROWSERLESS_POOL_MAX_SIZE:-}" ]]; then
        BROWSERLESS_POOL_MAX_SIZE="20"
        readonly BROWSERLESS_POOL_MAX_SIZE
        export BROWSERLESS_POOL_MAX_SIZE
    fi
    if [[ -z "${BROWSERLESS_POOL_SCALE_UP_THRESHOLD:-}" ]]; then
        BROWSERLESS_POOL_SCALE_UP_THRESHOLD="70"
        readonly BROWSERLESS_POOL_SCALE_UP_THRESHOLD
        export BROWSERLESS_POOL_SCALE_UP_THRESHOLD
    fi
    if [[ -z "${BROWSERLESS_POOL_SCALE_DOWN_THRESHOLD:-}" ]]; then
        BROWSERLESS_POOL_SCALE_DOWN_THRESHOLD="30"
        readonly BROWSERLESS_POOL_SCALE_DOWN_THRESHOLD
        export BROWSERLESS_POOL_SCALE_DOWN_THRESHOLD
    fi
    if [[ -z "${BROWSERLESS_POOL_SCALE_STEP:-}" ]]; then
        BROWSERLESS_POOL_SCALE_STEP="2"
        readonly BROWSERLESS_POOL_SCALE_STEP
        export BROWSERLESS_POOL_SCALE_STEP
    fi
    if [[ -z "${BROWSERLESS_POOL_MONITOR_INTERVAL:-}" ]]; then
        BROWSERLESS_POOL_MONITOR_INTERVAL="10"
        readonly BROWSERLESS_POOL_MONITOR_INTERVAL
        export BROWSERLESS_POOL_MONITOR_INTERVAL
    fi
    if [[ -z "${BROWSERLESS_POOL_COOLDOWN_PERIOD:-}" ]]; then
        BROWSERLESS_POOL_COOLDOWN_PERIOD="30"
        readonly BROWSERLESS_POOL_COOLDOWN_PERIOD
        export BROWSERLESS_POOL_COOLDOWN_PERIOD
    fi
    
    # Mark configuration as initialized to prevent re-running
    readonly BROWSERLESS_CONFIG_INITIALIZED=1
    export BROWSERLESS_CONFIG_INITIALIZED
}

# Export function for subshell availability
export -f browserless::export_config
