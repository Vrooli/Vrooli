#!/usr/bin/env bash
# Claude Code Configuration and Defaults
# This file contains all constants and default values

# Source guard to prevent multiple sourcing
[[ -n "${_CLAUDE_CODE_DEFAULTS_SOURCED:-}" ]] && return 0
_CLAUDE_CODE_DEFAULTS_SOURCED=1

# Helper function to detect test environment
claude_code::is_test_environment() {
    [[ -n "${BATS_TEST_DIRNAME:-}" ]] || [[ "${TEST_MODE:-}" == "true" ]]
}

# Resource configuration
if claude_code::is_test_environment; then
    RESOURCE_NAME="claude-code"
    MIN_NODE_VERSION="18"
    CLAUDE_PACKAGE="@anthropic-ai/claude-code"
else
    readonly RESOURCE_NAME="claude-code"
    readonly MIN_NODE_VERSION="18"
    readonly CLAUDE_PACKAGE="@anthropic-ai/claude-code"
fi

# MCP configuration constants
if claude_code::is_test_environment; then
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
if claude_code::is_test_environment; then
    DEFAULT_MAX_TURNS="5"
    DEFAULT_TIMEOUT="600"
    DEFAULT_OUTPUT_FORMAT="text"
    DEFAULT_MCP_SCOPE="auto"
    # Rate limit constants (estimates based on subscription tiers)
    CLAUDE_FREE_DAILY_LIMIT="50"
    CLAUDE_PRO_DAILY_LIMIT="1000"
    CLAUDE_MAX_DAILY_LIMIT="5000"
    CLAUDE_RATE_LIMIT_RESET_HOURS="5"  # Claude resets every 5 hours
    CLAUDE_WEEKLY_RESET_DAYS="7"       # Weekly limits for Claude Code
else
    readonly DEFAULT_MAX_TURNS="5"
    readonly DEFAULT_TIMEOUT="600"
    readonly DEFAULT_OUTPUT_FORMAT="text"
    readonly DEFAULT_MCP_SCOPE="auto"
    # Rate limit constants (estimates based on subscription tiers)
    readonly CLAUDE_FREE_DAILY_LIMIT="50"
    readonly CLAUDE_PRO_DAILY_LIMIT="1000"
    readonly CLAUDE_MAX_DAILY_LIMIT="5000"
    readonly CLAUDE_RATE_LIMIT_RESET_HOURS="5"  # Claude resets every 5 hours
    readonly CLAUDE_WEEKLY_RESET_DAYS="7"       # Weekly limits for Claude Code
fi

# Common directories and paths
if claude_code::is_test_environment; then
    CLAUDE_CONFIG_DIR="$HOME/.claude"
    CLAUDE_SESSIONS_DIR="$CLAUDE_CONFIG_DIR/sessions"
    CLAUDE_SETTINGS_FILE="$CLAUDE_CONFIG_DIR/settings.json"
    CLAUDE_PROJECT_SETTINGS="$(pwd)/.claude/settings.json"
    CLAUDE_USAGE_FILE="$CLAUDE_CONFIG_DIR/usage_tracking.json"
else
    readonly CLAUDE_CONFIG_DIR="$HOME/.claude"
    readonly CLAUDE_SESSIONS_DIR="$CLAUDE_CONFIG_DIR/sessions"
    readonly CLAUDE_SETTINGS_FILE="$CLAUDE_CONFIG_DIR/settings.json"
    readonly CLAUDE_PROJECT_SETTINGS="$(pwd)/.claude/settings.json"
    readonly CLAUDE_USAGE_FILE="$CLAUDE_CONFIG_DIR/usage_tracking.json"
fi

# Log file locations
if claude_code::is_test_environment; then
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