#!/usr/bin/env bash
################################################################################
# Haystack Default Configuration - v2.0 Universal Contract Compliant
# 
# Default settings for Haystack resource
# Can be overridden by environment variables
################################################################################

# Service configuration
export HAYSTACK_PORT="${HAYSTACK_PORT:-$(resources::get_port "haystack" 2>/dev/null || echo "8075")}"
export HAYSTACK_HOST="${HAYSTACK_HOST:-0.0.0.0}"
export HAYSTACK_WORKERS="${HAYSTACK_WORKERS:-2}"

# Python environment
export HAYSTACK_PYTHON="${HAYSTACK_PYTHON:-python3}"
export HAYSTACK_VENV_NAME="${HAYSTACK_VENV_NAME:-haystack-venv}"

# Model configuration
export HAYSTACK_EMBEDDING_MODEL="${HAYSTACK_EMBEDDING_MODEL:-sentence-transformers/all-MiniLM-L6-v2}"
export HAYSTACK_EMBEDDING_DIM="${HAYSTACK_EMBEDDING_DIM:-384}"
export HAYSTACK_MAX_SEQ_LENGTH="${HAYSTACK_MAX_SEQ_LENGTH:-512}"

# Document store configuration
export HAYSTACK_DOCUMENT_STORE="${HAYSTACK_DOCUMENT_STORE:-inmemory}"  # inmemory, qdrant, elasticsearch
export HAYSTACK_SIMILARITY="${HAYSTACK_SIMILARITY:-cosine}"  # cosine, dot_product, l2
export HAYSTACK_INDEX_NAME="${HAYSTACK_INDEX_NAME:-documents}"

# Query configuration
export HAYSTACK_TOP_K="${HAYSTACK_TOP_K:-10}"
export HAYSTACK_MAX_QUERY_LENGTH="${HAYSTACK_MAX_QUERY_LENGTH:-200}"

# Performance settings
export HAYSTACK_BATCH_SIZE="${HAYSTACK_BATCH_SIZE:-32}"
export HAYSTACK_USE_GPU="${HAYSTACK_USE_GPU:-false}"
export HAYSTACK_CACHE_DIR="${HAYSTACK_CACHE_DIR:-${HOME}/.cache/haystack}"

# Logging
export HAYSTACK_LOG_LEVEL="${HAYSTACK_LOG_LEVEL:-INFO}"
export HAYSTACK_LOG_FORMAT="${HAYSTACK_LOG_FORMAT:-json}"

# Timeouts
export HAYSTACK_STARTUP_TIMEOUT="${HAYSTACK_STARTUP_TIMEOUT:-30}"
export HAYSTACK_SHUTDOWN_TIMEOUT="${HAYSTACK_SHUTDOWN_TIMEOUT:-10}"
export HAYSTACK_REQUEST_TIMEOUT="${HAYSTACK_REQUEST_TIMEOUT:-30}"

# Resource limits
export HAYSTACK_MAX_MEMORY="${HAYSTACK_MAX_MEMORY:-2G}"
export HAYSTACK_MAX_DOCUMENTS="${HAYSTACK_MAX_DOCUMENTS:-100000}"
export HAYSTACK_MAX_UPLOAD_SIZE="${HAYSTACK_MAX_UPLOAD_SIZE:-10M}"

# Integration settings
export HAYSTACK_ENABLE_OLLAMA="${HAYSTACK_ENABLE_OLLAMA:-true}"
export HAYSTACK_ENABLE_QDRANT="${HAYSTACK_ENABLE_QDRANT:-true}"
export HAYSTACK_ENABLE_UNSTRUCTURED="${HAYSTACK_ENABLE_UNSTRUCTURED:-true}"

# API settings
export HAYSTACK_API_KEY="${HAYSTACK_API_KEY:-}"  # Optional API key for authentication
export HAYSTACK_CORS_ORIGINS="${HAYSTACK_CORS_ORIGINS:-*}"
export HAYSTACK_MAX_CONNECTIONS="${HAYSTACK_MAX_CONNECTIONS:-100}"