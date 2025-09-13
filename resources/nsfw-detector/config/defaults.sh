#!/usr/bin/env bash
# Default configuration for NSFW Detector resource

# Port allocation (from registry)
get_port_for() {
    local service="$1"
    local port_registry="${VROOLI_ROOT:-${HOME}/Vrooli}/scripts/resources/port_registry.sh"
    if [[ -f "$port_registry" ]]; then
        source "$port_registry"
        echo "${RESOURCE_PORTS[$service]:-}"
    fi
}

# Service configuration
export NSFW_DETECTOR_PORT="${NSFW_DETECTOR_PORT:-$(get_port_for "nsfw-detector")}"
# No fallback - port must come from environment or registry

# Model configuration
export NSFW_DETECTOR_DEFAULT_MODEL="${NSFW_DETECTOR_DEFAULT_MODEL:-nsfwjs}"
export NSFW_DETECTOR_MODEL_PATH="${NSFW_DETECTOR_MODEL_PATH:-${HOME}/.local/share/nsfw-detector/models}"
export NSFW_DETECTOR_MODEL_CACHE_SIZE="${NSFW_DETECTOR_MODEL_CACHE_SIZE:-3}"

# Classification thresholds (0.0 - 1.0)
export NSFW_DETECTOR_ADULT_THRESHOLD="${NSFW_DETECTOR_ADULT_THRESHOLD:-0.7}"
export NSFW_DETECTOR_RACY_THRESHOLD="${NSFW_DETECTOR_RACY_THRESHOLD:-0.6}"
export NSFW_DETECTOR_GORE_THRESHOLD="${NSFW_DETECTOR_GORE_THRESHOLD:-0.8}"
export NSFW_DETECTOR_VIOLENCE_THRESHOLD="${NSFW_DETECTOR_VIOLENCE_THRESHOLD:-0.8}"
export NSFW_DETECTOR_SAFE_THRESHOLD="${NSFW_DETECTOR_SAFE_THRESHOLD:-0.5}"

# Performance settings
export NSFW_DETECTOR_MAX_BATCH_SIZE="${NSFW_DETECTOR_MAX_BATCH_SIZE:-10}"
export NSFW_DETECTOR_TIMEOUT_MS="${NSFW_DETECTOR_TIMEOUT_MS:-5000}"
export NSFW_DETECTOR_MAX_IMAGE_SIZE="${NSFW_DETECTOR_MAX_IMAGE_SIZE:-10485760}"  # 10MB
export NSFW_DETECTOR_CACHE_SIZE="${NSFW_DETECTOR_CACHE_SIZE:-100}"
export NSFW_DETECTOR_WORKER_THREADS="${NSFW_DETECTOR_WORKER_THREADS:-4}"

# Logging configuration
export NSFW_DETECTOR_LOG_LEVEL="${NSFW_DETECTOR_LOG_LEVEL:-info}"
export NSFW_DETECTOR_LOG_DIR="${NSFW_DETECTOR_LOG_DIR:-${HOME}/.local/share/nsfw-detector/logs}"
export NSFW_DETECTOR_AUDIT_LOG="${NSFW_DETECTOR_AUDIT_LOG:-false}"

# Security settings
export NSFW_DETECTOR_ENCRYPT_MODELS="${NSFW_DETECTOR_ENCRYPT_MODELS:-false}"
export NSFW_DETECTOR_ALLOWED_ORIGINS="${NSFW_DETECTOR_ALLOWED_ORIGINS:-*}"
export NSFW_DETECTOR_RATE_LIMIT="${NSFW_DETECTOR_RATE_LIMIT:-100}"  # Requests per minute

# Data paths
export NSFW_DETECTOR_DATA_DIR="${NSFW_DETECTOR_DATA_DIR:-${HOME}/.local/share/nsfw-detector}"
export NSFW_DETECTOR_TEMP_DIR="${NSFW_DETECTOR_TEMP_DIR:-/tmp/nsfw-detector}"