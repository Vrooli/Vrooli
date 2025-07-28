#!/usr/bin/env bash
set -euo pipefail

# Claude Code management script
# This script manages the Claude Code CLI installation

DESCRIPTION="Manages Claude Code CLI installation and configuration"

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
RESOURCES_DIR="${SCRIPT_DIR}/../.."

# Source common resources
# shellcheck disable=SC1091
source "${RESOURCES_DIR}/common.sh"
# shellcheck disable=SC1091
source "${RESOURCES_DIR}/../helpers/utils/args.sh"

# Source configuration
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/config/defaults.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/config/messages.sh"

# Export configuration
claude_code::export_config

# Source all library modules
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/common.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/install.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/status.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/execute.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/session.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/settings.sh"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/mcp.sh"

#######################################
# Parse command line arguments
#######################################
claude_code::parse_arguments() {
    args::reset
    
    args::register_help
    args::register_yes
    
    args::register \
        --name "action" \
        --flag "a" \
        --desc "Action to perform" \
        --type "value" \
        --options "install|uninstall|status|info|run|batch|session|settings|logs|register-mcp|unregister-mcp|mcp-status|mcp-test|help" \
        --default "help"
    
    args::register \
        --name "force" \
        --flag "f" \
        --desc "Force action even if already installed" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    args::register \
        --name "prompt" \
        --flag "p" \
        --desc "Prompt to send to Claude" \
        --type "value" \
        --default ""
    
    args::register \
        --name "max-turns" \
        --desc "Maximum number of turns for Claude" \
        --type "value" \
        --default "$DEFAULT_MAX_TURNS"
    
    args::register \
        --name "session-id" \
        --desc "Session ID to resume or manage" \
        --type "value" \
        --default ""
    
    args::register \
        --name "allowed-tools" \
        --desc "Comma-separated list of allowed tools" \
        --type "value" \
        --default ""
    
    args::register \
        --name "timeout" \
        --desc "Timeout in seconds for Claude execution" \
        --type "value" \
        --default "$DEFAULT_TIMEOUT"
    
    args::register \
        --name "output-format" \
        --desc "Output format (stream-json, text)" \
        --type "value" \
        --options "stream-json|text" \
        --default "$DEFAULT_OUTPUT_FORMAT"
    
    args::register \
        --name "dangerously-skip-permissions" \
        --desc "Skip permission checks (USE WITH EXTREME CAUTION)" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    # MCP-specific arguments
    args::register \
        --name "scope" \
        --desc "MCP registration scope (local, project, user, auto)" \
        --type "value" \
        --options "local|project|user|auto" \
        --default "$DEFAULT_MCP_SCOPE"
    
    args::register \
        --name "api-key" \
        --desc "API key for MCP authentication" \
        --type "value" \
        --default ""
    
    args::register \
        --name "server-url" \
        --desc "Vrooli server URL for MCP registration" \
        --type "value" \
        --default ""
    
    args::register \
        --name "format" \
        --desc "Output format for status commands (text, json)" \
        --type "value" \
        --options "text|json" \
        --default "text"
    
    if args::is_asking_for_help "$@"; then
        claude_code::usage
        exit 0
    fi
    
    args::parse "$@"
    
    export ACTION=$(args::get "action")
    export FORCE=$(args::get "force")
    export YES=$(args::get "yes")
    export PROMPT=$(args::get "prompt")
    export MAX_TURNS=$(args::get "max-turns")
    export SESSION_ID=$(args::get "session-id")
    export ALLOWED_TOOLS=$(args::get "allowed-tools")
    export TIMEOUT=$(args::get "timeout")
    export OUTPUT_FORMAT=$(args::get "output-format")
    export SKIP_PERMISSIONS=$(args::get "dangerously-skip-permissions")
    
    # MCP-specific variables
    export MCP_SCOPE=$(args::get "scope")
    export MCP_API_KEY=$(args::get "api-key")
    export MCP_SERVER_URL=$(args::get "server-url")
    export MCP_FORMAT=$(args::get "format")
}

#######################################
# Main execution function
#######################################
claude_code::main() {
    claude_code::parse_arguments "$@"
    
    case "$ACTION" in
        "install")
            claude_code::install
            ;;
        "uninstall")
            claude_code::uninstall
            ;;
        "status")
            claude_code::status
            ;;
        "info")
            claude_code::info
            ;;
        "run")
            claude_code::run
            ;;
        "batch")
            claude_code::batch
            ;;
        "session")
            claude_code::session
            ;;
        "settings")
            claude_code::settings
            ;;
        "logs")
            claude_code::logs
            ;;
        "register-mcp")
            claude_code::register_mcp
            ;;
        "unregister-mcp")
            claude_code::unregister_mcp
            ;;
        "mcp-status")
            claude_code::mcp_status
            ;;
        "mcp-test")
            claude_code::mcp_test
            ;;
        "help"|*)
            claude_code::usage
            ;;
    esac
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    claude_code::main "$@"
fi