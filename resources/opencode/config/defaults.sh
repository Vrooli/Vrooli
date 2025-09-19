#!/bin/bash
# OpenCode Resource Configuration Defaults

# Basic configuration
OPENCODE_RESOURCE_NAME="opencode"
OPENCODE_DISPLAY_NAME="OpenCode AI CLI"
OPENCODE_PORT=3355

# Default provider configuration
OPENCODE_DEFAULT_PROVIDER="ollama"
OPENCODE_DEFAULT_CHAT_MODEL="llama3.2:3b"
OPENCODE_DEFAULT_COMPLETION_MODEL="qwen2.5-coder:3b"

# CLI settings
OPENCODE_CLI_ENTRYPOINT="${APP_ROOT}/resources/opencode/lib/opencode_cli.py"

# Supported providers (used for validation)
OPENCODE_SUPPORTED_PROVIDERS=(
    "ollama"
    "openrouter"
    "cloudflare"
)
