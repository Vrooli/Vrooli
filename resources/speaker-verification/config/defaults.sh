#!/usr/bin/env bash
# Speaker Verification Configuration Defaults
# All configuration constants and default values

#######################################
# Export configuration constants
# Idempotent - safe to call multiple times
#######################################
defaults::export_config() {
    # Service configuration (only set if not already defined)
    if [[ -z "${SPEAKER_VERIFICATION_PORT:-}" ]]; then
        readonly SPEAKER_VERIFICATION_PORT="${SPEAKER_VERIFICATION_CUSTOM_PORT:-11452}"
    fi
    if [[ -z "${SPEAKER_VERIFICATION_BASE_URL:-}" ]]; then
        readonly SPEAKER_VERIFICATION_BASE_URL="http://localhost:${SPEAKER_VERIFICATION_PORT}"
    fi
    if [[ -z "${SPEAKER_VERIFICATION_CONTAINER_NAME:-}" ]]; then
        readonly SPEAKER_VERIFICATION_CONTAINER_NAME="speaker-verification"
    fi
    if [[ -z "${SPEAKER_VERIFICATION_DATA_DIR:-}" ]]; then
        readonly SPEAKER_VERIFICATION_DATA_DIR="${HOME}/.local/share/vrooli/resources/speaker-verification"
    fi
    if [[ -z "${SPEAKER_VERIFICATION_PROFILES_DIR:-}" ]]; then
        readonly SPEAKER_VERIFICATION_PROFILES_DIR="${SPEAKER_VERIFICATION_DATA_DIR}/profiles"
    fi
    if [[ -z "${SPEAKER_VERIFICATION_MODELS_DIR:-}" ]]; then
        readonly SPEAKER_VERIFICATION_MODELS_DIR="${SPEAKER_VERIFICATION_DATA_DIR}/model-cache"
    fi
    if [[ -z "${SPEAKER_VERIFICATION_API_TIMEOUT:-}" ]]; then
        readonly SPEAKER_VERIFICATION_API_TIMEOUT="15"
    fi

    # Docker image / model configuration (only set if not already defined)
    if [[ -z "${SPEAKER_VERIFICATION_IMAGE:-}" ]]; then
        readonly SPEAKER_VERIFICATION_IMAGE="${SPEAKER_VERIFICATION_IMAGE:-vrooli/speaker-verification:latest}"
    fi
    if [[ -z "${SPEAKER_VERIFICATION_MODEL:-}" ]]; then
        readonly SPEAKER_VERIFICATION_MODEL="${SPEAKER_VERIFICATION_MODEL:-speechbrain/spkrec-ecapa-voxceleb}"
    fi

    # Model facts (fixed by the ECAPA-TDNN model)
    if [[ -z "${SPEAKER_VERIFICATION_EMBEDDING_DIM:-}" ]]; then
        readonly SPEAKER_VERIFICATION_EMBEDDING_DIM=192
    fi
    if [[ -z "${SPEAKER_VERIFICATION_SAMPLE_RATE:-}" ]]; then
        readonly SPEAKER_VERIFICATION_SAMPLE_RATE=16000
    fi

    # Health check configuration (only set if not already defined)
    if [[ -z "${SPEAKER_VERIFICATION_HEALTH_CHECK_INTERVAL:-}" ]]; then
        readonly SPEAKER_VERIFICATION_HEALTH_CHECK_INTERVAL=5
    fi
    if [[ -z "${SPEAKER_VERIFICATION_HEALTH_CHECK_MAX_ATTEMPTS:-}" ]]; then
        readonly SPEAKER_VERIFICATION_HEALTH_CHECK_MAX_ATTEMPTS=18
    fi

    # Wait timeouts (only set if not already defined)
    if [[ -z "${SPEAKER_VERIFICATION_STARTUP_MAX_WAIT:-}" ]]; then
        # First start downloads the ECAPA-TDNN weights; allow generous time.
        readonly SPEAKER_VERIFICATION_STARTUP_MAX_WAIT=180
    fi
    if [[ -z "${SPEAKER_VERIFICATION_STARTUP_WAIT_INTERVAL:-}" ]]; then
        readonly SPEAKER_VERIFICATION_STARTUP_WAIT_INTERVAL=5
    fi
    if [[ -z "${SPEAKER_VERIFICATION_INITIALIZATION_WAIT:-}" ]]; then
        readonly SPEAKER_VERIFICATION_INITIALIZATION_WAIT=20
    fi

    # GPU configuration (only set if not already defined)
    if [[ -z "${SPEAKER_VERIFICATION_GPU_ENABLED:-}" ]]; then
        # CPU is the default path. GPU is opt-in: enabled only when an NVIDIA
        # stack is present AND Docker exposes the nvidia runtime. Uses command -v
        # because utility libs aren't sourced when defaults.sh loads.
        if command -v nvidia-smi >/dev/null 2>&1 \
            && nvidia-smi >/dev/null 2>&1 \
            && docker info 2>/dev/null | grep -q nvidia; then
            readonly SPEAKER_VERIFICATION_GPU_ENABLED="yes"
        else
            readonly SPEAKER_VERIFICATION_GPU_ENABLED="no"
        fi
    fi

    # Export for global access
    export SPEAKER_VERIFICATION_PORT SPEAKER_VERIFICATION_BASE_URL SPEAKER_VERIFICATION_CONTAINER_NAME
    export SPEAKER_VERIFICATION_DATA_DIR SPEAKER_VERIFICATION_PROFILES_DIR SPEAKER_VERIFICATION_MODELS_DIR
    export SPEAKER_VERIFICATION_IMAGE SPEAKER_VERIFICATION_MODEL
    export SPEAKER_VERIFICATION_EMBEDDING_DIM SPEAKER_VERIFICATION_SAMPLE_RATE
    export SPEAKER_VERIFICATION_HEALTH_CHECK_INTERVAL SPEAKER_VERIFICATION_HEALTH_CHECK_MAX_ATTEMPTS
    export SPEAKER_VERIFICATION_API_TIMEOUT SPEAKER_VERIFICATION_STARTUP_MAX_WAIT
    export SPEAKER_VERIFICATION_STARTUP_WAIT_INTERVAL SPEAKER_VERIFICATION_INITIALIZATION_WAIT
    export SPEAKER_VERIFICATION_GPU_ENABLED
}

# Export function for subshell availability
export -f defaults::export_config
