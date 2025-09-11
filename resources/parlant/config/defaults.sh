#!/usr/bin/env bash
# Parlant Resource Configuration

# Service configuration
export PARLANT_NAME="parlant"
export PARLANT_DISPLAY_NAME="Parlant LLM Agent Framework"
export PARLANT_PORT="${PARLANT_PORT:-11458}"
export PARLANT_HOST="${PARLANT_HOST:-127.0.0.1}"

# Directory configuration
export PARLANT_DATA_DIR="${PARLANT_DATA_DIR:-${HOME}/.parlant}"
export PARLANT_CONFIG_DIR="${PARLANT_DATA_DIR}/config"
export PARLANT_AGENTS_DIR="${PARLANT_DATA_DIR}/agents"
export PARLANT_LOGS_DIR="${PARLANT_DATA_DIR}/logs"
export PARLANT_WORKSPACE_DIR="${PARLANT_DATA_DIR}/workspace"

# Runtime files
export PARLANT_PID_FILE="${PARLANT_DATA_DIR}/parlant.pid"
export PARLANT_LOG_FILE="${PARLANT_LOGS_DIR}/parlant.log"
export PARLANT_SERVER_FILE="${PARLANT_DATA_DIR}/server.py"

# Python environment
export PARLANT_VENV_DIR="${PARLANT_DATA_DIR}/venv"
export PARLANT_PYTHON="${PARLANT_VENV_DIR}/bin/python"
export PARLANT_PIP="${PARLANT_VENV_DIR}/bin/pip"

# Server configuration
export PARLANT_WORKERS="${PARLANT_WORKERS:-4}"
export PARLANT_MAX_AGENTS="${PARLANT_MAX_AGENTS:-50}"
export PARLANT_SESSION_TIMEOUT="${PARLANT_SESSION_TIMEOUT:-3600}"

# Database configuration (optional, for persistence)
export PARLANT_USE_POSTGRES="${PARLANT_USE_POSTGRES:-false}"
export PARLANT_POSTGRES_HOST="${PARLANT_POSTGRES_HOST:-localhost}"
export PARLANT_POSTGRES_PORT="${PARLANT_POSTGRES_PORT:-25432}"
export PARLANT_POSTGRES_DB="${PARLANT_POSTGRES_DB:-parlant}"
export PARLANT_POSTGRES_USER="${PARLANT_POSTGRES_USER:-parlant}"
export PARLANT_POSTGRES_PASSWORD="${PARLANT_POSTGRES_PASSWORD:-parlant123}"

# Redis configuration (optional, for session state)
export PARLANT_USE_REDIS="${PARLANT_USE_REDIS:-false}"
export PARLANT_REDIS_HOST="${PARLANT_REDIS_HOST:-localhost}"
export PARLANT_REDIS_PORT="${PARLANT_REDIS_PORT:-6379}"

# API configuration
export PARLANT_API_KEY="${PARLANT_API_KEY:-}"
export PARLANT_DEBUG="${PARLANT_DEBUG:-false}"

# Health check
export PARLANT_HEALTH_CHECK_TIMEOUT="${PARLANT_HEALTH_CHECK_TIMEOUT:-5}"
export PARLANT_STARTUP_TIMEOUT="${PARLANT_STARTUP_TIMEOUT:-60}"
export PARLANT_STARTUP_CHECK_INTERVAL="${PARLANT_STARTUP_CHECK_INTERVAL:-2}"