#!/bin/bash
# OpenCode Resource Configuration Defaults

# Basic configuration
OPENCODE_RESOURCE_NAME="opencode"
OPENCODE_DISPLAY_NAME="OpenCode AI CLI"
# Default model configuration (provider/model syntax for official CLI).
# OpenRouter is the wired cloud default; Ollama is auto-selected at
# config-write time when a local daemon is reachable and no OpenRouter key
# is present (see opencode::ensure_config).
#
# Cloud default: deepseek-v4-flash — cheap, fast, and (verified) returns
# structured tool calls through OpenRouter. Replaced x-ai/grok-code-fast-1,
# which was delisted from OpenRouter (every run failed ProviderModelNotFound).
OPENCODE_DEFAULT_PROVIDER="openrouter"
OPENCODE_DEFAULT_CHAT_MODEL="deepseek/deepseek-v4-flash"
OPENCODE_DEFAULT_COMPLETION_MODEL="deepseek/deepseek-v4-flash"

# Default local model used when Ollama is auto-selected. Aligned with the
# `code.local` role in resources/ollama/model-policy.json (gemma4:12b — the
# balanced local coding model). pick_model still probes tool-calling support
# at config-write time and falls through its preference list if this model is
# absent or does not return structured tool_calls (see opencode::ollama::pick_model).
OPENCODE_OLLAMA_DEFAULT_MODEL="${OPENCODE_OLLAMA_DEFAULT_MODEL:-gemma4:12b}"
