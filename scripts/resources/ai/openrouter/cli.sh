#!/bin/bash
# OpenRouter CLI interface

# Get script directory
OPENROUTER_CLI_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source dependencies
source "${OPENROUTER_CLI_DIR}/lib/core.sh"
source "${OPENROUTER_CLI_DIR}/lib/status.sh"
source "${OPENROUTER_CLI_DIR}/lib/install.sh"
source "${OPENROUTER_CLI_DIR}/lib/inject.sh"

# Main CLI handler
openrouter::cli() {
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
            openrouter::status "$verbose" "$format"
            ;;
        install)
            openrouter::install "$@"
            ;;
        uninstall)
            openrouter::uninstall "$@"
            ;;
        start)
            # OpenRouter is an API service, always "running"
            echo "OpenRouter is an API service (no start needed)"
            return 0
            ;;
        stop)
            # OpenRouter is an API service, can't be stopped
            echo "OpenRouter is an API service (cannot be stopped)"
            return 0
            ;;
        test|test-connection)
            if openrouter::test_connection; then
                echo "✓ OpenRouter API is accessible"
                return 0
            else
                echo "✗ Failed to connect to OpenRouter API"
                return 1
            fi
            ;;
        list-models)
            openrouter::list_models
            ;;
        usage|credits)
            openrouter::get_usage | jq '.' 2>/dev/null || openrouter::get_usage
            ;;
        inject)
            openrouter::inject "$@"
            ;;
        help|--help|-h)
            cat <<EOF
OpenRouter Resource CLI

Usage: $0 <command> [options]

Commands:
  status [--verbose] [--format json|text]  Check OpenRouter status
  install [--verbose]                      Install/configure OpenRouter
  uninstall [--verbose]                    Remove OpenRouter configuration
  start                                     No-op (API service)
  stop                                      No-op (API service)
  test|test-connection                     Test API connectivity
  list-models                               List available models
  usage|credits                             Show usage and credits
  inject <target> [data]                   Inject config to other resources
  help                                      Show this help message

Examples:
  $0 status
  $0 install --verbose
  $0 list-models
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
    openrouter::cli "$@"
fi