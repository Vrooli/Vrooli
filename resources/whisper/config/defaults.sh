#!/usr/bin/env bash
# Whisper Configuration Defaults
# All configuration constants and default values

#######################################
# Export configuration constants
# Idempotent - safe to call multiple times
#######################################
defaults::export_config() {
    # Service configuration (only set if not already defined)
    if [[ -z "${WHISPER_PORT:-}" ]]; then
        readonly WHISPER_PORT="${WHISPER_CUSTOM_PORT:-8090}"
    fi
    if [[ -z "${WHISPER_BASE_URL:-}" ]]; then
        readonly WHISPER_BASE_URL="http://localhost:${WHISPER_PORT}"
    fi
    if [[ -z "${WHISPER_CONTAINER_NAME:-}" ]]; then
        readonly WHISPER_CONTAINER_NAME="whisper"
    fi
    if [[ -z "${WHISPER_DATA_DIR:-}" ]]; then
        readonly WHISPER_DATA_DIR="${HOME}/.whisper"
    fi
    if [[ -z "${WHISPER_MODELS_DIR:-}" ]]; then
        readonly WHISPER_MODELS_DIR="${WHISPER_DATA_DIR}/models"
    fi
    if [[ -z "${WHISPER_UPLOADS_DIR:-}" ]]; then
        readonly WHISPER_UPLOADS_DIR="${WHISPER_DATA_DIR}/uploads"
    fi
    if [[ -z "${WHISPER_API_TIMEOUT:-}" ]]; then
        readonly WHISPER_API_TIMEOUT="10"
    fi

    # Docker image configuration (only set if not already defined)
    if [[ -z "${WHISPER_IMAGE:-}" ]]; then
        readonly WHISPER_IMAGE="${WHISPER_IMAGE:-onerahmet/openai-whisper-asr-webservice:latest-gpu}"
    fi
    if [[ -z "${WHISPER_CPU_IMAGE:-}" ]]; then
        readonly WHISPER_CPU_IMAGE="${WHISPER_CPU_IMAGE:-onerahmet/openai-whisper-asr-webservice:latest}"
    fi

    # Model configuration (only set if not already defined)
    if [[ -z "${WHISPER_DEFAULT_MODEL:-}" ]]; then
        readonly WHISPER_DEFAULT_MODEL="${WHISPER_DEFAULT_MODEL:-large}"
    fi
    if [[ -z "${WHISPER_MODEL_SIZES:-}" ]]; then
        readonly WHISPER_MODEL_SIZES=("tiny" "base" "small" "medium" "large" "large-v2" "large-v3")
    fi

    # Model size information (only set if not already defined) - approximate in GB
    if [[ -z "${WHISPER_MODEL_SIZE_TINY:-}" ]]; then
        readonly WHISPER_MODEL_SIZE_TINY="0.04"
    fi
    if [[ -z "${WHISPER_MODEL_SIZE_BASE:-}" ]]; then
        readonly WHISPER_MODEL_SIZE_BASE="0.07"
    fi
    if [[ -z "${WHISPER_MODEL_SIZE_SMALL:-}" ]]; then
        readonly WHISPER_MODEL_SIZE_SMALL="0.24"
    fi
    if [[ -z "${WHISPER_MODEL_SIZE_MEDIUM:-}" ]]; then
        readonly WHISPER_MODEL_SIZE_MEDIUM="0.77"
    fi
    if [[ -z "${WHISPER_MODEL_SIZE_LARGE:-}" ]]; then
        readonly WHISPER_MODEL_SIZE_LARGE="1.55"
    fi

    # Health check configuration (only set if not already defined)
    if [[ -z "${WHISPER_HEALTH_CHECK_INTERVAL:-}" ]]; then
        readonly WHISPER_HEALTH_CHECK_INTERVAL=5
    fi
    if [[ -z "${WHISPER_HEALTH_CHECK_MAX_ATTEMPTS:-}" ]]; then
        readonly WHISPER_HEALTH_CHECK_MAX_ATTEMPTS=12
    fi
    if [[ -z "${WHISPER_API_TIMEOUT:-}" ]]; then
        readonly WHISPER_API_TIMEOUT=10
    fi

    # Wait timeouts (only set if not already defined)
    if [[ -z "${WHISPER_STARTUP_MAX_WAIT:-}" ]]; then
        readonly WHISPER_STARTUP_MAX_WAIT=120  # Whisper takes longer to start due to model loading
    fi
    if [[ -z "${WHISPER_STARTUP_WAIT_INTERVAL:-}" ]]; then
        readonly WHISPER_STARTUP_WAIT_INTERVAL=5
    fi
    if [[ -z "${WHISPER_INITIALIZATION_WAIT:-}" ]]; then
        readonly WHISPER_INITIALIZATION_WAIT=30  # Model loading can take time
    fi

    # GPU configuration (only set if not already defined)
    if [[ -z "${WHISPER_GPU_ENABLED:-}" ]]; then
        readonly WHISPER_GPU_ENABLED="${GPU:-no}"
    fi

    # Export for global access
    export WHISPER_PORT WHISPER_BASE_URL WHISPER_CONTAINER_NAME
    export WHISPER_DATA_DIR WHISPER_MODELS_DIR WHISPER_UPLOADS_DIR
    export WHISPER_IMAGE WHISPER_CPU_IMAGE WHISPER_DEFAULT_MODEL
    export WHISPER_HEALTH_CHECK_INTERVAL WHISPER_HEALTH_CHECK_MAX_ATTEMPTS
    export WHISPER_API_TIMEOUT WHISPER_STARTUP_MAX_WAIT
    export WHISPER_STARTUP_WAIT_INTERVAL WHISPER_INITIALIZATION_WAIT
    export WHISPER_GPU_ENABLED WHISPER_MODEL_SIZE_TINY WHISPER_MODEL_SIZE_BASE
    export WHISPER_MODEL_SIZE_SMALL WHISPER_MODEL_SIZE_MEDIUM WHISPER_MODEL_SIZE_LARGE
}

# Export function for subshell availability
export -f defaults::export_config