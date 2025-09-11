#!/bin/bash
# Gemini configuration defaults

# API Configuration
export GEMINI_API_BASE="${GEMINI_API_BASE:-https://generativelanguage.googleapis.com/v1beta}"
export GEMINI_API_KEY="${GEMINI_API_KEY:-}"
export GEMINI_DEFAULT_MODEL="${GEMINI_DEFAULT_MODEL:-gemini-pro}"
export GEMINI_TIMEOUT="${GEMINI_TIMEOUT:-30}"

# Health Check Settings
export GEMINI_HEALTH_CHECK_TIMEOUT="${GEMINI_HEALTH_CHECK_TIMEOUT:-5}"
export GEMINI_HEALTH_CHECK_MODEL="${GEMINI_HEALTH_CHECK_MODEL:-gemini-pro}"

# Rate Limiting
export GEMINI_RATE_LIMIT_RPM="${GEMINI_RATE_LIMIT_RPM:-60}"
export GEMINI_RATE_LIMIT_TPM="${GEMINI_RATE_LIMIT_TPM:-1000000}"

# Cache Configuration (Redis)
export GEMINI_CACHE_ENABLED="${GEMINI_CACHE_ENABLED:-true}"
export GEMINI_CACHE_TTL="${GEMINI_CACHE_TTL:-3600}"  # 1 hour default
export GEMINI_CACHE_PREFIX="${GEMINI_CACHE_PREFIX:-gemini:cache}"
export REDIS_HOST="${REDIS_HOST:-localhost}"
export REDIS_PORT="${REDIS_PORT:-6379}"

# Export configuration
gemini::export_config() {
    export GEMINI_API_BASE
    export GEMINI_API_KEY
    export GEMINI_DEFAULT_MODEL
    export GEMINI_TIMEOUT
    export GEMINI_HEALTH_CHECK_TIMEOUT
    export GEMINI_HEALTH_CHECK_MODEL
    export GEMINI_RATE_LIMIT_RPM
    export GEMINI_RATE_LIMIT_TPM
    export GEMINI_CACHE_ENABLED
    export GEMINI_CACHE_TTL
    export GEMINI_CACHE_PREFIX
    export REDIS_HOST
    export REDIS_PORT
}