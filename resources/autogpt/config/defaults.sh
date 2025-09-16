#!/usr/bin/env bash
################################################################################
# AutoGPT Default Configuration
# Defines all configuration variables with sensible defaults
################################################################################

# Container configuration
export AUTOGPT_CONTAINER_NAME="${AUTOGPT_CONTAINER_NAME:-autogpt-main}"
export AUTOGPT_IMAGE="${AUTOGPT_IMAGE:-significantgravitas/auto-gpt:latest}"

# Port allocation - Use a unique port for AutoGPT
export AUTOGPT_PORT_API="${AUTOGPT_PORT_API:-8501}"

# Data directories
export AUTOGPT_DATA_DIR="${AUTOGPT_DATA_DIR:-${APP_ROOT}/data/resources/autogpt}"
export AUTOGPT_AGENTS_DIR="${AUTOGPT_AGENTS_DIR:-${AUTOGPT_DATA_DIR}/agents}"
export AUTOGPT_TOOLS_DIR="${AUTOGPT_TOOLS_DIR:-${AUTOGPT_DATA_DIR}/tools}"
export AUTOGPT_WORKSPACE_DIR="${AUTOGPT_WORKSPACE_DIR:-${AUTOGPT_DATA_DIR}/workspace}"
export AUTOGPT_LOGS_DIR="${AUTOGPT_LOGS_DIR:-${AUTOGPT_DATA_DIR}/logs}"

# LLM Configuration
export AUTOGPT_AI_PROVIDER="${AUTOGPT_AI_PROVIDER:-none}"  # openrouter, ollama, openai, none
export AUTOGPT_MODEL="${AUTOGPT_MODEL:-gpt-3.5-turbo}"
export AUTOGPT_MAX_ITERATIONS="${AUTOGPT_MAX_ITERATIONS:-25}"
export AUTOGPT_TEMPERATURE="${AUTOGPT_TEMPERATURE:-0.7}"

# Memory Configuration
export AUTOGPT_MEMORY_BACKEND="${AUTOGPT_MEMORY_BACKEND:-local}"  # redis, postgres, local
export AUTOGPT_REDIS_HOST="${AUTOGPT_REDIS_HOST:-localhost}"
export AUTOGPT_REDIS_PORT="${AUTOGPT_REDIS_PORT:-6379}"
export AUTOGPT_POSTGRES_HOST="${AUTOGPT_POSTGRES_HOST:-localhost}"
export AUTOGPT_POSTGRES_PORT="${AUTOGPT_POSTGRES_PORT:-5432}"
export AUTOGPT_POSTGRES_DB="${AUTOGPT_POSTGRES_DB:-autogpt}"

# Tool Integration
export AUTOGPT_ENABLE_BROWSER="${AUTOGPT_ENABLE_BROWSER:-false}"
export AUTOGPT_BROWSERLESS_URL="${AUTOGPT_BROWSERLESS_URL:-http://localhost:3000}"
export AUTOGPT_ENABLE_CODE_EXEC="${AUTOGPT_ENABLE_CODE_EXEC:-false}"
export AUTOGPT_JUDGE0_URL="${AUTOGPT_JUDGE0_URL:-http://localhost:2358}"

# Performance Settings
export AUTOGPT_WORKER_THREADS="${AUTOGPT_WORKER_THREADS:-4}"
export AUTOGPT_TASK_TIMEOUT="${AUTOGPT_TASK_TIMEOUT:-300}"  # seconds
export AUTOGPT_MAX_MEMORY="${AUTOGPT_MAX_MEMORY:-2048}"     # MB

# Security Settings
export AUTOGPT_ALLOW_FILE_ACCESS="${AUTOGPT_ALLOW_FILE_ACCESS:-restricted}"  # full, restricted, none
export AUTOGPT_ALLOWED_DOMAINS="${AUTOGPT_ALLOWED_DOMAINS:-}"  # comma-separated list
export AUTOGPT_API_KEY="${AUTOGPT_API_KEY:-}"  # Optional API key for AutoGPT API

# Logging
export AUTOGPT_LOG_LEVEL="${AUTOGPT_LOG_LEVEL:-info}"  # debug, info, warning, error
export AUTOGPT_LOG_FORMAT="${AUTOGPT_LOG_FORMAT:-json}"  # json, text

# Development Settings
export AUTOGPT_DEBUG="${AUTOGPT_DEBUG:-false}"
export AUTOGPT_DEV_MODE="${AUTOGPT_DEV_MODE:-false}"