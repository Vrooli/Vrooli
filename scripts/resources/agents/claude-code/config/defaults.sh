#!/usr/bin/env bash
# Claude Code Configuration and Defaults
# This file contains all constants and default values

# Helper function to detect test environment
is_test_environment() {
    [[ -n "${BATS_TEST_DIRNAME:-}" ]] || [[ "${TEST_MODE:-}" == "true" ]]
}

# Resource configuration
if is_test_environment; then
    RESOURCE_NAME="claude-code"
    MIN_NODE_VERSION="18"
    CLAUDE_PACKAGE="@anthropic-ai/claude-code"
else
    readonly RESOURCE_NAME="claude-code"
    readonly MIN_NODE_VERSION="18"
    readonly CLAUDE_PACKAGE="@anthropic-ai/claude-code"
fi

# MCP configuration constants
if is_test_environment; then
    VROOLI_MCP_SERVER_NAME="vrooli-local"
    VROOLI_DEFAULT_PORT="3000"
    VROOLI_MCP_ENDPOINT="/mcp/sse"
    VROOLI_HEALTH_ENDPOINT="/mcp/health"
else
    readonly VROOLI_MCP_SERVER_NAME="vrooli-local"
    readonly VROOLI_DEFAULT_PORT="3000"
    readonly VROOLI_MCP_ENDPOINT="/mcp/sse"
    readonly VROOLI_HEALTH_ENDPOINT="/mcp/health"
fi

# Default values for CLI arguments
if is_test_environment; then
    DEFAULT_MAX_TURNS="5"
    DEFAULT_TIMEOUT="600"
    DEFAULT_OUTPUT_FORMAT="text"
    DEFAULT_MCP_SCOPE="auto"
else
    readonly DEFAULT_MAX_TURNS="5"
    readonly DEFAULT_TIMEOUT="600"
    readonly DEFAULT_OUTPUT_FORMAT="text"
    readonly DEFAULT_MCP_SCOPE="auto"
fi

# Common directories and paths
if is_test_environment; then
    CLAUDE_CONFIG_DIR="$HOME/.claude"
    CLAUDE_SESSIONS_DIR="$CLAUDE_CONFIG_DIR/sessions"
    CLAUDE_SETTINGS_FILE="$CLAUDE_CONFIG_DIR/settings.json"
    CLAUDE_PROJECT_SETTINGS="$(pwd)/.claude/settings.json"
else
    readonly CLAUDE_CONFIG_DIR="$HOME/.claude"
    readonly CLAUDE_SESSIONS_DIR="$CLAUDE_CONFIG_DIR/sessions"
    readonly CLAUDE_SETTINGS_FILE="$CLAUDE_CONFIG_DIR/settings.json"
    readonly CLAUDE_PROJECT_SETTINGS="$(pwd)/.claude/settings.json"
fi

# Log file locations
if is_test_environment; then
    CLAUDE_LOG_LOCATIONS=(
        "$HOME/.claude/logs"
        "$HOME/.cache/claude/logs"
        "$HOME/Library/Logs/claude"  # macOS
    )
else
    readonly CLAUDE_LOG_LOCATIONS=(
        "$HOME/.claude/logs"
        "$HOME/.cache/claude/logs"
        "$HOME/Library/Logs/claude"  # macOS
    )
fi

#######################################
# Export configuration for use in other scripts
#######################################
claude_code::export_config() {
    export RESOURCE_NAME
    export MIN_NODE_VERSION
    export CLAUDE_PACKAGE
    export VROOLI_MCP_SERVER_NAME
    export VROOLI_DEFAULT_PORT
    export VROOLI_MCP_ENDPOINT
    export VROOLI_HEALTH_ENDPOINT
}

# Export function for subshell availability
export -f claude_code::export_config