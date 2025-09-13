#!/usr/bin/env bash
# Ollama Resource Environment Exports v2.0
# Self-contained exports without circular dependencies

# Source required utilities
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
export OLLAMA_CONFIG_DIR="${APP_ROOT}/resources/ollama/config"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"

# Source secrets management (from scripts/lib, not resource lib)
if ! declare -f secrets::resolve &>/dev/null; then
    source "${APP_ROOT}/scripts/lib/service/secrets.sh"
fi

# Helper for debug output
_ollama_debug() {
    [[ "${DEBUG:-false}" == "true" ]] && echo "[ollama/exports] $*" >&2 || true
}

# Resource metadata
export OLLAMA_RESOURCE_VERSION="2.0.0"
export OLLAMA_RESOURCE_NAME="ollama"
export OLLAMA_RESOURCE_DIR="${OLLAMA_RESOURCE_DIR:-${var_SCRIPTS_RESOURCES_DIR}/ai/ollama}"
export OLLAMA_RESOURCE_DIR="${OLLAMA_RESOURCE_DIR:-${APP_ROOT}/resources/ollama}"

# Get port from registry (if function is available)
if declare -f secrets::source_port_registry &>/dev/null; then
    secrets::source_port_registry
fi
export OLLAMA_PORT="${OLLAMA_PORT:-${RESOURCE_PORTS[ollama]:-11434}}"

# Ollama service configuration
export OLLAMA_HOST="${OLLAMA_HOST:-localhost}"
export OLLAMA_URL="http://${OLLAMA_HOST}:${OLLAMA_PORT}"
export OLLAMA_BASE_URL="$OLLAMA_URL"

# Ollama service details
export OLLAMA_SERVICE_NAME="ollama"
export OLLAMA_INSTALL_DIR="/usr/local/bin"
export OLLAMA_USER="ollama"

# Ollama performance configuration
export OLLAMA_NUM_PARALLEL="${OLLAMA_NUM_PARALLEL:-16}"
export OLLAMA_MAX_LOADED_MODELS="${OLLAMA_MAX_LOADED_MODELS:-3}"
export OLLAMA_FLASH_ATTENTION="${OLLAMA_FLASH_ATTENTION:-1}"
export OLLAMA_ORIGINS="${OLLAMA_ORIGINS:-*}"

# Health check command
export OLLAMA_HEALTH_CHECK="curl -s ${OLLAMA_URL}/api/tags"

# API endpoints
export OLLAMA_API_TAGS="${OLLAMA_URL}/api/tags"
export OLLAMA_API_CHAT="${OLLAMA_URL}/api/chat"
export OLLAMA_API_GENERATE="${OLLAMA_URL}/api/generate"
export OLLAMA_API_PULL="${OLLAMA_URL}/api/pull"
export OLLAMA_API_SHOW="${OLLAMA_URL}/api/show"

_ollama_debug "Ollama exports configured: URL=${OLLAMA_URL}, PORT=${OLLAMA_PORT}"