#!/bin/bash
# OpenRouter configuration defaults

openrouter_xdg_config_home="${XDG_CONFIG_HOME:-${HOME}/.config}"
openrouter_xdg_data_home="${XDG_DATA_HOME:-${HOME}/.local/share}"
openrouter_xdg_cache_home="${XDG_CACHE_HOME:-${HOME}/.cache}"
openrouter_xdg_state_home="${XDG_STATE_HOME:-${HOME}/.local/state}"

# Service configuration
export OPENROUTER_SERVICE_NAME="openrouter"
export OPENROUTER_SERVICE_CATEGORY="ai"
export OPENROUTER_SERVICE_TYPE="api"
export OPENROUTER_SERVICE_DESCRIPTION="Unified API to many model providers"

# API configuration
export OPENROUTER_API_BASE="https://openrouter.ai/api/v1"
export OPENROUTER_DEFAULT_MODEL="openai/gpt-3.5-turbo"
export OPENROUTER_TIMEOUT="${OPENROUTER_TIMEOUT:-30}"

# Canonical resource storage directories
export OPENROUTER_DATA_DIR="${OPENROUTER_DATA_DIR:-${RESOURCE_DATA_DIR:-${openrouter_xdg_data_home}/vrooli/resources/openrouter}}"
export OPENROUTER_CONFIG_DIR="${OPENROUTER_CONFIG_DIR:-${RESOURCE_CONFIG_DIR:-${openrouter_xdg_config_home}/vrooli/resources/openrouter}}"
export OPENROUTER_CACHE_DIR="${OPENROUTER_CACHE_DIR:-${RESOURCE_CACHE_DIR:-${openrouter_xdg_cache_home}/vrooli/resources/openrouter}}"
export OPENROUTER_LOG_DIR="${OPENROUTER_LOG_DIR:-${RESOURCE_LOGS_DIR:-${openrouter_xdg_state_home}/logs/vrooli/resources/openrouter}}"
export OPENROUTER_STATE_DIR="${OPENROUTER_STATE_DIR:-${RESOURCE_STATE_DIR:-${openrouter_xdg_state_home}/vrooli/resources/openrouter}}"
export OPENROUTER_CONTENT_DIR="${OPENROUTER_CONTENT_DIR:-${OPENROUTER_DATA_DIR}/content}"
export OPENROUTER_USAGE_DIR="${OPENROUTER_USAGE_DIR:-${OPENROUTER_DATA_DIR}/usage}"
export OPENROUTER_BENCHMARK_DIR="${OPENROUTER_BENCHMARK_DIR:-${OPENROUTER_DATA_DIR}/benchmarks}"
export OPENROUTER_RATE_LIMIT_DIR="${OPENROUTER_RATE_LIMIT_DIR:-${OPENROUTER_STATE_DIR}/ratelimits}"
export OPENROUTER_CREDENTIALS_FILE="${OPENROUTER_CREDENTIALS_FILE:-${OPENROUTER_CONFIG_DIR}/openrouter-credentials.json}"
export OPENROUTER_CLOUDFLARE_CONFIG_FILE="${OPENROUTER_CLOUDFLARE_CONFIG_FILE:-${OPENROUTER_CONFIG_DIR}/cloudflare-config.json}"
export OPENROUTER_MANUAL_MODELS_FILE="${OPENROUTER_MANUAL_MODELS_FILE:-${OPENROUTER_CONFIG_DIR}/manual-models.json}"
export OPENROUTER_ROUTING_RULES_FILE="${OPENROUTER_ROUTING_RULES_FILE:-${OPENROUTER_CONFIG_DIR}/routing-rules.json}"
export OPENROUTER_ROUTING_HISTORY_FILE="${OPENROUTER_ROUTING_HISTORY_FILE:-${OPENROUTER_STATE_DIR}/routing-history.json}"
export OPENROUTER_TEST_RESULTS_DIR="${OPENROUTER_TEST_RESULTS_DIR:-${OPENROUTER_STATE_DIR}/test-results}"
export OPENROUTER_AGENT_CONFIG_FILE="${OPENROUTER_AGENT_CONFIG_FILE:-${OPENROUTER_CONFIG_DIR}/openrouter-agent-config.json}"

# Credentials
export OPENROUTER_API_KEY="${OPENROUTER_API_KEY:-}"

# Health check configuration
export OPENROUTER_HEALTH_CHECK_TIMEOUT="${OPENROUTER_HEALTH_CHECK_TIMEOUT:-10}"
export OPENROUTER_HEALTH_CHECK_MODEL="${OPENROUTER_HEALTH_CHECK_MODEL:-openai/gpt-3.5-turbo}"
