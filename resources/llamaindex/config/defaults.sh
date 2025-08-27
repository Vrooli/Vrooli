#!/bin/bash

# LlamaIndex Configuration Defaults
# These settings can be overridden by environment variables

# Service Configuration
export LLAMAINDEX_PORT="${LLAMAINDEX_PORT:-8091}"
export LLAMAINDEX_VERSION="${LLAMAINDEX_VERSION:-0.11.14}"
export LLAMAINDEX_PYTHON_VERSION="${LLAMAINDEX_PYTHON_VERSION:-3.11}"

# Directory Configuration  
export LLAMAINDEX_VENV_DIR="${LLAMAINDEX_VENV_DIR:-${HOME}/.vrooli/llamaindex/venv}"
export LLAMAINDEX_DATA_DIR="${LLAMAINDEX_DATA_DIR:-${HOME}/.vrooli/llamaindex/data}"
export LLAMAINDEX_CONFIG_DIR="${LLAMAINDEX_CONFIG_DIR:-${HOME}/.vrooli/llamaindex/config}"

# Docker Configuration
export LLAMAINDEX_CONTAINER_NAME="${LLAMAINDEX_CONTAINER_NAME:-llamaindex}"
export LLAMAINDEX_IMAGE_NAME="${LLAMAINDEX_IMAGE_NAME:-llamaindex:latest}"

# AI Configuration
export LLAMAINDEX_USE_OLLAMA="${LLAMAINDEX_USE_OLLAMA:-true}"
export LLAMAINDEX_DEFAULT_EMBED_MODEL="${LLAMAINDEX_DEFAULT_EMBED_MODEL:-nomic-embed-text:latest}"
export LLAMAINDEX_DEFAULT_LLM_MODEL="${LLAMAINDEX_DEFAULT_LLM_MODEL:-llama3.2:3b}"