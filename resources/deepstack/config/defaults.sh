#!/usr/bin/env bash
# DeepStack Resource - Default Configuration

# Service configuration
export DEEPSTACK_PORT="${DEEPSTACK_PORT:-11453}"
export DEEPSTACK_HOST="${DEEPSTACK_HOST:-127.0.0.1}"
export DEEPSTACK_API_KEY="${DEEPSTACK_API_KEY:-}"

# Docker configuration
export DEEPSTACK_IMAGE="${DEEPSTACK_IMAGE:-deepquestai/deepstack:latest}"
export DEEPSTACK_CONTAINER_NAME="${DEEPSTACK_CONTAINER_NAME:-vrooli-deepstack}"
export DEEPSTACK_NETWORK="${DEEPSTACK_NETWORK:-vrooli-network}"

# Model configuration
export DEEPSTACK_DETECTION_MODEL="${DEEPSTACK_DETECTION_MODEL:-yolov5}"
export DEEPSTACK_FACE_MODEL="${DEEPSTACK_FACE_MODEL:-retinaface}"
export DEEPSTACK_SCENE_MODEL="${DEEPSTACK_SCENE_MODEL:-places365}"
export DEEPSTACK_CONFIDENCE_THRESHOLD="${DEEPSTACK_CONFIDENCE_THRESHOLD:-0.45}"
export DEEPSTACK_MAX_SIZE="${DEEPSTACK_MAX_SIZE:-10485760}"  # 10MB

# Performance configuration
export DEEPSTACK_MODE="${DEEPSTACK_MODE:-High}"  # High, Medium, Low
export DEEPSTACK_ENABLE_GPU="${DEEPSTACK_ENABLE_GPU:-auto}"  # auto, true, false
export DEEPSTACK_THREADS="${DEEPSTACK_THREADS:-4}"
export DEEPSTACK_TIMEOUT="${DEEPSTACK_TIMEOUT:-5000}"  # milliseconds

# Storage configuration
export DEEPSTACK_DATA_DIR="${DEEPSTACK_DATA_DIR:-${HOME}/.vrooli/resources/deepstack/data}"
export DEEPSTACK_MODEL_DIR="${DEEPSTACK_MODEL_DIR:-${HOME}/.vrooli/resources/deepstack/models}"
export DEEPSTACK_TEMP_DIR="${DEEPSTACK_TEMP_DIR:-${HOME}/.vrooli/resources/deepstack/temp}"

# Redis configuration (optional)
export DEEPSTACK_REDIS_ENABLED="${DEEPSTACK_REDIS_ENABLED:-false}"
export DEEPSTACK_REDIS_HOST="${DEEPSTACK_REDIS_HOST:-127.0.0.1}"
export DEEPSTACK_REDIS_PORT="${DEEPSTACK_REDIS_PORT:-6380}"
export DEEPSTACK_REDIS_TTL="${DEEPSTACK_REDIS_TTL:-3600}"  # 1 hour cache

# Logging configuration
export DEEPSTACK_LOG_LEVEL="${DEEPSTACK_LOG_LEVEL:-INFO}"
export DEEPSTACK_LOG_FILE="${DEEPSTACK_LOG_FILE:-${HOME}/.vrooli/logs/deepstack.log}"

# Health check configuration
export DEEPSTACK_HEALTH_TIMEOUT="${DEEPSTACK_HEALTH_TIMEOUT:-5}"
export DEEPSTACK_HEALTH_RETRIES="${DEEPSTACK_HEALTH_RETRIES:-3}"
export DEEPSTACK_HEALTH_INTERVAL="${DEEPSTACK_HEALTH_INTERVAL:-10}"