#!/bin/bash
# OpenCode Resource Configuration Defaults

# Basic configuration
OPENCODE_RESOURCE_NAME="opencode"
OPENCODE_DISPLAY_NAME="OpenCode AI CLI"
# Default model configuration (provider/model syntax for official CLI)
OPENCODE_DEFAULT_PROVIDER="ollama"
OPENCODE_DEFAULT_CHAT_MODEL="llama3.2:3b"
OPENCODE_DEFAULT_COMPLETION_MODEL="qwen2.5-coder:3b"

# Supported providers (used for validation)
OPENCODE_SUPPORTED_PROVIDERS=(
    "ollama"
    "openrouter"
    "cloudflare"
)
