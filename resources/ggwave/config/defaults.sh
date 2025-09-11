#!/bin/bash
# GGWave Resource Configuration Defaults

# Resource identification
export GGWAVE_RESOURCE_NAME="ggwave"
export GGWAVE_DISPLAY_NAME="GGWave Data-over-Sound"
export GGWAVE_VERSION="0.4.0"

# Port configuration - MUST use port registry
if [[ -f "${SCRIPTS_DIR:-/opt/vrooli/scripts}/resources/port_registry.sh" ]]; then
    source "${SCRIPTS_DIR:-/opt/vrooli/scripts}/resources/port_registry.sh"
    export GGWAVE_PORT=$(ports::get_port "ggwave")
else
    # Fallback for development only - production must use registry
    export GGWAVE_PORT="${GGWAVE_PORT:-8196}"
fi

# Docker configuration
export GGWAVE_IMAGE="${GGWAVE_IMAGE:-vrooli/ggwave:latest}"
export GGWAVE_CONTAINER_NAME="${GGWAVE_CONTAINER_NAME:-vrooli-ggwave}"
export GGWAVE_NETWORK="${GGWAVE_NETWORK:-vrooli-network}"

# Data persistence
export GGWAVE_DATA_DIR="${GGWAVE_DATA_DIR:-${HOME}/.vrooli/resources/ggwave}"
export GGWAVE_CONFIG_DIR="${GGWAVE_CONFIG_DIR:-${GGWAVE_DATA_DIR}/config}"
export GGWAVE_LOGS_DIR="${GGWAVE_LOGS_DIR:-${GGWAVE_DATA_DIR}/logs}"

# Service configuration
export GGWAVE_MODE="${GGWAVE_MODE:-auto}"  # auto|normal|fast|dt|ultrasonic
export GGWAVE_SAMPLE_RATE="${GGWAVE_SAMPLE_RATE:-48000}"  # Hz
export GGWAVE_VOLUME="${GGWAVE_VOLUME:-0.8}"  # 0.0-1.0
export GGWAVE_ERROR_CORRECTION="${GGWAVE_ERROR_CORRECTION:-true}"
export GGWAVE_LOG_LEVEL="${GGWAVE_LOG_LEVEL:-info}"  # debug|info|warn|error

# Performance settings
export GGWAVE_MAX_SESSIONS="${GGWAVE_MAX_SESSIONS:-10}"
export GGWAVE_SESSION_TIMEOUT="${GGWAVE_SESSION_TIMEOUT:-300}"  # seconds
export GGWAVE_BUFFER_SIZE="${GGWAVE_BUFFER_SIZE:-4096}"

# Health check configuration
export GGWAVE_HEALTH_CHECK_INTERVAL="${GGWAVE_HEALTH_CHECK_INTERVAL:-30}"
export GGWAVE_HEALTH_CHECK_TIMEOUT="${GGWAVE_HEALTH_CHECK_TIMEOUT:-5}"
export GGWAVE_STARTUP_TIMEOUT="${GGWAVE_STARTUP_TIMEOUT:-30}"

# Audio device configuration (for Docker passthrough)
export GGWAVE_AUDIO_DEVICE="${GGWAVE_AUDIO_DEVICE:-default}"
export GGWAVE_AUDIO_INPUT="${GGWAVE_AUDIO_INPUT:-/dev/snd}"
export GGWAVE_AUDIO_OUTPUT="${GGWAVE_AUDIO_OUTPUT:-/dev/snd}"

# Test configuration
export GGWAVE_TEST_MODE="${GGWAVE_TEST_MODE:-false}"
export GGWAVE_TEST_DATA="${GGWAVE_TEST_DATA:-Hello GGWave}"
export GGWAVE_TEST_ITERATIONS="${GGWAVE_TEST_ITERATIONS:-10}"

# Resource limits
export GGWAVE_MEMORY_LIMIT="${GGWAVE_MEMORY_LIMIT:-512m}"
export GGWAVE_CPU_LIMIT="${GGWAVE_CPU_LIMIT:-1.0}"

# Ensure directories exist
mkdir -p "${GGWAVE_DATA_DIR}" "${GGWAVE_CONFIG_DIR}" "${GGWAVE_LOGS_DIR}"