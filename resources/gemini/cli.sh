#!/bin/bash
# Gemini CLI interface

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    GEMINI_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    # Recalculate APP_ROOT from resolved symlink location
    APP_ROOT="$(builtin cd "$(dirname "$GEMINI_CLI_SCRIPT")/../.." && builtin pwd)"
fi
GEMINI_CLI_DIR="${APP_ROOT}/resources/gemini"

# Source dependencies
source "${GEMINI_CLI_DIR}/lib/core.sh"
source "${GEMINI_CLI_DIR}/lib/status.sh"
source "${GEMINI_CLI_DIR}/lib/install.sh"
source "${GEMINI_CLI_DIR}/lib/inject.sh"
source "${GEMINI_CLI_DIR}/lib/content.sh"
source "${APP_ROOT}/scripts/lib/utils/log.sh"

# Main CLI handler
gemini::cli() {
    local command="${1:-help}"
    shift
    
    case "$command" in
        status)
            # Pass all arguments directly to status function
            gemini::status "$@"
            ;;
        install)
            local verbose="false"
            if [[ "${1:-}" == "--verbose" ]]; then
                verbose="true"
            fi
            gemini::install "$verbose"
            ;;
        uninstall)
            local verbose="false"
            if [[ "${1:-}" == "--verbose" ]]; then
                verbose="true"
            fi
            gemini::uninstall "$verbose"
            ;;
        start)
            # Gemini is an API service, always "running"
            echo "Gemini is an API service (no start needed)"
            return 0
            ;;
        stop)
            # Gemini is an API service, can't be stopped
            echo "Gemini is an API service (cannot be stopped)"
            return 0
            ;;
        test|test-connection)
            if gemini::test_connection; then
                echo "✓ Gemini API is accessible"
                return 0
            else
                echo "✗ Failed to connect to Gemini API"
                return 1
            fi
            ;;
        list-models)
            gemini::list_models
            ;;
        generate)
            if [[ -z "$1" ]]; then
                echo "Usage: $0 generate <prompt> [model]"
                return 1
            fi
            gemini::generate "$@"
            ;;
        inject)
            # Backwards compatibility - redirect to content for prompts/templates
            log::warn "inject is deprecated, use 'content' instead"
            gemini::inject "$@"
            ;;
        content)
            gemini::content "$@"
            ;;
        help|--help|-h)
            cat <<EOF
Gemini Resource CLI

Usage: $0 <command> [options]

Commands:
  status [--verbose] [--format json|text]  Check Gemini status
  install [--verbose]                      Install/configure Gemini
  uninstall [--verbose]                    Remove Gemini configuration
  start                                     No-op (API service)
  stop                                      No-op (API service)
  test|test-connection                     Test API connectivity
  list-models                               List available models
  generate <prompt> [model]                Generate content
  content <action> [options]               Manage prompts/templates/functions
  inject <target> [data]                   [DEPRECATED] Use 'content' instead
  help                                      Show this help message

Content Management:
  content add <name> [type] [file|-]       Add prompt/template/function
  content list [type] [format]             List stored content
  content get <name> [type]                Get stored content
  content remove <name> [type]             Remove stored content
  content execute <name> [model] [params]  Execute stored content

Examples:
  $0 status
  $0 install --verbose
  $0 list-models
  $0 generate "Explain quantum computing"
  $0 content add my-prompt prompt -
  $0 content list
  $0 content execute my-prompt
EOF
            ;;
        *)
            echo "Unknown command: $command"
            echo "Run '$0 help' for usage information"
            return 1
            ;;
    esac
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    set +e  # Disable strict error handling for CLI
    gemini::cli "$@"
fi