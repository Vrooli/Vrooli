#!/bin/bash
# OpenCode Resource Configuration Defaults

# Basic configuration
OPENCODE_RESOURCE_NAME="opencode"
OPENCODE_DISPLAY_NAME="OpenCode VS Code AI Extension"
OPENCODE_EXTENSION_ID="rjmacarthy.twinny"
OPENCODE_PORT=3355

# Data directories (inherit from common.sh)
# OPENCODE_DATA_DIR is defined in lib/common.sh
# OPENCODE_CONFIG_FILE is defined in lib/common.sh

# Default models and configuration
DEFAULT_MODEL_PROVIDER="ollama"
DEFAULT_CHAT_MODEL="llama3.2:3b"
DEFAULT_COMPLETION_MODEL="qwen2.5-coder:3b"

# Installation settings
REQUIRE_VSCODE=true
EXTENSION_AUTO_INSTALL=true