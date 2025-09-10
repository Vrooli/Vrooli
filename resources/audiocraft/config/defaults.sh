#!/usr/bin/env bash
################################################################################
# AudioCraft Configuration Defaults
################################################################################

# Set APP_ROOT if not already set
export APP_ROOT="${APP_ROOT:-/home/matthalloran8/Vrooli}"

# AudioCraft directories
export AUDIOCRAFT_DATA_DIR="${AUDIOCRAFT_DATA_DIR:-${APP_ROOT}/data/resources/audiocraft}"
export AUDIOCRAFT_MODELS_DIR="${AUDIOCRAFT_MODELS_DIR:-${AUDIOCRAFT_DATA_DIR}/models}"
export AUDIOCRAFT_OUTPUT_DIR="${AUDIOCRAFT_OUTPUT_DIR:-${AUDIOCRAFT_DATA_DIR}/output}"
export AUDIOCRAFT_CONFIG_DIR="${AUDIOCRAFT_CONFIG_DIR:-${AUDIOCRAFT_DATA_DIR}/config}"

# AudioCraft network settings
export AUDIOCRAFT_PORT="${AUDIOCRAFT_PORT:-7862}"
export AUDIOCRAFT_HOST="${AUDIOCRAFT_HOST:-0.0.0.0}"

# Container settings
export AUDIOCRAFT_CONTAINER_NAME="${AUDIOCRAFT_CONTAINER_NAME:-vrooli-audiocraft}"
export AUDIOCRAFT_IMAGE_NAME="${AUDIOCRAFT_IMAGE_NAME:-vrooli/audiocraft:latest}"

# Model configuration
export AUDIOCRAFT_MODEL_SIZE="${AUDIOCRAFT_MODEL_SIZE:-medium}"  # small/medium/large
export AUDIOCRAFT_USE_GPU="${AUDIOCRAFT_USE_GPU:-auto}"  # auto/true/false
export AUDIOCRAFT_MAX_DURATION="${AUDIOCRAFT_MAX_DURATION:-120}"  # Maximum generation duration in seconds

# Performance settings
export AUDIOCRAFT_BATCH_SIZE="${AUDIOCRAFT_BATCH_SIZE:-1}"
export AUDIOCRAFT_NUM_WORKERS="${AUDIOCRAFT_NUM_WORKERS:-2}"
export AUDIOCRAFT_CACHE_SIZE="${AUDIOCRAFT_CACHE_SIZE:-100}"  # Number of cached generations

# Memory management
export AUDIOCRAFT_MAX_MEMORY="${AUDIOCRAFT_MAX_MEMORY:-8G}"
export AUDIOCRAFT_MEMORY_EFFICIENT="${AUDIOCRAFT_MEMORY_EFFICIENT:-false}"

# Logging
export AUDIOCRAFT_LOG_LEVEL="${AUDIOCRAFT_LOG_LEVEL:-INFO}"
export AUDIOCRAFT_LOG_FILE="${AUDIOCRAFT_LOG_FILE:-${AUDIOCRAFT_DATA_DIR}/audiocraft.log}"