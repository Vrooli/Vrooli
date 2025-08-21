#!/bin/bash
# LiteLLM resource default configuration

# Basic resource information
[[ -z "${LITELLM_RESOURCE_NAME:-}" ]] && readonly LITELLM_RESOURCE_NAME="litellm"
[[ -z "${LITELLM_RESOURCE_CATEGORY:-}" ]] && readonly LITELLM_RESOURCE_CATEGORY="ai"
[[ -z "${LITELLM_CONTAINER_NAME:-}" ]] && readonly LITELLM_CONTAINER_NAME="vrooli-litellm"
[[ -z "${LITELLM_IMAGE:-}" ]] && readonly LITELLM_IMAGE="ghcr.io/berriai/litellm:main-latest"

# Network configuration
[[ -z "${LITELLM_NETWORK:-}" ]] && readonly LITELLM_NETWORK="vrooli-network"
[[ -z "${LITELLM_HOSTNAME:-}" ]] && readonly LITELLM_HOSTNAME="vrooli-litellm"

# Port configuration (from port registry)
[[ -z "${LITELLM_PORT:-}" ]] && readonly LITELLM_PORT="11435"
[[ -z "${LITELLM_INTERNAL_PORT:-}" ]] && readonly LITELLM_INTERNAL_PORT="4000"

# Paths and directories
[[ -z "${LITELLM_CONFIG_DIR:-}" ]] && readonly LITELLM_CONFIG_DIR="${var_ROOT_DIR}/data/litellm/config"
[[ -z "${LITELLM_LOG_DIR:-}" ]] && readonly LITELLM_LOG_DIR="${var_ROOT_DIR}/data/litellm/logs"
[[ -z "${LITELLM_DATA_DIR:-}" ]] && readonly LITELLM_DATA_DIR="${var_ROOT_DIR}/data/litellm/data"

# Configuration files
[[ -z "${LITELLM_CONFIG_FILE:-}" ]] && readonly LITELLM_CONFIG_FILE="${LITELLM_CONFIG_DIR}/config.yaml"
[[ -z "${LITELLM_ENV_FILE:-}" ]] && readonly LITELLM_ENV_FILE="${LITELLM_CONFIG_DIR}/.env"

# Default configuration
[[ -z "${LITELLM_DEFAULT_MODEL:-}" ]] && readonly LITELLM_DEFAULT_MODEL="gpt-3.5-turbo"
[[ -z "${LITELLM_DEFAULT_API_BASE:-}" ]] && readonly LITELLM_DEFAULT_API_BASE="http://localhost:${LITELLM_PORT}"

# Health check configuration
[[ -z "${LITELLM_HEALTH_CHECK_URL:-}" ]] && readonly LITELLM_HEALTH_CHECK_URL="http://localhost:${LITELLM_PORT}/health"
[[ -z "${LITELLM_HEALTH_CHECK_TIMEOUT:-}" ]] && readonly LITELLM_HEALTH_CHECK_TIMEOUT="10"
[[ -z "${LITELLM_STARTUP_TIMEOUT:-}" ]] && readonly LITELLM_STARTUP_TIMEOUT="60"

# API configuration
[[ -z "${LITELLM_API_BASE:-}" ]] && readonly LITELLM_API_BASE="http://localhost:${LITELLM_PORT}"
[[ -z "${LITELLM_API_VERSION:-}" ]] && readonly LITELLM_API_VERSION="v1"

# Timeouts
[[ -z "${LITELLM_TIMEOUT:-}" ]] && readonly LITELLM_TIMEOUT="30"
[[ -z "${LITELLM_INSTALL_TIMEOUT:-}" ]] && readonly LITELLM_INSTALL_TIMEOUT="300"

# Logging
[[ -z "${LITELLM_LOG_LEVEL:-}" ]] && readonly LITELLM_LOG_LEVEL="INFO"
[[ -z "${LITELLM_LOG_FILE:-}" ]] && readonly LITELLM_LOG_FILE="${LITELLM_LOG_DIR}/litellm.log"

# Security
if [[ -z "${LITELLM_MASTER_KEY:-}" ]]; then
    LITELLM_MASTER_KEY="sk-vrooli-litellm-$(openssl rand -hex 8)"
    readonly LITELLM_MASTER_KEY
fi

# Default provider configurations
[[ -z "${LITELLM_DEFAULT_PROVIDERS:-}" ]] && readonly LITELLM_DEFAULT_PROVIDERS='[
  {
    "model_name": "gpt-3.5-turbo",
    "litellm_params": {
      "model": "openai/gpt-3.5-turbo",
      "api_key": "PLACEHOLDER"
    }
  },
  {
    "model_name": "claude-3-haiku",
    "litellm_params": {
      "model": "anthropic/claude-3-haiku-20240307",
      "api_key": "PLACEHOLDER"
    }
  },
  {
    "model_name": "gemini-pro",
    "litellm_params": {
      "model": "gemini/gemini-pro",
      "api_key": "PLACEHOLDER"
    }
  }
]'

# Router configuration
[[ -z "${LITELLM_DEFAULT_ROUTING_STRATEGY:-}" ]] && readonly LITELLM_DEFAULT_ROUTING_STRATEGY="simple-shuffle"
[[ -z "${LITELLM_ENABLE_FALLBACKS:-}" ]] && readonly LITELLM_ENABLE_FALLBACKS="true"
[[ -z "${LITELLM_ENABLE_RETRIES:-}" ]] && readonly LITELLM_ENABLE_RETRIES="true"
[[ -z "${LITELLM_MAX_RETRIES:-}" ]] && readonly LITELLM_MAX_RETRIES="3"

# Budget and rate limiting
[[ -z "${LITELLM_ENABLE_BUDGET_CONTROL:-}" ]] && readonly LITELLM_ENABLE_BUDGET_CONTROL="true"
[[ -z "${LITELLM_DEFAULT_MAX_BUDGET:-}" ]] && readonly LITELLM_DEFAULT_MAX_BUDGET="100"
[[ -z "${LITELLM_DEFAULT_TPM:-}" ]] && readonly LITELLM_DEFAULT_TPM="1000"
[[ -z "${LITELLM_DEFAULT_RPM:-}" ]] && readonly LITELLM_DEFAULT_RPM="100"