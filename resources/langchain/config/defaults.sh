#!/usr/bin/env bash
# LangChain Resource Configuration
# Framework for developing applications with LLMs

# Source standard variables
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
LANGCHAIN_CONFIG_DIR="${APP_ROOT}/resources/langchain/config"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"

# Resource identity
export LANGCHAIN_RESOURCE_NAME="langchain"
export LANGCHAIN_RESOURCE_CATEGORY="execution"
export LANGCHAIN_RESOURCE_DESCRIPTION="Framework for developing applications with LLMs"

# Service configuration
export LANGCHAIN_CONTAINER_NAME="${LANGCHAIN_CONTAINER_NAME:-langchain}"
export LANGCHAIN_PORT="${LANGCHAIN_PORT:-8100}"
export LANGCHAIN_API_PORT="${LANGCHAIN_API_PORT:-8101}"
export LANGCHAIN_IMAGE="${LANGCHAIN_IMAGE:-python:3.11-slim}"

# API Configuration
export LANGCHAIN_BASE_URL="${LANGCHAIN_BASE_URL:-http://localhost:${LANGCHAIN_PORT}}"
export LANGCHAIN_API_URL="${LANGCHAIN_API_URL:-http://localhost:${LANGCHAIN_API_PORT}}"

# Data directories
export LANGCHAIN_DATA_DIR="${LANGCHAIN_DATA_DIR:-${HOME}/.langchain}"
export LANGCHAIN_WORKSPACE_DIR="${LANGCHAIN_WORKSPACE_DIR:-${LANGCHAIN_DATA_DIR}/workspace}"
export LANGCHAIN_CHAINS_DIR="${LANGCHAIN_CHAINS_DIR:-${LANGCHAIN_DATA_DIR}/chains}"
export LANGCHAIN_AGENTS_DIR="${LANGCHAIN_AGENTS_DIR:-${LANGCHAIN_DATA_DIR}/agents}"

# Python environment
export LANGCHAIN_VENV_DIR="${LANGCHAIN_VENV_DIR:-${LANGCHAIN_DATA_DIR}/venv}"
export LANGCHAIN_PYTHON_VERSION="${LANGCHAIN_PYTHON_VERSION:-3.11}"

# Framework packages
export LANGCHAIN_CORE_PACKAGES="${LANGCHAIN_CORE_PACKAGES:-langchain langchain-community langchain-experimental}"
export LANGCHAIN_INTEGRATION_PACKAGES="${LANGCHAIN_INTEGRATION_PACKAGES:-langchain-openai langchain-anthropic}"

# Integration settings
export LANGCHAIN_ENABLE_OLLAMA="${LANGCHAIN_ENABLE_OLLAMA:-true}"
export LANGCHAIN_ENABLE_OPENROUTER="${LANGCHAIN_ENABLE_OPENROUTER:-true}"
export LANGCHAIN_ENABLE_VECTORDB="${LANGCHAIN_ENABLE_VECTORDB:-true}"

# Memory and caching
export LANGCHAIN_MEMORY_TYPE="${LANGCHAIN_MEMORY_TYPE:-conversation_buffer}"
export LANGCHAIN_CACHE_TYPE="${LANGCHAIN_CACHE_TYPE:-redis}"
export LANGCHAIN_REDIS_URL="${LANGCHAIN_REDIS_URL:-redis://localhost:6380}"

# Resource limits
export LANGCHAIN_MAX_MEMORY="${LANGCHAIN_MAX_MEMORY:-2g}"
export LANGCHAIN_MAX_CPU="${LANGCHAIN_MAX_CPU:-2}"

# Export configuration
langchain::export_config() {
    # Export all LANGCHAIN_ variables
    export | grep "^declare -x LANGCHAIN_" | cut -d' ' -f3- | while IFS='=' read -r name value; do
        export "$name=$value"
    done
}

# Export on source
langchain::export_config