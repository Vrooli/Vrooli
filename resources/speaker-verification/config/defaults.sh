#!/usr/bin/env bash
# Speaker Verification Configuration Defaults
# All configuration constants and default values

# Default port constant for test reference
if [[ -z "${SPEAKER_VERIFICATION_DEFAULT_PORT:-}" ]]; then
    readonly SPEAKER_VERIFICATION_DEFAULT_PORT="8891"
fi

#######################################
# Export configuration constants
# Idempotent - safe to call multiple times
#######################################
speaker_verification::export_config() {
    # Guard against multiple calls
    if [[ -n "${SPEAKER_VERIFICATION_CONFIG_EXPORTED:-}" ]]; then
        return 0
    fi
    readonly SPEAKER_VERIFICATION_CONFIG_EXPORTED="true"

    # Service configuration
    if [[ -z "${SPEAKER_VERIFICATION_PORT:-}" ]]; then
        readonly SPEAKER_VERIFICATION_PORT="${SPEAKER_VERIFICATION_CUSTOM_PORT:-$SPEAKER_VERIFICATION_DEFAULT_PORT}"
    fi
    if [[ -z "${SPEAKER_VERIFICATION_BASE_URL:-}" ]]; then
        readonly SPEAKER_VERIFICATION_BASE_URL="http://localhost:${SPEAKER_VERIFICATION_PORT}"
    fi
    if [[ -z "${SPEAKER_VERIFICATION_CONTAINER_NAME:-}" ]]; then
        readonly SPEAKER_VERIFICATION_CONTAINER_NAME="speaker-verification"
    fi

    # Data directories
    if [[ -z "${SPEAKER_VERIFICATION_DATA_DIR:-}" ]]; then
        readonly SPEAKER_VERIFICATION_DATA_DIR="${HOME}/.speaker-verification"
    fi
    if [[ -z "${SPEAKER_VERIFICATION_PROFILES_DIR:-}" ]]; then
        readonly SPEAKER_VERIFICATION_PROFILES_DIR="${SPEAKER_VERIFICATION_DATA_DIR}/profiles"
    fi
    if [[ -z "${SPEAKER_VERIFICATION_CACHE_DIR:-}" ]]; then
        readonly SPEAKER_VERIFICATION_CACHE_DIR="${SPEAKER_VERIFICATION_DATA_DIR}/cache"
    fi
    if [[ -z "${SPEAKER_VERIFICATION_LOG_DIR:-}" ]]; then
        readonly SPEAKER_VERIFICATION_LOG_DIR="${SPEAKER_VERIFICATION_DATA_DIR}/logs"
    fi

    # Docker image
    if [[ -z "${SPEAKER_VERIFICATION_IMAGE:-}" ]]; then
        readonly SPEAKER_VERIFICATION_IMAGE="vrooli/speaker-verification:latest"
    fi

    # Device selection: auto|cpu|cuda
    if [[ -z "${SPEAKER_VERIFICATION_DEVICE:-}" ]]; then
        readonly SPEAKER_VERIFICATION_DEVICE="auto"
    fi

    # Model configuration
    if [[ -z "${SPEAKER_VERIFICATION_MODEL:-}" ]]; then
        readonly SPEAKER_VERIFICATION_MODEL="nvidia/speakerverification_en_titanet_large"
    fi

    # Verification thresholds
    if [[ -z "${SPEAKER_VERIFICATION_DEFAULT_THRESHOLD:-}" ]]; then
        readonly SPEAKER_VERIFICATION_DEFAULT_THRESHOLD="0.7"
    fi

    # Audio constraints
    if [[ -z "${SPEAKER_VERIFICATION_ENROLLMENT_MIN_SECONDS:-}" ]]; then
        readonly SPEAKER_VERIFICATION_ENROLLMENT_MIN_SECONDS="3"
    fi
    if [[ -z "${SPEAKER_VERIFICATION_VERIFY_MIN_SECONDS:-}" ]]; then
        readonly SPEAKER_VERIFICATION_VERIFY_MIN_SECONDS="1"
    fi
    if [[ -z "${SPEAKER_VERIFICATION_SAMPLE_RATE:-}" ]]; then
        readonly SPEAKER_VERIFICATION_SAMPLE_RATE="16000"
    fi

    # API settings
    if [[ -z "${SPEAKER_VERIFICATION_API_TIMEOUT:-}" ]]; then
        readonly SPEAKER_VERIFICATION_API_TIMEOUT="30"
    fi
    if [[ -z "${SPEAKER_VERIFICATION_MAX_UPLOAD_MB:-}" ]]; then
        readonly SPEAKER_VERIFICATION_MAX_UPLOAD_MB="50"
    fi

    # Network configuration
    if [[ -z "${SPEAKER_VERIFICATION_NETWORK_NAME:-}" ]]; then
        readonly SPEAKER_VERIFICATION_NETWORK_NAME="speaker-verification-network"
    fi

    # Health check configuration
    if [[ -z "${SPEAKER_VERIFICATION_HEALTH_CHECK_INTERVAL:-}" ]]; then
        readonly SPEAKER_VERIFICATION_HEALTH_CHECK_INTERVAL=5
    fi
    if [[ -z "${SPEAKER_VERIFICATION_HEALTH_CHECK_MAX_ATTEMPTS:-}" ]]; then
        readonly SPEAKER_VERIFICATION_HEALTH_CHECK_MAX_ATTEMPTS=24
    fi

    # Startup wait configuration
    if [[ -z "${SPEAKER_VERIFICATION_STARTUP_MAX_WAIT:-}" ]]; then
        readonly SPEAKER_VERIFICATION_STARTUP_MAX_WAIT=180
    fi
    if [[ -z "${SPEAKER_VERIFICATION_STARTUP_WAIT_INTERVAL:-}" ]]; then
        readonly SPEAKER_VERIFICATION_STARTUP_WAIT_INTERVAL=5
    fi
    if [[ -z "${SPEAKER_VERIFICATION_INITIALIZATION_WAIT:-}" ]]; then
        readonly SPEAKER_VERIFICATION_INITIALIZATION_WAIT=10
    fi

    # GPU detection
    if [[ -z "${SPEAKER_VERIFICATION_GPU_ENABLED:-}" ]]; then
        if command -v nvidia-smi &>/dev/null && nvidia-smi &>/dev/null; then
            if docker info 2>/dev/null | grep -qi nvidia; then
                readonly SPEAKER_VERIFICATION_GPU_ENABLED="true"
            else
                readonly SPEAKER_VERIFICATION_GPU_ENABLED="false"
            fi
        else
            readonly SPEAKER_VERIFICATION_GPU_ENABLED="false"
        fi
    fi

    # Export for global access
    export SPEAKER_VERIFICATION_DEFAULT_PORT SPEAKER_VERIFICATION_PORT SPEAKER_VERIFICATION_BASE_URL
    export SPEAKER_VERIFICATION_CONTAINER_NAME
    export SPEAKER_VERIFICATION_DATA_DIR SPEAKER_VERIFICATION_PROFILES_DIR
    export SPEAKER_VERIFICATION_CACHE_DIR SPEAKER_VERIFICATION_LOG_DIR
    export SPEAKER_VERIFICATION_IMAGE SPEAKER_VERIFICATION_DEVICE SPEAKER_VERIFICATION_MODEL
    export SPEAKER_VERIFICATION_DEFAULT_THRESHOLD
    export SPEAKER_VERIFICATION_ENROLLMENT_MIN_SECONDS SPEAKER_VERIFICATION_VERIFY_MIN_SECONDS
    export SPEAKER_VERIFICATION_SAMPLE_RATE
    export SPEAKER_VERIFICATION_API_TIMEOUT SPEAKER_VERIFICATION_MAX_UPLOAD_MB
    export SPEAKER_VERIFICATION_NETWORK_NAME
    export SPEAKER_VERIFICATION_HEALTH_CHECK_INTERVAL SPEAKER_VERIFICATION_HEALTH_CHECK_MAX_ATTEMPTS
    export SPEAKER_VERIFICATION_STARTUP_MAX_WAIT SPEAKER_VERIFICATION_STARTUP_WAIT_INTERVAL
    export SPEAKER_VERIFICATION_INITIALIZATION_WAIT SPEAKER_VERIFICATION_GPU_ENABLED
}
