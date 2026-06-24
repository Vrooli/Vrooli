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
# balanced local coding model). The Go config writer resolves the local model
# and its sampling from the resource-ollama SSOT (`policy resolve --role
# code.local`); tool-calling capability is validated by `resource-ollama models
# doctor`, not by a bash probe.
OPENCODE_OLLAMA_DEFAULT_MODEL="${OPENCODE_OLLAMA_DEFAULT_MODEL:-gemma4:12b}"

# Per-model context window (num_ctx) for the local Ollama coding agent. Opencode
# injects a large system prompt (~7.5k tokens), so the Ollama default of 4096
# overflows before a thinking model (gemma4/qwen3) can act — every run dies
# finish_reason=length with output:1. This value is written into opencode.json
# per-model as provider.ollama.models[<model>].options.options.num_ctx and
# mirrored into limit.{context,output}, so the local coder has room to reason +
# tool-call. It layers ABOVE the global OLLAMA_CONTEXT_LENGTH daemon default.
OPENCODE_OLLAMA_NUM_CTX="${OPENCODE_OLLAMA_NUM_CTX:-16384}"
