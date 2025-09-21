#!/bin/bash
# OpenCode Resource Configuration Defaults

# Basic configuration
OPENCODE_RESOURCE_NAME="opencode"
OPENCODE_DISPLAY_NAME="OpenCode AI CLI"
# Default model configuration (provider/model syntax for official CLI)
OPENCODE_DEFAULT_PROVIDER="openrouter"
OPENCODE_DEFAULT_CHAT_MODEL="qwen3-coder"
OPENCODE_DEFAULT_COMPLETION_MODEL="qwen3-coder"

# Supported providers (used for validation)
OPENCODE_SUPPORTED_PROVIDERS=(
    "ollama"
    "openrouter"
    "cloudflare"
)
