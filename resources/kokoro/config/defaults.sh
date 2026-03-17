#!/usr/bin/env bash
# Kokoro Configuration Defaults
# All configuration constants and default values

#######################################
# Export configuration constants
# Idempotent - safe to call multiple times
#######################################
defaults::export_config() {
    # Service configuration (only set if not already defined)
    if [[ -z "${KOKORO_PORT:-}" ]]; then
        readonly KOKORO_PORT="${KOKORO_CUSTOM_PORT:-8880}"
    fi
    if [[ -z "${KOKORO_BASE_URL:-}" ]]; then
        readonly KOKORO_BASE_URL="http://localhost:${KOKORO_PORT}"
    fi
    if [[ -z "${KOKORO_CONTAINER_NAME:-}" ]]; then
        readonly KOKORO_CONTAINER_NAME="kokoro"
    fi
    if [[ -z "${KOKORO_DATA_DIR:-}" ]]; then
        readonly KOKORO_DATA_DIR="${HOME}/.kokoro"
    fi
    if [[ -z "${KOKORO_VOICES_DIR:-}" ]]; then
        readonly KOKORO_VOICES_DIR="${KOKORO_DATA_DIR}/voices"
    fi
    if [[ -z "${KOKORO_API_TIMEOUT:-}" ]]; then
        readonly KOKORO_API_TIMEOUT="10"
    fi

    # Docker image configuration (only set if not already defined)
    if [[ -z "${KOKORO_IMAGE:-}" ]]; then
        readonly KOKORO_IMAGE="${KOKORO_IMAGE:-ghcr.io/remsky/kokoro-fastapi-gpu:latest}"
    fi
    if [[ -z "${KOKORO_CPU_IMAGE:-}" ]]; then
        readonly KOKORO_CPU_IMAGE="${KOKORO_CPU_IMAGE:-ghcr.io/remsky/kokoro-fastapi-cpu:latest}"
    fi

    # Voice configuration (only set if not already defined)
    if [[ -z "${KOKORO_DEFAULT_VOICE:-}" ]]; then
        readonly KOKORO_DEFAULT_VOICE="${KOKORO_DEFAULT_VOICE:-af_heart}"
    fi

    # Health check configuration (only set if not already defined)
    if [[ -z "${KOKORO_HEALTH_CHECK_INTERVAL:-}" ]]; then
        readonly KOKORO_HEALTH_CHECK_INTERVAL=5
    fi
    if [[ -z "${KOKORO_HEALTH_CHECK_MAX_ATTEMPTS:-}" ]]; then
        readonly KOKORO_HEALTH_CHECK_MAX_ATTEMPTS=12
    fi

    # Wait timeouts (only set if not already defined)
    if [[ -z "${KOKORO_STARTUP_MAX_WAIT:-}" ]]; then
        readonly KOKORO_STARTUP_MAX_WAIT=120  # Kokoro takes time to load the model on startup
    fi
    if [[ -z "${KOKORO_STARTUP_WAIT_INTERVAL:-}" ]]; then
        readonly KOKORO_STARTUP_WAIT_INTERVAL=5
    fi
    if [[ -z "${KOKORO_INITIALIZATION_WAIT:-}" ]]; then
        readonly KOKORO_INITIALIZATION_WAIT=30  # Model loading can take time
    fi

    # GPU configuration (only set if not already defined)
    if [[ -z "${KOKORO_GPU_ENABLED:-}" ]]; then
        # Auto-detect: nvidia-smi present + functional + Docker nvidia runtime.
        # Uses command -v (not system::is_command) because utility libs aren't
        # sourced yet when defaults.sh loads.
        if command -v nvidia-smi >/dev/null 2>&1 \
            && nvidia-smi >/dev/null 2>&1 \
            && docker info 2>/dev/null | grep -q nvidia; then
            readonly KOKORO_GPU_ENABLED="yes"
        else
            readonly KOKORO_GPU_ENABLED="no"
        fi
    fi

    # Export for global access
    export KOKORO_PORT KOKORO_BASE_URL KOKORO_CONTAINER_NAME
    export KOKORO_DATA_DIR KOKORO_VOICES_DIR
    export KOKORO_IMAGE KOKORO_CPU_IMAGE KOKORO_DEFAULT_VOICE
    export KOKORO_HEALTH_CHECK_INTERVAL KOKORO_HEALTH_CHECK_MAX_ATTEMPTS
    export KOKORO_API_TIMEOUT KOKORO_STARTUP_MAX_WAIT
    export KOKORO_STARTUP_WAIT_INTERVAL KOKORO_INITIALIZATION_WAIT
    export KOKORO_GPU_ENABLED
}

# Export function for subshell availability
export -f defaults::export_config
