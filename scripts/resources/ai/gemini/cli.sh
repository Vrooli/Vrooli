#!/bin/bash
# Gemini CLI interface

# Get script directory (resolving symlinks for installed CLI)
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    GEMINI_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
else
    GEMINI_CLI_SCRIPT="${BASH_SOURCE[0]}"
fi
GEMINI_CLI_DIR="$(cd "$(dirname "$GEMINI_CLI_SCRIPT")" && pwd)"

# Source dependencies
source "${GEMINI_CLI_DIR}/lib/core.sh"
source "${GEMINI_CLI_DIR}/lib/status.sh"
source "${GEMINI_CLI_DIR}/lib/install.sh"
source "${GEMINI_CLI_DIR}/lib/inject.sh"

# Main CLI handler
gemini::cli() {
    local command="${1:-help}"
    shift
    
    case "$command" in
        status)
            # Parse status arguments
            local verbose="false"
            local format="text"
            while [[ $# -gt 0 ]]; do
                case "$1" in
                    --verbose)
                        verbose="true"
                        shift
                        ;;
                    --format)
                        format="${2:-text}"
                        shift 2
                        ;;
                    *)
                        shift
                        ;;
                esac
            done
            gemini::status "$verbose" "$format"
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
            gemini::inject "$@"
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
  inject <target> [data]                   Inject config to other resources
  help                                      Show this help message

Examples:
  $0 status
  $0 install --verbose
  $0 list-models
  $0 generate "Explain quantum computing"
  $0 inject n8n
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