#!/bin/bash
# OpenCode Resource Configuration Defaults

# Basic configuration
OPENCODE_RESOURCE_NAME="opencode"
OPENCODE_DISPLAY_NAME="OpenCode AI CLI"
# Default model configuration (provider/model syntax for official CLI).
# OpenRouter is the wired cloud default; Ollama is auto-selected at
# config-write time when a local daemon is reachable and no OpenRouter key
# is present (see opencode::ensure_config).
OPENCODE_DEFAULT_PROVIDER="openrouter"
OPENCODE_DEFAULT_CHAT_MODEL="x-ai/grok-code-fast-1"
OPENCODE_DEFAULT_COMPLETION_MODEL="x-ai/grok-code-fast-1"

# Default local model used when Ollama is auto-selected.
OPENCODE_OLLAMA_DEFAULT_MODEL="${OPENCODE_OLLAMA_DEFAULT_MODEL:-llama3.1}"
