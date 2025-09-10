#!/usr/bin/env bash
# QwenCoder Default Configuration

# Service configuration
# Source port registry for proper port allocation
if [[ -f "/home/matthalloran8/Vrooli/scripts/resources/port_registry.sh" ]]; then
    source "/home/matthalloran8/Vrooli/scripts/resources/port_registry.sh"
    export QWENCODER_PORT="${RESOURCE_PORTS[qwencoder]:-11452}"
else
    export QWENCODER_PORT="${QWENCODER_PORT:-11452}"
fi
export QWENCODER_HOST="${QWENCODER_HOST:-0.0.0.0}"

# Model configuration
export QWENCODER_MODEL="${QWENCODER_MODEL:-qwencoder-1.5b}"
export QWENCODER_DEVICE="${QWENCODER_DEVICE:-auto}"  # auto, cpu, cuda
export QWENCODER_MAX_MEMORY="${QWENCODER_MAX_MEMORY:-16GB}"
export QWENCODER_QUANTIZE="${QWENCODER_QUANTIZE:-none}"  # none, 8bit, 4bit

# Performance configuration
export QWENCODER_BATCH_SIZE="${QWENCODER_BATCH_SIZE:-1}"
export QWENCODER_MAX_TOKENS="${QWENCODER_MAX_TOKENS:-2048}"
export QWENCODER_CONTEXT_LENGTH="${QWENCODER_CONTEXT_LENGTH:-262144}"  # 256K tokens
export QWENCODER_NUM_WORKERS="${QWENCODER_NUM_WORKERS:-4}"

# API configuration
export QWENCODER_API_TIMEOUT="${QWENCODER_API_TIMEOUT:-60}"
export QWENCODER_API_MAX_REQUESTS="${QWENCODER_API_MAX_REQUESTS:-100}"
export QWENCODER_API_RATE_LIMIT="${QWENCODER_API_RATE_LIMIT:-10}"  # requests per second

# Model idle configuration
export QWENCODER_IDLE_TIMEOUT="${QWENCODER_IDLE_TIMEOUT:-600}"  # 10 minutes
export QWENCODER_PRELOAD_MODELS="${QWENCODER_PRELOAD_MODELS:-qwencoder-1.5b}"

# Logging configuration
export QWENCODER_LOG_LEVEL="${QWENCODER_LOG_LEVEL:-INFO}"
export QWENCODER_LOG_FORMAT="${QWENCODER_LOG_FORMAT:-json}"

# Health check configuration
export QWENCODER_HEALTH_CHECK_INTERVAL="${QWENCODER_HEALTH_CHECK_INTERVAL:-30}"
export QWENCODER_HEALTH_CHECK_TIMEOUT="${QWENCODER_HEALTH_CHECK_TIMEOUT:-5}"

# Model repository
export QWENCODER_MODEL_REPO="${QWENCODER_MODEL_REPO:-https://huggingface.co/Qwen}"
export QWENCODER_CACHE_DIR="${QWENCODER_CACHE_DIR:-${HOME}/.cache/qwencoder}"