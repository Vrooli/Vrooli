#!/bin/bash

# Common variables and functions for SageMath resource

VROOLI_ROOT="${VROOLI_ROOT:-${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}}"
# shellcheck disable=SC1091
source "${VROOLI_ROOT}/resources/sagemath/config/defaults.sh"

# Cache dir is not set by the older defaults file in some callers; keep the canonical fallback here.
SAGEMATH_CACHE_DIR="${SAGEMATH_CACHE_DIR:-${RESOURCE_CACHE_DIR:-${XDG_CACHE_HOME:-${HOME}/.cache}/vrooli/resources/sagemath}}"

# Ensure data directories exist
sagemath_ensure_directories() {
    mkdir -p "$SAGEMATH_SCRIPTS_DIR"
    mkdir -p "$SAGEMATH_NOTEBOOKS_DIR"
    mkdir -p "$SAGEMATH_OUTPUTS_DIR"
    mkdir -p "$SAGEMATH_CONFIG_DIR"
    mkdir -p "$SAGEMATH_CACHE_DIR"
    mkdir -p "${SAGEMATH_LOGS_DIR:-}"
    mkdir -p "${SAGEMATH_STATE_DIR:-}"
}

# Check if container exists
sagemath_container_exists() {
    docker ps -a --format "{{.Names}}" | grep -q "^${SAGEMATH_CONTAINER_NAME}$"
}

# Check if container is running
sagemath_container_running() {
    docker ps --format "{{.Names}}" | grep -q "^${SAGEMATH_CONTAINER_NAME}$"
}

# Get container ID
sagemath_get_container_id() {
    docker ps -aq -f "name=${SAGEMATH_CONTAINER_NAME}"
}

# Format output based on requested format
sagemath_format_output() {
    local format="${1:-text}"
    local content="$2"
    
    if [[ "$format" == "json" ]]; then
        echo "$content"
    else
        echo "$content" | jq -r '.'
    fi
}
