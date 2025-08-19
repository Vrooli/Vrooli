#!/usr/bin/env bash
# Cline Configuration Defaults

# Resource name
export CLINE_RESOURCE_NAME="cline"
export CLINE_CATEGORY="agents"

# Version and installation
export CLINE_EXTENSION_ID="saoudrizwan.claude-dev"
export CLINE_MIN_NODE_VERSION="18.0.0"

# Paths
export CLINE_HOME="${HOME}/.cline"
export CLINE_CONFIG_DIR="${CLINE_HOME}/config"
export CLINE_EXTENSIONS_DIR="${HOME}/.vscode/extensions"
export CLINE_SETTINGS_FILE="${CLINE_CONFIG_DIR}/settings.json"

# API configuration
export CLINE_DEFAULT_PROVIDER="${CLINE_DEFAULT_PROVIDER:-openrouter}"
export CLINE_MAX_RETRIES=3
export CLINE_TIMEOUT=300

# Model configuration
export CLINE_DEFAULT_MODEL="${CLINE_DEFAULT_MODEL:-anthropic/claude-3.5-sonnet}"
export CLINE_TEMPERATURE="${CLINE_TEMPERATURE:-0.7}"
export CLINE_MAX_TOKENS="${CLINE_MAX_TOKENS:-4096}"

# Operational settings
export CLINE_AUTO_APPROVE="${CLINE_AUTO_APPROVE:-false}"
export CLINE_EXPERIMENTAL_MODE="${CLINE_EXPERIMENTAL_MODE:-false}"

# Health check settings
export CLINE_HEALTH_CHECK_INTERVAL=30
export CLINE_HEALTH_CHECK_TIMEOUT=10

# VS Code settings
export CLINE_VSCODE_COMMAND="${CLINE_VSCODE_COMMAND:-code}"
export CLINE_VSCODE_SERVER_PORT="${CLINE_VSCODE_SERVER_PORT:-}"

# Integration with other resources
export CLINE_USE_OLLAMA="${CLINE_USE_OLLAMA:-false}"
export CLINE_OLLAMA_BASE_URL="${CLINE_OLLAMA_BASE_URL:-http://localhost:11434}"
export CLINE_USE_OPENROUTER="${CLINE_USE_OPENROUTER:-true}"

# Logging
export CLINE_LOG_LEVEL="${CLINE_LOG_LEVEL:-info}"
export CLINE_LOG_FILE="${CLINE_HOME}/cline.log"