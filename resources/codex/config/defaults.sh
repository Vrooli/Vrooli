#!/usr/bin/env bash
# Codex Default Configuration

# Get the directory where this script is located
CODEX_CONFIG_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# Default values
export CODEX_NAME="codex"
export CODEX_CATEGORY="ai"
export CODEX_DESCRIPTION="AI-powered code completion and generation via OpenAI Codex"

# Configuration paths
export CODEX_BASE_DIR="${CODEX_CONFIG_DIR}/.."
export CODEX_LIB_DIR="${CODEX_BASE_DIR}/lib"
export CODEX_INJECTED_DIR="${CODEX_BASE_DIR}/injected"
export CODEX_SCRIPTS_DIR="${HOME}/.codex/scripts"
export CODEX_OUTPUT_DIR="${HOME}/.codex/outputs"

# API configuration
export CODEX_API_ENDPOINT="${CODEX_API_ENDPOINT:-https://api.openai.com/v1}"
export CODEX_DEFAULT_MODEL="${CODEX_DEFAULT_MODEL:-code-davinci-002}"
export CODEX_DEFAULT_TEMPERATURE="${CODEX_DEFAULT_TEMPERATURE:-0.0}"
export CODEX_DEFAULT_MAX_TOKENS="${CODEX_DEFAULT_MAX_TOKENS:-2048}"

# Runtime configuration
export CODEX_TIMEOUT="${CODEX_TIMEOUT:-30}"
export CODEX_RETRY_COUNT="${CODEX_RETRY_COUNT:-3}"
export CODEX_RETRY_DELAY="${CODEX_RETRY_DELAY:-2}"

# Status file
export CODEX_STATUS_FILE="${CODEX_BASE_DIR}/.status"

# Create necessary directories
mkdir -p "${CODEX_SCRIPTS_DIR}" "${CODEX_OUTPUT_DIR}" "${CODEX_INJECTED_DIR}" 2>/dev/null || true