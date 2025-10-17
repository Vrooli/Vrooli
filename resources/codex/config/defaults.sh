#!/usr/bin/env bash
# Codex Default Configuration

# Get the directory where this script is located
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
CODEX_CONFIG_DIR="${APP_ROOT}/resources/codex/config"

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

# API configuration (Updated with GPT-5 models - Released August 2025)
export CODEX_API_ENDPOINT="${CODEX_API_ENDPOINT:-https://api.openai.com/v1}"
# Priority: codex-mini-latest (2025 CLI model) > gpt-5-nano > gpt-4o
export CODEX_DEFAULT_MODEL="${CODEX_DEFAULT_MODEL:-gpt-5-nano}"
export CODEX_DEFAULT_TEMPERATURE="${CODEX_DEFAULT_TEMPERATURE:-0.2}"
export CODEX_DEFAULT_MAX_TOKENS="${CODEX_DEFAULT_MAX_TOKENS:-8192}"

# Codex CLI settings (2025 OpenAI agent)
export CODEX_CLI_ENABLED="${CODEX_CLI_ENABLED:-auto}"  # auto|true|false
export CODEX_CLI_MODE="${CODEX_CLI_MODE:-auto}"        # auto|approve|always
export CODEX_WORKSPACE="${CODEX_WORKSPACE:-/tmp/codex-workspace}"
export CODEX_PREFER_MODEL="${CODEX_PREFER_MODEL:-true}" # Prefer codex-mini-latest when available

# Model selection priority (used by smart routing)
export CODEX_MODEL_PRIORITY="${CODEX_MODEL_PRIORITY:-codex-mini-latest,gpt-5-nano,gpt-4o}"

# Runtime configuration
export CODEX_TIMEOUT="${CODEX_TIMEOUT:-30}"
export CODEX_RETRY_COUNT="${CODEX_RETRY_COUNT:-3}"
export CODEX_RETRY_DELAY="${CODEX_RETRY_DELAY:-2}"

# Status file
export CODEX_STATUS_FILE="${CODEX_BASE_DIR}/.status"

# Create necessary directories
mkdir -p "${CODEX_SCRIPTS_DIR}" "${CODEX_OUTPUT_DIR}" "${CODEX_INJECTED_DIR}" 2>/dev/null || true