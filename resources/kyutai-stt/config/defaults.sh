#!/usr/bin/env bash
# Kyutai STT Configuration Defaults
# All configuration constants and default values

#######################################
# Export configuration constants
# Idempotent - safe to call multiple times
#######################################
defaults::export_config() {
    # Service configuration (only set if not already defined)
    if [[ -z "${KYUTAI_STT_PORT:-}" ]]; then
        readonly KYUTAI_STT_PORT="${KYUTAI_STT_CUSTOM_PORT:-8094}"
    fi
    if [[ -z "${KYUTAI_STT_BASE_URL:-}" ]]; then
        readonly KYUTAI_STT_BASE_URL="http://localhost:${KYUTAI_STT_PORT}"
    fi
    if [[ -z "${KYUTAI_STT_WS_URL:-}" ]]; then
        readonly KYUTAI_STT_WS_URL="ws://localhost:${KYUTAI_STT_PORT}/v1/stream"
    fi
    if [[ -z "${KYUTAI_STT_CONTAINER_NAME:-}" ]]; then
        readonly KYUTAI_STT_CONTAINER_NAME="kyutai-stt"
    fi
    if [[ -z "${KYUTAI_STT_DATA_DIR:-}" ]]; then
        readonly KYUTAI_STT_DATA_DIR="${RESOURCE_DATA_DIR:-${XDG_DATA_HOME:-${HOME}/.local/share}/vrooli/resources/kyutai-stt}"
    fi
    if [[ -z "${KYUTAI_STT_MODELS_DIR:-}" ]]; then
        readonly KYUTAI_STT_MODELS_DIR="${KYUTAI_STT_DATA_DIR}/models"
    fi
    if [[ -z "${KYUTAI_STT_API_TIMEOUT:-}" ]]; then
        readonly KYUTAI_STT_API_TIMEOUT="10"
    fi

    # Docker image configuration (only set if not already defined).
    # Image is built locally from docker/Dockerfile via compose.
    if [[ -z "${KYUTAI_STT_IMAGE:-}" ]]; then
        readonly KYUTAI_STT_IMAGE="${KYUTAI_STT_IMAGE:-vrooli/kyutai-stt:latest}"
    fi

    # Model configuration (only set if not already defined).
    # The 1B en_fr model fits the local RTX 4070 Ti SUPER VRAM budget.
    if [[ -z "${KYUTAI_STT_HF_REPO:-}" ]]; then
        readonly KYUTAI_STT_HF_REPO="${KYUTAI_STT_HF_REPO:-kyutai/stt-1b-en_fr}"
    fi

    # Device configuration (only set if not already defined)
    if [[ -z "${KYUTAI_STT_DEVICE:-}" ]]; then
        readonly KYUTAI_STT_DEVICE="${KYUTAI_STT_DEVICE:-cuda}"
    fi

    # Optional HF token (dev default empty; public models do not require it)
    if [[ -z "${KYUTAI_STT_HF_TOKEN:-}" ]]; then
        readonly KYUTAI_STT_HF_TOKEN="${KYUTAI_STT_HF_TOKEN:-}"
    fi

    # Health check configuration (only set if not already defined)
    if [[ -z "${KYUTAI_STT_HEALTH_CHECK_INTERVAL:-}" ]]; then
        readonly KYUTAI_STT_HEALTH_CHECK_INTERVAL=5
    fi
    if [[ -z "${KYUTAI_STT_HEALTH_CHECK_MAX_ATTEMPTS:-}" ]]; then
        readonly KYUTAI_STT_HEALTH_CHECK_MAX_ATTEMPTS=12
    fi

    # Wait timeouts (only set if not already defined).
    # First run downloads multi-GB weights, so allow a generous window.
    if [[ -z "${KYUTAI_STT_STARTUP_MAX_WAIT:-}" ]]; then
        readonly KYUTAI_STT_STARTUP_MAX_WAIT=600
    fi
    if [[ -z "${KYUTAI_STT_STARTUP_WAIT_INTERVAL:-}" ]]; then
        readonly KYUTAI_STT_STARTUP_WAIT_INTERVAL=5
    fi
    if [[ -z "${KYUTAI_STT_INITIALIZATION_WAIT:-}" ]]; then
        readonly KYUTAI_STT_INITIALIZATION_WAIT=30
    fi

    # GPU configuration (only set if not already defined)
    if [[ -z "${KYUTAI_STT_GPU_ENABLED:-}" ]]; then
        # Auto-detect: nvidia-smi present + functional + Docker nvidia runtime.
        # Uses command -v (not system::is_command) because utility libs aren't
        # sourced yet when defaults.sh loads.
        if command -v nvidia-smi >/dev/null 2>&1 \
            && nvidia-smi >/dev/null 2>&1 \
            && docker info 2>/dev/null | grep -q nvidia; then
            readonly KYUTAI_STT_GPU_ENABLED="yes"
        else
            readonly KYUTAI_STT_GPU_ENABLED="no"
        fi
    fi

    # Export for global access
    export KYUTAI_STT_PORT KYUTAI_STT_BASE_URL KYUTAI_STT_WS_URL KYUTAI_STT_CONTAINER_NAME
    export KYUTAI_STT_DATA_DIR KYUTAI_STT_MODELS_DIR
    export KYUTAI_STT_IMAGE KYUTAI_STT_HF_REPO KYUTAI_STT_DEVICE KYUTAI_STT_HF_TOKEN
    export KYUTAI_STT_HEALTH_CHECK_INTERVAL KYUTAI_STT_HEALTH_CHECK_MAX_ATTEMPTS
    export KYUTAI_STT_API_TIMEOUT KYUTAI_STT_STARTUP_MAX_WAIT
    export KYUTAI_STT_STARTUP_WAIT_INTERVAL KYUTAI_STT_INITIALIZATION_WAIT
    export KYUTAI_STT_GPU_ENABLED
}

# Export function for subshell availability
export -f defaults::export_config
