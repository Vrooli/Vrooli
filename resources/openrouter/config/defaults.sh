#!/bin/bash
# OpenRouter configuration defaults

# Service configuration
export OPENROUTER_SERVICE_NAME="openrouter"
export OPENROUTER_SERVICE_CATEGORY="ai"
export OPENROUTER_SERVICE_TYPE="api"
export OPENROUTER_SERVICE_DESCRIPTION="Unified API to many model providers"

# API configuration
export OPENROUTER_API_BASE="https://openrouter.ai/api/v1"
export OPENROUTER_DEFAULT_MODEL="openai/gpt-3.5-turbo"
export OPENROUTER_TIMEOUT="${OPENROUTER_TIMEOUT:-30}"

# Credentials
export OPENROUTER_API_KEY="${OPENROUTER_API_KEY:-}"

# Health check configuration
export OPENROUTER_HEALTH_CHECK_TIMEOUT="${OPENROUTER_HEALTH_CHECK_TIMEOUT:-10}"
export OPENROUTER_HEALTH_CHECK_MODEL="${OPENROUTER_HEALTH_CHECK_MODEL:-openai/gpt-3.5-turbo}"